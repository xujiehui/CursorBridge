package config

import (
	"encoding/base64"
	"net/url"
	"testing"
)

func TestParseAdapterImportSourceFromJSON(t *testing.T) {
	preview, err := ParseAdapterImportSource(`{
		"name": "Example Relay",
		"base_url": "https://relay.example.com/v1/chat/completions",
		"api_key": "sk-relay",
		"models": ["gpt-4o", "claude-3-5-sonnet"]
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Adapters) != 2 {
		t.Fatalf("got %d adapters", len(preview.Adapters))
	}
	first := preview.Adapters[0]
	if first.BaseURL != "https://relay.example.com/v1" {
		t.Fatalf("baseURL was not normalized: %q", first.BaseURL)
	}
	if first.ID != "example-relay-relay-example-com-gpt-4o" {
		t.Fatalf("unexpected generated id %q", first.ID)
	}
	if first.APIKey != "sk-relay" {
		t.Fatalf("api key was not carried through")
	}
	if first.ModelID != "gpt-4o" {
		t.Fatalf("model id got %q", first.ModelID)
	}
}

func TestParseAdapterImportSourceFromQueryLink(t *testing.T) {
	source := "cursorbridge://import?" + url.Values{
		"name":    []string{"Relay"},
		"baseURL": []string{"https://relay.example.com/v1"},
		"apiKey":  []string{"sk-relay"},
		"models":  []string{"gpt-4o,gpt-4.1"},
	}.Encode()

	preview, err := ParseAdapterImportSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if preview.SourceType != "link:query" {
		t.Fatalf("sourceType got %q", preview.SourceType)
	}
	if len(preview.Adapters) != 2 {
		t.Fatalf("got %d adapters", len(preview.Adapters))
	}
	if preview.Adapters[1].ModelID != "gpt-4.1" {
		t.Fatalf("second model got %q", preview.Adapters[1].ModelID)
	}
}

func TestParseAdapterImportSourceFromEmbeddedBase64JSON(t *testing.T) {
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{
		"providerName": "Claude Relay",
		"type": "anthropic",
		"apiBaseURL": "https://claude.example.com/v1/messages",
		"token": "sk-ant",
		"model": "claude-3-5-sonnet"
	}`))

	preview, err := ParseAdapterImportSource("https://relay.example.com/import?config=" + payload)
	if err != nil {
		t.Fatal(err)
	}
	if preview.SourceType != "link:config" {
		t.Fatalf("sourceType got %q", preview.SourceType)
	}
	adapter := preview.Adapters[0]
	if adapter.Type != AdapterAnthropic {
		t.Fatalf("type got %q", adapter.Type)
	}
	if adapter.BaseURL != "https://claude.example.com/v1" {
		t.Fatalf("baseURL got %q", adapter.BaseURL)
	}
	if adapter.ID != "claude-relay-claude-example-com-claude-3-5-sonnet" {
		t.Fatalf("id got %q", adapter.ID)
	}
}

func TestUpsertAdaptersReportsImportsAndUpdates(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	report, _, err := store.UpsertAdapters([]ModelAdapter{
		{
			ID: "relay", DisplayName: "Relay GPT-4o", Type: AdapterOpenAI,
			BaseURL: "https://relay.example.com/v1", APIKey: "sk-one", ModelID: "gpt-4o", Enabled: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Imported != 1 || report.Updated != 0 {
		t.Fatalf("first report = %+v", report)
	}

	report, cfg, err := store.UpsertAdapters([]ModelAdapter{
		{
			ID: "relay", DisplayName: "Relay GPT-4o Updated", Type: AdapterOpenAI,
			BaseURL: "https://relay.example.com/v1", APIKey: "********", ModelID: "gpt-4o", Enabled: true,
		},
		{
			ID: "relay-41", DisplayName: "Relay GPT-4.1", Type: AdapterOpenAI,
			BaseURL: "https://relay.example.com/v1", APIKey: "sk-one", ModelID: "gpt-4.1", Enabled: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Imported != 1 || report.Updated != 1 {
		t.Fatalf("second report = %+v", report)
	}
	if cfg.ModelAdapters[0].APIKey != "sk-one" {
		t.Fatalf("masked import overwrote key: %q", cfg.ModelAdapters[0].APIKey)
	}
	if len(cfg.ModelAdapters) != 2 {
		t.Fatalf("adapter count got %d", len(cfg.ModelAdapters))
	}
}
