package config

import "testing"

func TestSavePreservesMaskedSecrets(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.UpsertAdapter(ModelAdapter{
		ID:          "openai",
		DisplayName: "OpenAI",
		Type:        AdapterOpenAI,
		BaseURL:     "https://api.openai.com/v1",
		APIKey:      "sk-real",
		ModelID:     "gpt-4o",
		Enabled:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(UserConfig{
		BaseURL:     DefaultBaseURL,
		LicenseCode: "********",
		ProxyURL:    DefaultProxy,
		ModelAdapters: []ModelAdapter{{
			ID:          "openai",
			DisplayName: "OpenAI",
			Type:        AdapterOpenAI,
			BaseURL:     "https://api.openai.com/v1",
			APIKey:      "********",
			ModelID:     "gpt-4o",
			Enabled:     true,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	cfg := store.Raw()
	if cfg.ModelAdapters[0].APIKey != "sk-real" {
		t.Fatalf("masked API key overwrote secret: %q", cfg.ModelAdapters[0].APIKey)
	}
}

func TestSaveAllowsClearingLicenseCode(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(UserConfig{
		BaseURL:     DefaultBaseURL,
		LicenseCode: "license",
		ProxyURL:    DefaultProxy,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(UserConfig{
		BaseURL:     DefaultBaseURL,
		LicenseCode: "",
		ProxyURL:    DefaultProxy,
	}); err != nil {
		t.Fatal(err)
	}

	cfg := store.Raw()
	if cfg.LicenseCode != "" {
		t.Fatalf("license should be cleared, got %q", cfg.LicenseCode)
	}
}

func TestProxyListenAddrRequiresLoopbackHTTP(t *testing.T) {
	addr, err := ProxyListenAddr("http://127.0.0.1:18080")
	if err != nil {
		t.Fatal(err)
	}
	if addr != "127.0.0.1:18080" {
		t.Fatalf("got %q", addr)
	}

	for _, value := range []string{
		"https://127.0.0.1:18080",
		"http://0.0.0.0:18080",
		"http://192.168.1.10:18080",
		"http://127.0.0.1:18080/proxy",
	} {
		if _, err := ProxyListenAddr(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}
