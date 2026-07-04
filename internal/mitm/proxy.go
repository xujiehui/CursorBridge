package mitm

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"cursorbridge/internal/apperrors"
	"cursorbridge/internal/certs"
	"cursorbridge/internal/relay"
)

type Proxy struct {
	addr    string
	gateway *relay.Gateway
	certs   *certs.Manager

	mu     sync.Mutex
	server *http.Server
}

type Status struct {
	Addr    string `json:"addr"`
	Running bool   `json:"running"`
	MITM    bool   `json:"mitm"`
}

func New(addr string, gateway *relay.Gateway, certManager *certs.Manager) *Proxy {
	return &Proxy{addr: addr, gateway: gateway, certs: certManager}
}

func (p *Proxy) SetAddr(addr string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.server != nil {
		return apperrors.New(apperrors.ErrInvalidSystemSetting, "proxy must be stopped before changing its listen address", http.StatusConflict)
	}
	p.addr = addr
	return nil
}

func (p *Proxy) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.server != nil {
		return nil
	}

	server := &http.Server{
		Addr:              p.addr,
		Handler:           p,
		ReadHeaderTimeout: 10 * time.Second,
	}
	ln, err := net.Listen("tcp", p.addr)
	if err != nil {
		return err
	}
	p.server = server

	go func() {
		slog.Info("local proxy listening", "addr", p.addr)
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("proxy server stopped", "error", err)
		}
		p.mu.Lock()
		if p.server == server {
			p.server = nil
		}
		p.mu.Unlock()
	}()
	return nil
}

func (p *Proxy) Stop(ctx context.Context) error {
	p.mu.Lock()
	server := p.server
	p.mu.Unlock()
	if server == nil {
		return nil
	}
	err := server.Shutdown(ctx)
	p.mu.Lock()
	if p.server == server {
		p.server = nil
	}
	p.mu.Unlock()
	return err
}

func (p *Proxy) Status() Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	return Status{Addr: p.addr, Running: p.server != nil, MITM: p.certs != nil}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleTunnel(w, r)
		return
	}
	if err := p.gateway.ServeHTTP(w, r); err != nil {
		writeProxyError(w, err)
	}
}

func (p *Proxy) handleTunnel(w http.ResponseWriter, r *http.Request) {
	if p.certs != nil {
		p.handleMITM(w, r)
		return
	}
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		writeProxyError(w, apperrors.Wrap(apperrors.ErrUpstream, "connect upstream", http.StatusBadGateway, err))
		return
	}
	defer destConn.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		writeProxyError(w, apperrors.New(apperrors.ErrInvalidSystemSetting, "hijacking is not supported", http.StatusInternalServerError))
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		writeProxyError(w, apperrors.Wrap(apperrors.ErrInvalidSystemSetting, "hijack client connection", http.StatusInternalServerError, err))
		return
	}
	defer clientConn.Close()

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		return
	}

	errCh := make(chan error, 2)
	go copyTunnel(errCh, destConn, clientConn)
	go copyTunnel(errCh, clientConn, destConn)
	<-errCh
}

func (p *Proxy) handleMITM(w http.ResponseWriter, r *http.Request) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		writeProxyError(w, apperrors.New(apperrors.ErrInvalidSystemSetting, "hijacking is not supported", http.StatusInternalServerError))
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		writeProxyError(w, apperrors.Wrap(apperrors.ErrInvalidSystemSetting, "hijack client connection", http.StatusInternalServerError, err))
		return
	}
	defer clientConn.Close()

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		return
	}

	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	certPEM, keyPEM, err := p.certs.ServerCertificate(host)
	if err != nil {
		slog.Error("generate MITM certificate", "host", host, "error", err)
		return
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		slog.Error("parse MITM certificate", "host", host, "error", err)
		return
	}

	tlsConn := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})
	if err := tlsConn.Handshake(); err != nil {
		slog.Debug("MITM TLS handshake failed", "host", host, "error", err)
		return
	}

	server := &http.Server{
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			req.URL.Scheme = "https"
			req.URL.Host = r.Host
			req.RequestURI = ""
			req.Host = r.Host
			if err := p.gateway.ServeHTTP(resp, req); err != nil {
				writeProxyError(resp, err)
			}
		}),
	}
	_ = server.Serve(&singleConnListener{Conn: tlsConn})
}

func copyTunnel(errCh chan<- error, dst io.Writer, src io.Reader) {
	_, err := io.Copy(dst, src)
	errCh <- err
}

type singleConnListener struct {
	net.Conn
	used bool
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	if l.used {
		return nil, net.ErrClosed
	}
	l.used = true
	return l.Conn, nil
}

func (l *singleConnListener) Close() error {
	return l.Conn.Close()
}

func (l *singleConnListener) Addr() net.Addr {
	return l.Conn.LocalAddr()
}

func writeProxyError(w http.ResponseWriter, err error) {
	appErr := apperrors.Public(err)
	http.Error(w, appErr.Message, appErr.HTTPStatus)
}
