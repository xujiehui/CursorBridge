package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestHTTPHandlerServesStaticIndexAndSPAFallback(t *testing.T) {
	application, err := New(Options{DataDir: t.TempDir(), ProxyAddr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	staticDir := t.TempDir()
	index := []byte(`<!doctype html><html><body><div id="app"></div></body></html>`)
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), index, 0o600); err != nil {
		t.Fatal(err)
	}
	handler := application.HTTPHandler(staticDir)

	for _, path := range []string{"/", "/settings/model"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s got status %d body %s", path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `<div id="app">`) {
			t.Fatalf("%s did not serve index.html: %s", path, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("api route got status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestAdapterImportPreviewAndImport(t *testing.T) {
	application, err := New(Options{DataDir: t.TempDir(), ProxyAddr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	handler := application.HTTPHandler("")

	body, _ := json.Marshal(map[string]string{
		"source": `{
			"name": "Example Relay",
			"baseURL": "https://relay.example.com/v1/chat/completions",
			"apiKey": "sk-relay",
			"models": ["gpt-4o"]
		}`,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/adapters/import/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("preview got status %d body %s", rec.Code, rec.Body.String())
	}
	var preview AdapterImportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if len(preview.Adapters) != 1 {
		t.Fatalf("preview adapter count got %d", len(preview.Adapters))
	}
	if preview.Adapters[0].APIKey != "********" {
		t.Fatalf("preview leaked api key: %q", preview.Adapters[0].APIKey)
	}
	if preview.Adapters[0].BaseURL != "https://relay.example.com/v1" {
		t.Fatalf("preview baseURL got %q", preview.Adapters[0].BaseURL)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/adapters/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("import got status %d body %s", rec.Code, rec.Body.String())
	}
	var result AdapterImportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Report.Imported != 1 || result.Report.Updated != 0 {
		t.Fatalf("report got %+v", result.Report)
	}
	if len(application.Status().Config.ModelAdapters) != 1 {
		t.Fatalf("status adapter count got %d", len(application.Status().Config.ModelAdapters))
	}
}

func TestSetupStatusTracksModelProxyAndCursor(t *testing.T) {
	application, err := New(Options{DataDir: t.TempDir(), ProxyAddr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}

	setup := application.SetupStatus()
	if setup.ModelConfigured {
		t.Fatal("setup should not report model configured without adapters")
	}
	if setup.Ready {
		t.Fatal("setup should not be ready without a model and proxy")
	}

	if _, err := application.UpsertAdapter(config.ModelAdapter{
		ID: "relay", DisplayName: "Relay", Type: config.AdapterOpenAI,
		BaseURL: "https://relay.example.com/v1", APIKey: "sk", ModelID: "gpt-4o", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	setup = application.SetupStatus()
	if !setup.ModelConfigured || setup.EnabledAdapters != 1 {
		t.Fatalf("expected one configured model, got %+v", setup)
	}
	if len(setup.NextActions) == 0 {
		t.Fatal("expected setup to expose remaining local bridge actions")
	}
}

func TestPrepareSetupStartsProxyAndAppliesCursorSettings(t *testing.T) {
	application, err := New(Options{DataDir: t.TempDir(), ProxyAddr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.UpsertAdapter(config.ModelAdapter{
		ID: "relay", DisplayName: "Relay", Type: config.AdapterOpenAI,
		BaseURL: "https://relay.example.com/v1", APIKey: "sk", ModelID: "gpt-4o", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = application.StopProxy(context.Background())
	}()

	setup, err := application.PrepareSetup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !setup.Proxy.Running {
		t.Fatalf("proxy should be running after prepare: %+v", setup)
	}
	if !setup.ModelConfigured {
		t.Fatalf("model should be configured after prepare: %+v", setup)
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
