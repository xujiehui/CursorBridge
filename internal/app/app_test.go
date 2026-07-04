package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cursorbridge/internal/config"
)

func TestHealth(t *testing.T) {
	application, err := New(Options{DataDir: t.TempDir(), ProxyAddr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	application.HTTPHandler("").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d", rec.Code)
	}
}

func TestStatusCAAndDecision(t *testing.T) {
	application, err := New(Options{DataDir: t.TempDir(), ProxyAddr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	handler := application.HTTPHandler("")

	for _, path := range []string{"/api/status", "/api/diagnostics", "/api/certs/ca", "/api/certs/install-plan", "/api/cursor/plan"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s got status %d body %s", path, rec.Code, rec.Body.String())
		}
	}

	body, _ := json.Marshal(map[string]any{
		"path": "/v1/chat/completions",
		"headers": map[string]string{
			"x-cursor-model": "byok/gpt-4o",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/relay/decision", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("decision got status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestSaveConfigSyncsStoppedProxyAddr(t *testing.T) {
	application, err := New(Options{DataDir: t.TempDir(), ProxyAddr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := application.SaveConfig(config.UserConfig{
		BaseURL:  config.DefaultBaseURL,
		ProxyURL: "http://localhost:19090",
	}); err != nil {
		t.Fatal(err)
	}

	if got := application.Status().Proxy.Addr; got != "localhost:19090" {
		t.Fatalf("proxy addr was not synced, got %q", got)
	}
}

func TestSaveConfigRejectsProxyAddrChangeWhileRunning(t *testing.T) {
	application, err := New(Options{DataDir: t.TempDir(), ProxyAddr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.StartProxy(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = application.StopProxy(context.Background())
	}()

	_, err = application.SaveConfig(config.UserConfig{
		BaseURL:  config.DefaultBaseURL,
		ProxyURL: "http://localhost:19090",
	})
	if err == nil {
		t.Fatal("expected proxyURL change to be rejected while proxy is running")
	}
}
