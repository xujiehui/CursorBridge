package relay

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"cursorbridge/internal/apperrors"
	"cursorbridge/internal/observability"
)

type Gateway struct {
	store ConfigStore
	byok  *BYOKGateway
	logs  *observability.Logger
	http  *http.Client
}

func NewGateway(store ConfigStore, logs *observability.Logger) *Gateway {
	return &Gateway{
		store: store,
		logs:  logs,
		byok:  NewBYOKGateway(store, logs),
		http: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (g *Gateway) Decision(r *http.Request) Decision {
	return Decide(r, g.store)
}

func (g *Gateway) BYOK() *BYOKGateway {
	return g.byok
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32<<20))
	if err != nil {
		return apperrors.BadRequest("request body is too large")
	}
	decision := DecideWithBody(r, body, g.store)
	if decision.Mode == RouteSelfImpl {
		return g.byok.ServeBytes(w, r, body)
	}
	return g.forwardBody(w, r, decision, body)
}

func (g *Gateway) forward(w http.ResponseWriter, r *http.Request, decision Decision) error {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32<<20))
	if err != nil {
		return apperrors.BadRequest("request body is too large")
	}
	return g.forwardBody(w, r, decision, body)
}

func (g *Gateway) forwardBody(w http.ResponseWriter, r *http.Request, decision Decision, body []byte) error {
	target, err := JoinTarget(decision.Target, r.URL)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrInvalidSystemSetting, "invalid target", http.StatusInternalServerError, err)
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, bytes.NewReader(body))
	if err != nil {
		return apperrors.Wrap(apperrors.ErrUpstream, "create upstream request", http.StatusBadGateway, err)
	}
	copyHeaders(req.Header, r.Header)
	req.Header.Del(RawCursorURLHeader)
	req.Host = req.URL.Host

	start := time.Now()
	resp, err := g.http.Do(req)
	if err != nil {
		_ = g.logs.RunUsage("relay_forward_error", map[string]any{"target": target, "error": err.Error()})
		return apperrors.Wrap(apperrors.ErrUpstream, "upstream request failed", http.StatusBadGateway, err)
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		return err
	}
	_ = g.logs.RunUsage("relay_forward", map[string]any{
		"mode":       decision.Mode,
		"target":     target,
		"status":     resp.StatusCode,
		"durationMs": time.Since(start).Milliseconds(),
	})
	return nil
}

func IsBYOKPath(path string) bool {
	path = strings.Trim(path, "/")
	return strings.HasSuffix(path, "chat/completions") || strings.Contains(path, "aiserver")
}
