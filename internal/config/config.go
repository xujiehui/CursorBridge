package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	AppName        = "Cursor助手"
	DefaultBaseURL = "http://127.0.0.1:8080"
	DefaultProxy   = "http://127.0.0.1:18080"
)

type AdapterType string

const (
	AdapterOpenAI    AdapterType = "openai"
	AdapterAnthropic AdapterType = "anthropic"
)

type UserConfig struct {
	BaseURL                 string         `json:"baseURL"`
	LicenseCode             string         `json:"licenseCode"`
	ProxyURL                string         `json:"proxyURL"`
	ObservabilityLogEnabled bool           `json:"observabilityLogEnabled"`
	ModelAdapters           []ModelAdapter `json:"modelAdapters"`
}

type ModelAdapter struct {
	ID          string      `json:"id"`
	DisplayName string      `json:"displayName"`
	Type        AdapterType `json:"type"`
	BaseURL     string      `json:"baseURL"`
	APIKey      string      `json:"apiKey"`
	ModelID     string      `json:"modelID"`
	Enabled     bool        `json:"enabled"`
}

type AdapterUpsertReport struct {
	Imported int `json:"imported"`
	Updated  int `json:"updated"`
}

type RuntimeConfigSnapshot struct {
	BaseURL                 string         `json:"baseURL"`
	LicenseCodeConfigured   bool           `json:"licenseCodeConfigured"`
	ProxyURL                string         `json:"proxyURL"`
	ObservabilityLogEnabled bool           `json:"observabilityLogEnabled"`
	ModelAdapters           []ModelAdapter `json:"modelAdapters"`
}

type Store struct {
	mu     sync.RWMutex
	path   string
	config UserConfig
}

func DefaultDataDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, AppName), nil
}

func NewStore(dataDir string) (*Store, error) {
	if dataDir == "" {
		var err error
		dataDir, err = DefaultDataDir()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}

	store := &Store{path: filepath.Join(dataDir, "config.json")}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) DataDir() string {
	return filepath.Dir(s.path)
}

func (s *Store) Snapshot() RuntimeConfigSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sanitize(s.config)
}

func (s *Store) Raw() UserConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConfig(s.config)
}

func (s *Store) Save(next UserConfig) error {
	next = normalize(next)
	s.mu.Lock()
	defer s.mu.Unlock()
	next = s.preserveSecretsLocked(next)
	if err := Validate(next); err != nil {
		return err
	}
	s.config = cloneConfig(next)
	return s.writeLocked()
}

func (s *Store) UpsertAdapter(adapter ModelAdapter) (UserConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	adapter = normalizeAdapter(adapter)
	adapter = s.preserveAdapterSecretLocked(adapter)
	if err := validateAdapter(adapter); err != nil {
		return UserConfig{}, err
	}

	found := false
	for i := range s.config.ModelAdapters {
		if s.config.ModelAdapters[i].ID == adapter.ID {
			s.config.ModelAdapters[i] = adapter
			found = true
			break
		}
	}
	if !found {
		s.config.ModelAdapters = append(s.config.ModelAdapters, adapter)
	}
	if err := s.writeLocked(); err != nil {
		return UserConfig{}, err
	}
	return cloneConfig(s.config), nil
}

func (s *Store) UpsertAdapters(adapters []ModelAdapter) (AdapterUpsertReport, UserConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(adapters) == 0 {
		return AdapterUpsertReport{}, UserConfig{}, errors.New("at least one adapter is required")
	}

	next := cloneConfig(s.config)
	report := AdapterUpsertReport{}
	seenIncoming := map[string]bool{}
	for _, adapter := range adapters {
		adapter = normalizeAdapter(adapter)
		adapter = s.preserveAdapterSecretLocked(adapter)
		if err := validateAdapter(adapter); err != nil {
			return AdapterUpsertReport{}, UserConfig{}, err
		}
		if seenIncoming[adapter.ID] {
			return AdapterUpsertReport{}, UserConfig{}, fmt.Errorf("duplicate imported adapter id %q", adapter.ID)
		}
		seenIncoming[adapter.ID] = true

		found := false
		for i := range next.ModelAdapters {
			if next.ModelAdapters[i].ID == adapter.ID {
				next.ModelAdapters[i] = adapter
				found = true
				report.Updated++
				break
			}
		}
		if !found {
			next.ModelAdapters = append(next.ModelAdapters, adapter)
			report.Imported++
		}
	}
	if err := Validate(next); err != nil {
		return AdapterUpsertReport{}, UserConfig{}, err
	}
	s.config = cloneConfig(next)
	if err := s.writeLocked(); err != nil {
		return AdapterUpsertReport{}, UserConfig{}, err
	}
	return report, cloneConfig(s.config), nil
}

