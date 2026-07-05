package desktop

import (
	"context"

	"cursorbridge/internal/app"
	"cursorbridge/internal/certs"
	"cursorbridge/internal/config"
	"cursorbridge/internal/cursor"
	"cursorbridge/internal/mitm"
	"cursorbridge/internal/relay"
)

type Service struct {
	app *app.App
}

func NewService(application *app.App) *Service {
	return &Service{app: application}
}

func (s *Service) Status() app.Status {
	return s.app.Status()
}

func (s *Service) Diagnostics() app.Diagnostics {
	return s.app.Diagnostics()
}

func (s *Service) SetupStatus() app.SetupStatus {
	return s.app.SetupStatus()
}

func (s *Service) PrepareSetup() (app.SetupStatus, error) {
	return s.app.PrepareSetup(context.Background())
}

func (s *Service) Config() config.RuntimeConfigSnapshot {
	return s.app.Config()
}

func (s *Service) SaveConfig(next config.UserConfig) (config.RuntimeConfigSnapshot, error) {
	return s.app.SaveConfig(next)
}

func (s *Service) UpsertAdapter(adapter config.ModelAdapter) (config.RuntimeConfigSnapshot, error) {
	return s.app.UpsertAdapter(adapter)
}

func (s *Service) DeleteAdapter(id string) (config.RuntimeConfigSnapshot, error) {
	return s.app.DeleteAdapter(id)
}

func (s *Service) PreviewAdapterImport(source string) (app.AdapterImportResponse, error) {
	return s.app.PreviewAdapterImport(source)
}

func (s *Service) ImportAdapters(source string) (app.AdapterImportResponse, error) {
	return s.app.ImportAdapters(source)
}

func (s *Service) StartProxy() (mitm.Status, error) {
	return s.app.StartProxy(context.Background())
}

func (s *Service) StopProxy() (mitm.Status, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.app.StopProxy(ctx); err != nil {
		return s.app.Status().Proxy, err
	}
	return s.app.Status().Proxy, nil
}

func (s *Service) CAInfo() (certs.CAInfo, error) {
	return s.app.CAInfo()
}

func (s *Service) CAInstallPlan() (certs.InstallPlan, error) {
	return s.app.CAInstallPlan()
}

func (s *Service) CursorPlan() (cursor.SettingsPlan, error) {
	return s.app.CursorPlan()
}

func (s *Service) ApplyCursorSettings() (map[string]any, error) {
	if err := s.app.ApplyCursorSettings(); err != nil {
		return nil, err
	}
	return s.app.Status().Cursor, nil
}

func (s *Service) Decision(input app.DecisionRequest) (relay.Decision, error) {
	return s.app.Decision(input)
}
