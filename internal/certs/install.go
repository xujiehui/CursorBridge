package certs

import (
	"context"
	"fmt"
	"runtime"
)

type InstallPlan struct {
	Supported bool     `json:"supported"`
	Platform  string   `json:"platform"`
	Commands  []string `json:"commands"`
	Note      string   `json:"note"`
}

func (m *Manager) InstallPlan() (InstallPlan, error) {
	info, err := m.Info()
	if err != nil {
		return InstallPlan{}, err
	}
	plan := InstallPlan{Platform: runtime.GOOS}
	switch runtime.GOOS {
	case "darwin":
		plan.Supported = true
		plan.Commands = []string{
			fmt.Sprintf("sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %q", info.CertPath),
		}
	case "windows":
		plan.Supported = true
		plan.Commands = []string{
			fmt.Sprintf("certutil -addstore -f Root %q", info.CertPath),
		}
	default:
		plan.Note = "automatic certificate installation is not implemented for this platform"
	}
	return plan, nil
}

func (m *Manager) InstallCA(context.Context) error {
	return fmt.Errorf("CA installation must be performed explicitly by the user using InstallPlan commands")
}
