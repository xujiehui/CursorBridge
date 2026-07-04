package relay

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"

	"cursorbridge/internal/config"
)

type testStore struct {
	cfg config.UserConfig
}

func (s testStore) Raw() config.UserConfig {
	return s.cfg
}

func (s testStore) AdapterByModel(model string) (config.ModelAdapter, bool) {
	model = model[len("byok/"):]
	for _, adapter := range s.cfg.ModelAdapters {
		if adapter.ModelID == model && adapter.Enabled {
			return adapter, true
		}
	}
	return config.ModelAdapter{}, false
}

func TestDecideBYOK(t *testing.T) {
	req := &http.Request{
		Header: http.Header{"X-Cursor-Model": []string{"byok/gpt-4o"}},
		URL:    &url.URL{Path: "/v1/chat/completions"},
	}
	decision := Decide(req, testStore{cfg: config.UserConfig{
		BaseURL:  "https://cursor.example",
		ProxyURL: config.DefaultProxy,
		ModelAdapters: []config.ModelAdapter{{
			ID: "openai", Type: config.AdapterOpenAI, ModelID: "gpt-4o", BaseURL: "https://api.example", APIKey: "sk", Enabled: true,
		}},
	}})
	if decision.Mode != RouteSelfImpl {
		t.Fatalf("expected self implemented route, got %s", decision.Mode)
	}
}

func TestDecideBYOKFromBody(t *testing.T) {
	req := &http.Request{
		Header: http.Header{},
		URL:    &url.URL{Path: "/v1/chat/completions"},
	}
	decision := DecideWithBody(req, []byte(`{"model":"byok/gpt-4o"}`), testStore{cfg: config.UserConfig{
		BaseURL:  "https://cursor.example",
		ProxyURL: config.DefaultProxy,
		ModelAdapters: []config.ModelAdapter{{
			ID: "openai", Type: config.AdapterOpenAI, ModelID: "gpt-4o", BaseURL: "https://api.example", APIKey: "sk", Enabled: true,
		}},
	}})
	if decision.Mode != RouteSelfImpl {
		t.Fatalf("expected self implemented route from body, got %s", decision.Mode)
	}
}

func TestGatewayReadsBodyBeforeDecision(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader([]byte(`{"model":"byok/missing"}`)))
	if err != nil {
		t.Fatal(err)
	}
	decision := DecideWithBody(req, []byte(`{"model":"byok/missing"}`), testStore{cfg: config.UserConfig{
		BaseURL:  "https://cursor.example",
		ProxyURL: config.DefaultProxy,
	}})
	if decision.Mode != RouteRelay {
		t.Fatalf("expected relay fallback for missing adapter, got %s", decision.Mode)
	}
}

func TestJoinTarget(t *testing.T) {
	u, _ := url.Parse("/aiserver/v1/run?model=x")
	got, err := JoinTarget("https://cursor.example/base", u)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://cursor.example/base/aiserver/v1/run?model=x"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
