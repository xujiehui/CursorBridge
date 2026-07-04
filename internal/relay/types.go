package relay

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"cursorbridge/internal/config"
)

type RouteMode string

const (
	RouteDirect        RouteMode = "routeDirect"
	RouteRelay         RouteMode = "routeRelay"
	RouteSelfImpl      RouteMode = "selfImplemented"
	RawCursorURLHeader           = "x-raw-cursor-server-url"
)

type Decision struct {
	Mode       RouteMode `json:"mode"`
	Target     string    `json:"target"`
	Reason     string    `json:"reason"`
	AdapterID  string    `json:"adapterID,omitempty"`
	AdapterTyp string    `json:"adapterType,omitempty"`
}

type ConfigStore interface {
	Raw() config.UserConfig
	AdapterByModel(model string) (config.ModelAdapter, bool)
}

func Decide(r *http.Request, store ConfigStore) Decision {
	return DecideWithBody(r, nil, store)
}

func DecideWithBody(r *http.Request, body []byte, store ConfigStore) Decision {
	model := ExtractModel(r)
	if model == "" && len(body) > 0 {
		model = extractModelFromBody(body)
	}
	if strings.HasPrefix(model, "byok/") {
		if adapter, ok := store.AdapterByModel(model); ok {
			return Decision{
				Mode:       RouteSelfImpl,
				Target:     adapter.BaseURL,
				Reason:     "model id uses byok prefix and adapter is configured",
				AdapterID:  adapter.ID,
				AdapterTyp: string(adapter.Type),
			}
		}
		return Decision{Mode: RouteRelay, Target: rawTarget(r, store), Reason: "byok model requested but no adapter matched"}
	}
	if r.Header.Get(RawCursorURLHeader) != "" {
		return Decision{Mode: RouteRelay, Target: rawTarget(r, store), Reason: "raw cursor server header is present"}
	}
	return Decision{Mode: RouteDirect, Target: rawTarget(r, store), Reason: "default direct forwarding"}
}

func ExtractModel(r *http.Request) string {
	for _, name := range []string{"x-cursor-model", "x-model", "model"} {
		if value := strings.TrimSpace(r.Header.Get(name)); value != "" {
			return value
		}
	}
	queryModel := strings.TrimSpace(r.URL.Query().Get("model"))
	return queryModel
}

func rawTarget(r *http.Request, store ConfigStore) string {
	if raw := strings.TrimSpace(r.Header.Get(RawCursorURLHeader)); raw != "" {
		return raw
	}
	if r.URL != nil && r.URL.Scheme != "" && r.URL.Host != "" {
		return r.URL.Scheme + "://" + r.URL.Host
	}
	cfg := store.Raw()
	if cfg.BaseURL != "" {
		return cfg.BaseURL
	}
	return config.DefaultBaseURL
}

func extractModelFromBody(body []byte) string {
	var payload struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Model)
}

func JoinTarget(base string, requestURL *url.URL) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}
	parsed.Path = singleJoiningSlash(parsed.Path, requestURL.Path)
	parsed.RawQuery = requestURL.RawQuery
	return parsed.String(), nil
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	default:
		return a + b
	}
}
