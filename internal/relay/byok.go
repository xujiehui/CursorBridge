package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cursorbridge/internal/apperrors"
	"cursorbridge/internal/config"
	"cursorbridge/internal/observability"
)

type BYOKGateway struct {
	store  ConfigStore
	client *http.Client
	logs   *observability.Logger
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func NewBYOKGateway(store ConfigStore, logs *observability.Logger) *BYOKGateway {
	return &BYOKGateway{
		store: store,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		logs: logs,
	}
}

func (g *BYOKGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8<<20))
	if err != nil {
		return apperrors.BadRequest("request body is too large")
	}
	return g.ServeBytes(w, r, body)
}

func (g *BYOKGateway) ServeBytes(w http.ResponseWriter, r *http.Request, body []byte) error {
	var payload ChatRequest
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&payload); err != nil {
		return apperrors.New(apperrors.ErrInvalidBidiAppendPayload, "invalid JSON request", http.StatusBadRequest)
	}
	if !strings.HasPrefix(payload.Model, "byok/") {
		return apperrors.BadRequest("model must use byok/<modelID>")
	}
	adapter, ok := g.store.AdapterByModel(payload.Model)
	if !ok {
		return apperrors.New(apperrors.ErrByokChannelNotAvailable, "no enabled BYOK adapter found for "+payload.Model, http.StatusServiceUnavailable)
	}

	payload.Model = strings.TrimPrefix(payload.Model, "byok/")
	body, endpoint, err := encodeAdapterRequest(adapter, payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return apperrors.Wrap(apperrors.ErrByokChannelNotAvailable, "create upstream request", http.StatusServiceUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	applyAuthHeaders(req, adapter)

	start := time.Now()
	resp, err := g.client.Do(req)
	if err != nil {
		_ = g.logs.ChannelCall("byok_error", map[string]any{"adapterID": adapter.ID, "error": err.Error()})
		return apperrors.Wrap(apperrors.ErrByokChannelNotAvailable, "call upstream model", http.StatusServiceUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return apperrors.New(apperrors.ErrByokChannelRateLimited, "upstream model rate limited this request", http.StatusTooManyRequests)
	}
	if resp.StatusCode >= 500 {
		return apperrors.New(apperrors.ErrByokChannelNotAvailable, fmt.Sprintf("upstream model returned %d", resp.StatusCode), http.StatusServiceUnavailable)
	}

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		return err
	}
	_ = g.logs.ChannelCall("byok_call", map[string]any{
		"adapterID":  adapter.ID,
		"type":       adapter.Type,
		"status":     resp.StatusCode,
		"durationMs": time.Since(start).Milliseconds(),
	})
	return nil
}

func (g *BYOKGateway) TestAdapter(ctx context.Context, adapter config.ModelAdapter) error {
	adapter.Enabled = true
	if err := config.Validate(config.UserConfig{
		BaseURL:       config.DefaultBaseURL,
		ProxyURL:      config.DefaultProxy,
		ModelAdapters: []config.ModelAdapter{adapter},
		LicenseCode:   "test",
	}); err != nil {
		return err
	}

	body, endpoint, err := encodeAdapterRequest(adapter, ChatRequest{
		Model: adapter.ModelID,
		Messages: []ChatMessage{{
			Role:    "user",
			Content: "ping",
		}},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	applyAuthHeaders(req, adapter)
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("adapter returned %d", resp.StatusCode)
	}
	return nil
}

func encodeAdapterRequest(adapter config.ModelAdapter, payload ChatRequest) ([]byte, string, error) {
	switch adapter.Type {
	case config.AdapterOpenAI:
		payload.Model = adapter.ModelID
		body, err := json.Marshal(payload)
		return body, adapter.BaseURL + "/chat/completions", err
	case config.AdapterAnthropic:
		body, err := json.Marshal(map[string]any{
			"model":      adapter.ModelID,
			"messages":   payload.Messages,
			"stream":     payload.Stream,
			"max_tokens": defaultMaxTokens(payload.MaxTokens),
		})
		return body, adapter.BaseURL + "/messages", err
	default:
		return nil, "", apperrors.BadRequest("unsupported adapter type")
	}
}

func applyAuthHeaders(req *http.Request, adapter config.ModelAdapter) {
	switch adapter.Type {
	case config.AdapterAnthropic:
		req.Header.Set("x-api-key", adapter.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		req.Header.Set("Authorization", "Bearer "+adapter.APIKey)
	}
}

func defaultMaxTokens(value *int) int {
	if value == nil || *value <= 0 {
		return 1024
	}
	return *value
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