func (s *Store) DeleteAdapter(id string) (UserConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.config.ModelAdapters[:0]
	for _, adapter := range s.config.ModelAdapters {
		if adapter.ID != id {
			next = append(next, adapter)
		}
	}
	s.config.ModelAdapters = next
	if err := s.writeLocked(); err != nil {
		return UserConfig{}, err
	}
	return cloneConfig(s.config), nil
}

func (s *Store) AdapterByModel(model string) (ModelAdapter, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	model = strings.TrimPrefix(model, "byok/")
	for _, adapter := range s.config.ModelAdapters {
		if adapter.Enabled && adapter.ModelID == model {
			return adapter, true
		}
	}
	return ModelAdapter{}, false
}

func (s *Store) load() error {
	defaultConfig := normalize(UserConfig{})
	content, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.config = defaultConfig
		return s.writeLocked()
	}
	if err != nil {
		return err
	}
	if len(content) == 0 {
		s.config = defaultConfig
		return s.writeLocked()
	}
	if err := json.Unmarshal(content, &s.config); err != nil {
		return err
	}
	s.config = normalize(s.config)
	return nil
}

func (s *Store) writeLocked() error {
	content, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, append(content, '\n'), 0o600)
}

func Validate(cfg UserConfig) error {
	if err := validateBaseURL(cfg.BaseURL); err != nil {
		return fmt.Errorf("invalid baseURL: %w", err)
	}
	if _, err := ProxyListenAddr(cfg.ProxyURL); err != nil {
		return fmt.Errorf("invalid proxyURL: %w", err)
	}
	seen := map[string]bool{}
	for _, adapter := range cfg.ModelAdapters {
		if err := validateAdapter(adapter); err != nil {
			return err
		}
		if seen[adapter.ID] {
			return fmt.Errorf("duplicate adapter id %q", adapter.ID)
		}
		seen[adapter.ID] = true
	}
	return nil
}

func ProxyListenAddr(proxyURL string) (string, error) {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		proxyURL = DefaultProxy
	}
	parsed, err := url.ParseRequestURI(proxyURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" {
		return "", errors.New("proxyURL must use http scheme")
	}
	if parsed.User != nil {
		return "", errors.New("proxyURL must not contain user info")
	}
	host := parsed.Hostname()
	port := parsed.Port()
	if host == "" || port == "" {
		return "", errors.New("proxyURL must include host and port")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", errors.New("proxyURL must not include a path")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("proxyURL must not include query or fragment")
	}
	if !isLoopbackHost(host) {
		return "", errors.New("proxyURL host must be localhost or a loopback address")
	}
	return net.JoinHostPort(host, port), nil
}

func validateBaseURL(baseURL string) error {
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("baseURL must use http or https scheme")
	}
	if parsed.Host == "" {
		return errors.New("baseURL must include a host")
	}
	return nil
}

