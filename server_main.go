//go:build !desktop

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"cursorbridge/internal/app"
)

func main() {
	if err := runServer(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func runServer() error {
	var (
		addr      = flag.String("addr", "127.0.0.1:8080", "bridge API address")
		proxyAddr = flag.String("proxy-addr", "", "local proxy address override, for example 127.0.0.1:18080")
		dataDir   = flag.String("data-dir", "", "application data directory")
	)
	flag.Parse()
	proxyAddrExplicit := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "proxy-addr" {
			proxyAddrExplicit = true
		}
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := app.New(app.Options{
		DataDir:   *dataDir,
		ProxyAddr: proxyAddrOption(*proxyAddr, proxyAddrExplicit),
	})
	if err != nil {
		return err
	}

	handler := application.HTTPHandler(staticDir())
	server := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("bridge api listening", "addr", "http://"+*addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = application.StopProxy(shutdownCtx)
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func proxyAddrOption(value string, explicit bool) string {
	if explicit {
		return value
	}
	return ""
}

func staticDir() string {
	candidates := []string{
		filepath.Join("frontend", "dist"),
		"dist",
	}
	if executable, err := os.Executable(); err == nil {
		execDir := filepath.Dir(executable)
		candidates = append(candidates,
			filepath.Join(execDir, "frontend", "dist"),
			filepath.Join(execDir, "dist"),
			filepath.Join(execDir, "..", "Resources", "frontend", "dist"),
			filepath.Join(execDir, "..", "Resources", "dist"),
		)
	}
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			if _, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil {
				return candidate
			}
		}
	}
	fmt.Println("frontend/dist not found; serving API only")
	return ""
}
