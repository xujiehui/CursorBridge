package cursor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Settings struct {
	HTTPProxy       string `json:"http.proxy,omitempty"`
	HTTPProxyStrict bool   `json:"http.proxyStrictSSL"`
}

type Integration struct {
	proxyURL string
}

type SettingsPlan struct {
	Supported    bool     `json:"supported"`
	SettingsPath string   `json:"settingsPath"`
	ProxyURL     string   `json:"proxyURL"`
	Changes      Settings `json:"changes"`
	Current      Settings `json:"current"`
	Exists       bool     `json:"exists"`
	Applied      bool     `json:"applied"`
	Warnings     []string `json:"warnings"`
}

func NewIntegration(proxyURL string) *Integration {
	return &Integration{proxyURL: proxyURL}
}

func (i *Integration) SettingsPath() (string, error) {
	var base string
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "Library", "Application Support", "Cursor", "User")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", errors.New("APPDATA is not set")
		}
		base = filepath.Join(appData, "Cursor", "User")
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config", "Cursor", "User")
	}
	return filepath.Join(base, "settings.json"), nil
}

func (i *Integration) PlanSettingsUpdate() (SettingsPlan, error) {
	path, err := i.SettingsPath()
	if err != nil {
		return SettingsPlan{}, err
	}
	plan := SettingsPlan{
		Supported:    runtime.GOOS == "darwin" || runtime.GOOS == "windows" || runtime.GOOS == "linux",
		SettingsPath: path,
		ProxyURL:     i.proxyURL,
		Warnings:     []string{},
		Changes: Settings{
			HTTPProxy:       i.proxyURL,
			HTTPProxyStrict: false,
		},
	}
	current, exists, err := readSettings(path)
	if err != nil {
		plan.Warnings = append(plan.Warnings, err.Error())
	} else {
		plan.Current = current
		plan.Exists = exists
		plan.Applied = current.HTTPProxy == plan.Changes.HTTPProxy && current.HTTPProxyStrict == plan.Changes.HTTPProxyStrict
	}
	if runtime.GOOS == "linux" {
		plan.Warnings = append(plan.Warnings, "Cursor settings path can vary by Linux distribution")
	}
	return plan, nil
}

func (i *Integration) ApplySettings() error {
	plan, err := i.PlanSettingsUpdate()
	if err != nil {
		return err
	}
	if !plan.Supported {
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}

	current := map[string]any{}
	content, err := os.ReadFile(plan.SettingsPath)
	if err == nil && len(content) > 0 {
		if err := json.Unmarshal(content, &current); err != nil {
			return fmt.Errorf("parse cursor settings: %w", err)
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	current["http.proxy"] = plan.Changes.HTTPProxy
	current["http.proxyStrictSSL"] = plan.Changes.HTTPProxyStrict
	content, err = json.MarshalIndent(current, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(plan.SettingsPath), 0o700); err != nil {
		return err
	}
	return os.WriteFile(plan.SettingsPath, append(content, '\n'), 0o600)
}

func (i *Integration) Status() map[string]any {
	plan, err := i.PlanSettingsUpdate()
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{
		"ok":           true,
		"settingsPath": plan.SettingsPath,
		"settingsFile": plan.Exists,
		"proxyURL":     plan.ProxyURL,
		"applied":      plan.Applied,
	}
}

func readSettings(path string) (Settings, bool, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Settings{}, false, nil
	}
	if err != nil {
		return Settings{}, false, err
	}
	if len(content) == 0 {
		return Settings{}, true, nil
	}
	current := map[string]any{}
	if err := json.Unmarshal(content, &current); err != nil {
		return Settings{}, true, fmt.Errorf("parse cursor settings: %w", err)
	}
	return Settings{
		HTTPProxy:       stringValue(current["http.proxy"]),
		HTTPProxyStrict: boolValue(current["http.proxyStrictSSL"]),
	}, true, nil
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func boolValue(value any) bool {
	if flag, ok := value.(bool); ok {
		return flag
	}
	return false
}
