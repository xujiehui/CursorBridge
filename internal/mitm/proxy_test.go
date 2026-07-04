package mitm

import (
	"testing"

	"cursorbridge/internal/config"
	"cursorbridge/internal/relay"
)

type stubStore struct{}

func (stubStore) Raw() config.UserConfig {
	return config.UserConfig{BaseURL: "https://cursor.example", ProxyURL: config.DefaultProxy}
}

func (stubStore) AdapterByModel(string) (config.ModelAdapter, bool) {
	return config.ModelAdapter{}, false
}

func TestProxyStatus(t *testing.T) {
	gateway := relay.NewGateway(stubStore{}, nil)
	proxy := New("127.0.0.1:0", gateway, nil)
	status := proxy.Status()
	if status.Running {
		t.Fatalf("proxy should not be running before Start")
	}
}