func validateAdapter(adapter ModelAdapter) error {
	if strings.TrimSpace(adapter.ID) == "" {
		return errors.New("adapter id is required")
	}
	if strings.TrimSpace(adapter.DisplayName) == "" {
		return errors.New("adapter displayName is required")
	}
	if adapter.Type != AdapterOpenAI && adapter.Type != AdapterAnthropic {
		return fmt.Errorf("unsupported adapter type %q", adapter.Type)
	}
	if err := validateAdapterBaseURL(adapter.BaseURL); err != nil {
		return fmt.Errorf("invalid adapter baseURL: %w", err)
	}
	if strings.TrimSpace(adapter.APIKey) == "" {
		return errors.New("adapter apiKey is required")
	}
	if strings.TrimSpace(adapter.ModelID) == "" {
		return errors.New("adapter modelID is required")
	}
	return nil
}

func validateAdapterBaseURL(baseURL string) error {
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("adapter baseURL must use http or https scheme")
	}
	if parsed.Host == "" {
		return errors.New("adapter baseURL must include a host")
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func normalize(cfg UserConfig) UserConfig {
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	cfg.ProxyURL = strings.TrimSpace(cfg.ProxyURL)
	if cfg.ProxyURL == "" {
		cfg.ProxyURL = DefaultProxy
	}
	for i := range cfg.ModelAdapters {
		cfg.ModelAdapters[i] = normalizeAdapter(cfg.ModelAdapters[i])
	}
	return cfg
}

func normalizeAdapter(adapter ModelAdapter) ModelAdapter {
	adapter.ID = strings.TrimSpace(adapter.ID)
	adapter.DisplayName = strings.TrimSpace(adapter.DisplayName)
	adapter.Type = AdapterType(strings.TrimSpace(string(adapter.Type)))
	adapter.BaseURL = strings.TrimRight(strings.TrimSpace(adapter.BaseURL), "/")
	adapter.APIKey = strings.TrimSpace(adapter.APIKey)
	adapter.ModelID = strings.TrimPrefix(strings.TrimSpace(adapter.ModelID), "byok/")
	adapter.BaseURL = normalizeAdapterBaseURL(adapter.Type, adapter.BaseURL)
	return adapter
}

func normalizeAdapterBaseURL(adapterType AdapterType, baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	switch adapterType {
	case AdapterAnthropic:
		return strings.TrimSuffix(baseURL, "/messages")
	default:
		return strings.TrimSuffix(baseURL, "/chat/completions")
	}
}

func (s *Store) preserveSecretsLocked(next UserConfig) UserConfig {
	if isMaskPlaceholder(next.LicenseCode) {
		next.LicenseCode = s.config.LicenseCode
	}
	for i := range next.ModelAdapters {
		next.ModelAdapters[i] = s.preserveAdapterSecretLocked(next.ModelAdapters[i])
	}
	return next
}

func (s *Store) preserveAdapterSecretLocked(adapter ModelAdapter) ModelAdapter {
	if !isAdapterSecretPlaceholder(adapter.APIKey) {
		return adapter
	}
	for _, existing := range s.config.ModelAdapters {
		if existing.ID == adapter.ID {
			adapter.APIKey = existing.APIKey
			return adapter
		}
	}
	return adapter
}

func isMaskPlaceholder(value string) bool {
	value = strings.TrimSpace(value)
	return value == "********"
}

func isAdapterSecretPlaceholder(value string) bool {
	value = strings.TrimSpace(value)
	return value == "" || value == "********"
}

func sanitize(cfg UserConfig) RuntimeConfigSnapshot {
	adapters := make([]ModelAdapter, len(cfg.ModelAdapters))
	for i, adapter := range cfg.ModelAdapters {
		adapters[i] = adapter
		if adapters[i].APIKey != "" {
			adapters[i].APIKey = "********"
		}
	}
	return RuntimeConfigSnapshot{
		BaseURL:                 cfg.BaseURL,
		LicenseCodeConfigured:   cfg.LicenseCode != "",
		ProxyURL:                cfg.ProxyURL,
		ObservabilityLogEnabled: cfg.ObservabilityLogEnabled,
		ModelAdapters:           adapters,
	}
}

func cloneConfig(cfg UserConfig) UserConfig {
	cfg.ModelAdapters = append([]ModelAdapter(nil), cfg.ModelAdapters...)
	return cfg
}
