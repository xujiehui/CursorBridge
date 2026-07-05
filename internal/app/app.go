package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cursorbridge/internal/apperrors"
	"cursorbridge/internal/certs"
	"cursorbridge/internal/config"
	"cursorbridge/internal/cursor"
	"cursorbridge/internal/mitm"
	"cursorbridge/internal/observability"
	"cursorbridge/internal/relay"
)

type Options struct {
	DataDir   string
	ProxyAddr string
}

type App struct {
	config *config.Store
	certs  *certs.Manager
	cursor *cursor.Integration
	relay  *relay.Gateway
	proxy  *mitm.Proxy
	logs   *observability.Logger
}

type Status struct {
	Health     string                       `json:"health"`
	ConfigPath string                       `json:"configPath"`
	DataDir    string                       `json:"dataDir"`
	Proxy      mitm.Status                  `json:"proxy"`
	Config     config.RuntimeConfigSnapshot `json:"config"`
	Cursor     map[string]any               `json:"cursor"`
}

type Diagnostics struct {
	Ready       bool              `json:"ready"`
	Items       []DiagnosticItem  `json:"items"`
	Logs        map[string]string `json:"logs"`
	NextActions []string          `json:"nextActions"`
}

type SetupStatus struct {
	Ready           bool                `json:"ready"`
	ModelConfigured bool                `json:"modelConfigured"`
	EnabledAdapters int                 `json:"enabledAdapters"`
	Proxy           mitm.Status         `json:"proxy"`
	Cursor          cursor.SettingsPlan `json:"cursor"`
	Warnings        []string            `json:"warnings"`
	NextActions     []string            `json:"nextActions"`
}

type DiagnosticItem struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	State   string `json:"state"`
	Detail  string `json:"detail"`
	Healthy bool   `json:"healthy"`
}

type DecisionRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
}

type AdapterImportRequest struct {
	Source string `json:"source"`
}

type AdapterImportResponse struct {
	SourceType string                        `json:"sourceType"`
	Adapters   []config.ModelAdapter         `json:"adapters"`
	Warnings   []string                      `json:"warnings"`
	Report     *config.AdapterUpsertReport   `json:"report,omitempty"`
	Config     *config.RuntimeConfigSnapshot `json:"config,omitempty"`
}

func New(options Options) (*App, error) {
	store, err := config.NewStore(options.DataDir)
	if err != nil {
		return nil, err
	}
	certManager, err := certs.NewManager(store.DataDir())
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(options.ProxyAddr) == "" {
		var err error
		options.ProxyAddr, err = config.ProxyListenAddr(store.Raw().ProxyURL)
		if err != nil {
			return nil, err
		}
	}

	logs := observability.New(store.DataDir(), func() bool {
		return store.Raw().ObservabilityLogEnabled
	})
	gateway := relay.NewGateway(store, logs)
	cursorIntegration := cursor.NewIntegration(store.Raw().ProxyURL)
	proxy := mitm.New(options.ProxyAddr, gateway, certManager)

	return &App{
		config: store,
		certs:  certManager,
		cursor: cursorIntegration,
		relay:  gateway,
		proxy:  proxy,
		logs:   logs,
	}, nil
}

func (a *App) Status() Status {
	return Status{
		Health:     "ok",
		ConfigPath: a.config.Path(),
		DataDir:    a.config.DataDir(),
		Proxy:      a.proxy.Status(),
		Config:     a.config.Snapshot(),
		Cursor:     a.cursor.Status(),
	}
}

func (a *App) Diagnostics() Diagnostics {
	status := a.Status()
	caInfo, caErr := a.CAInfo()
	cursorPlan, cursorErr := a.CursorPlan()
	adapterCount := countEnabledAdapters(status.Config.ModelAdapters)

	items := []DiagnosticItem{
		{
			ID:      "bridge",
			Label:   "桥接服务",
			State:   status.Health,
			Detail:  "Go 服务已启动，前端可以访问本地 API",
			Healthy: status.Health == "ok",
		},
		{
			ID:      "proxy",
			Label:   "本地代理",
			State:   runningState(status.Proxy.Running),
			Detail:  "监听 " + status.Proxy.Addr,
			Healthy: status.Proxy.Running,
		},
		{
			ID:      "cert",
			Label:   "本地 CA",
			State:   certState(caErr),
			Detail:  certDetail(caInfo, caErr),
			Healthy: caErr == nil,
		},
		{
			ID:      "cursor",
			Label:   "Cursor 设置",
			State:   cursorState(cursorPlan, cursorErr),
			Detail:  cursorDetail(cursorPlan, cursorErr),
			Healthy: cursorErr == nil && cursorPlan.Applied,
		},
		{
			ID:      "adapters",
			Label:   "BYOK 适配器",
			State:   adapterState(adapterCount),
			Detail:  adapterDetail(adapterCount),
			Healthy: adapterCount > 0,
		},
	}

	nextActions := []string{}
	for _, item := range items {
		if !item.Healthy {
			switch item.ID {
			case "proxy":
				nextActions = append(nextActions, "启动本地代理")
			case "cursor":
				nextActions = append(nextActions, "写入 Cursor 代理设置")
			case "adapters":
				nextActions = append(nextActions, "添加至少一个 BYOK 模型适配器")
			}
		}
	}

	return Diagnostics{
		Ready: allHealthy(items),
		Items: items,
		Logs: map[string]string{
			"runUsage":     filepath.Join(a.config.DataDir(), "run-usage.jsonl"),
			"channelCalls": filepath.Join(a.config.DataDir(), "channel-calls.jsonl"),
		},
		NextActions: nextActions,
	}
}

func (a *App) StartProxy(context.Context) (mitm.Status, error) {
	if err := a.proxy.Start(); err != nil {
		return a.proxy.Status(), err
	}
	_ = a.logs.RunUsage("proxy_started", map[string]any{"addr": a.proxy.Status().Addr})
	return a.proxy.Status(), nil
}

func (a *App) StopProxy(ctx context.Context) error {
	if err := a.proxy.Stop(ctx); err != nil {
		return err
	}
	_ = a.logs.RunUsage("proxy_stopped", map[string]any{"addr": a.proxy.Status().Addr})
	return nil
}

func (a *App) SetupStatus() SetupStatus {
	return a.setupStatus(nil)
}

func (a *App) PrepareSetup(ctx context.Context) (SetupStatus, error) {
	warnings := []string{}
	if !a.proxy.Status().Running {
		if _, err := a.StartProxy(ctx); err != nil {
			warnings = append(warnings, "启动本地桥接失败: "+err.Error())
		}
	}
	if err := a.ApplyCursorSettings(); err != nil {
		warnings = append(warnings, "写入 Cursor 设置失败: "+err.Error())
	}
	return a.setupStatus(warnings), nil
}

func (a *App) Config() config.RuntimeConfigSnapshot {
	return a.config.Snapshot()
}

func (a *App) SaveConfig(next config.UserConfig) (config.RuntimeConfigSnapshot, error) {
	current := a.config.Raw()
	if strings.TrimSpace(next.ProxyURL) != "" && next.ProxyURL != current.ProxyURL && a.proxy.Status().Running {
		return config.RuntimeConfigSnapshot{}, apperrors.New(apperrors.ErrInvalidSystemSetting, "stop the proxy before changing proxyURL", http.StatusConflict)
	}
	if err := a.config.Save(next); err != nil {
		return config.RuntimeConfigSnapshot{}, err
	}
	if err := a.syncRuntimeConfig(); err != nil {
		return config.RuntimeConfigSnapshot{}, err
	}
	return a.config.Snapshot(), nil
}

func (a *App) UpsertAdapter(adapter config.ModelAdapter) (config.RuntimeConfigSnapshot, error) {
	if _, err := a.config.UpsertAdapter(adapter); err != nil {
		return config.RuntimeConfigSnapshot{}, err
	}
	return a.config.Snapshot(), nil
}

func (a *App) DeleteAdapter(id string) (config.RuntimeConfigSnapshot, error) {
	if _, err := a.config.DeleteAdapter(id); err != nil {
		return config.RuntimeConfigSnapshot{}, err
	}
	return a.config.Snapshot(), nil
}

func (a *App) PreviewAdapterImport(source string) (AdapterImportResponse, error) {
	preview, err := config.ParseAdapterImportSource(source)
	if err != nil {
		return AdapterImportResponse{}, err
	}
	return AdapterImportResponse{
		SourceType: preview.SourceType,
		Adapters:   maskAdapterSecrets(preview.Adapters),
		Warnings:   preview.Warnings,
	}, nil
}

func (a *App) ImportAdapters(source string) (AdapterImportResponse, error) {
	preview, err := config.ParseAdapterImportSource(source)
	if err != nil {
		return AdapterImportResponse{}, err
	}
	report, _, err := a.config.UpsertAdapters(preview.Adapters)
	if err != nil {
		return AdapterImportResponse{}, err
	}
	snapshot := a.config.Snapshot()
	return AdapterImportResponse{
		SourceType: preview.SourceType,
		Adapters:   maskAdapterSecrets(preview.Adapters),
		Warnings:   preview.Warnings,
		Report:     &report,
		Config:     &snapshot,
	}, nil
}

func (a *App) CAInfo() (certs.CAInfo, error) {
	return a.certs.Info()
}

func (a *App) CAInstallPlan() (certs.InstallPlan, error) {
	return a.certs.InstallPlan()
}

func (a *App) CursorPlan() (cursor.SettingsPlan, error) {
	return a.cursor.PlanSettingsUpdate()
}

func (a *App) ApplyCursorSettings() error {
	return a.cursor.ApplySettings()
}

func (a *App) syncRuntimeConfig() error {
	cfg := a.config.Raw()
	addr, err := config.ProxyListenAddr(cfg.ProxyURL)
	if err != nil {
		return err
	}
	if err := a.proxy.SetAddr(addr); err != nil {
		return err
	}
	a.cursor = cursor.NewIntegration(cfg.ProxyURL)
	return nil
}

func (a *App) Decision(input DecisionRequest) (relay.Decision, error) {
	if input.Method == "" {
		input.Method = http.MethodPost
	}
	req, err := http.NewRequest(input.Method, input.Path, nil)
	if err != nil {
		return relay.Decision{}, apperrors.BadRequest("invalid path")
	}
	for key, value := range input.Headers {
		req.Header.Set(key, value)
	}
	return a.relay.Decision(req), nil
}

func (a *App) HTTPHandler(staticDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("GET /api/status", a.handleStatus)
	mux.HandleFunc("GET /api/diagnostics", a.handleDiagnostics)
	mux.HandleFunc("GET /api/setup/status", a.handleSetupStatus)
	mux.HandleFunc("POST /api/setup/prepare", a.handlePrepareSetup)
	mux.HandleFunc("GET /api/config", a.handleGetConfig)
	mux.HandleFunc("PUT /api/config", a.handleSaveConfig)
	mux.HandleFunc("POST /api/adapters", a.handleUpsertAdapter)
	mux.HandleFunc("POST /api/adapters/import/preview", a.handlePreviewAdapterImport)
	mux.HandleFunc("POST /api/adapters/import", a.handleImportAdapters)
	mux.HandleFunc("DELETE /api/adapters/", a.handleDeleteAdapter)
	mux.HandleFunc("POST /api/proxy/start", a.handleStartProxy)
	mux.HandleFunc("POST /api/proxy/stop", a.handleStopProxy)
	mux.HandleFunc("GET /api/certs/ca", a.handleCAInfo)
	mux.HandleFunc("GET /api/certs/install-plan", a.handleCAInstallPlan)
	mux.HandleFunc("GET /api/cursor/plan", a.handleCursorPlan)
	mux.HandleFunc("POST /api/cursor/apply", a.handleCursorApply)
	mux.HandleFunc("POST /api/relay/byok/chat/completions", a.handleBYOK)
	mux.HandleFunc("POST /api/relay/decision", a.handleDecision)

	if staticDir != "" {
		mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	}

	return requestMiddleware(mux)
}

func requestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if origin := r.Header.Get("Origin"); origin == "http://127.0.0.1:5173" || origin == "http://localhost:5173" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Debug("request handled", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func (a *App) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.Status())
}

func (a *App) handleDiagnostics(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.Diagnostics())
}

func (a *App) handleSetupStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.SetupStatus())
}

func (a *App) handlePrepareSetup(w http.ResponseWriter, r *http.Request) {
	status, err := a.PrepareSetup(r.Context())
	if err != nil {
		writeError(w, apperrors.Wrap(apperrors.ErrInvalidSystemSetting, "prepare local bridge", http.StatusInternalServerError, err))
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (a *App) handleGetConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.Config())
}

func (a *App) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	var cfg config.UserConfig
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&cfg); err != nil {
		writeError(w, apperrors.BadRequest("invalid JSON config"))
		return
	}
	snapshot, err := a.SaveConfig(cfg)
	if err != nil {
		writeError(w, apperrors.Wrap(apperrors.ErrInvalidSystemSetting, "save config", http.StatusBadRequest, err))
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (a *App) handleUpsertAdapter(w http.ResponseWriter, r *http.Request) {
	var adapter config.ModelAdapter
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&adapter); err != nil {
		writeError(w, apperrors.BadRequest("invalid JSON adapter"))
		return
	}
	snapshot, err := a.UpsertAdapter(adapter)
	if err != nil {
		writeError(w, apperrors.Wrap(apperrors.ErrInvalidSystemSetting, "save adapter", http.StatusBadRequest, err))
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (a *App) handleDeleteAdapter(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/adapters/")
	if id == "" {
		writeError(w, apperrors.BadRequest("adapter id is required"))
		return
	}
	snapshot, err := a.DeleteAdapter(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (a *App) handlePreviewAdapterImport(w http.ResponseWriter, r *http.Request) {
	var body AdapterImportRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&body); err != nil {
		writeError(w, apperrors.BadRequest("invalid adapter import request"))
		return
	}
	preview, err := a.PreviewAdapterImport(body.Source)
	if err != nil {
		writeError(w, apperrors.Wrap(apperrors.ErrInvalidSystemSetting, "preview adapter import", http.StatusBadRequest, err))
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (a *App) handleImportAdapters(w http.ResponseWriter, r *http.Request) {
	var body AdapterImportRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&body); err != nil {
		writeError(w, apperrors.BadRequest("invalid adapter import request"))
		return
	}
	result, err := a.ImportAdapters(body.Source)
	if err != nil {
		writeError(w, apperrors.Wrap(apperrors.ErrInvalidSystemSetting, "import adapters", http.StatusBadRequest, err))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *App) handleStartProxy(w http.ResponseWriter, r *http.Request) {
	status, err := a.StartProxy(r.Context())
	if err != nil {
		writeError(w, apperrors.Wrap(apperrors.ErrInvalidSystemSetting, "start proxy", http.StatusInternalServerError, err))
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (a *App) handleStopProxy(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := a.StopProxy(ctx); err != nil {
		writeError(w, apperrors.Wrap(apperrors.ErrInvalidSystemSetting, "stop proxy", http.StatusInternalServerError, err))
		return
	}
	writeJSON(w, http.StatusOK, a.proxy.Status())
}

func (a *App) handleCAInfo(w http.ResponseWriter, _ *http.Request) {
	info, err := a.CAInfo()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (a *App) handleCAInstallPlan(w http.ResponseWriter, _ *http.Request) {
	plan, err := a.CAInstallPlan()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (a *App) handleCursorPlan(w http.ResponseWriter, _ *http.Request) {
	plan, err := a.CursorPlan()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (a *App) handleCursorApply(w http.ResponseWriter, _ *http.Request) {
	if err := a.ApplyCursorSettings(); err != nil {
		writeError(w, apperrors.Wrap(apperrors.ErrInvalidSystemSetting, "apply Cursor settings", http.StatusInternalServerError, err))
		return
	}
	writeJSON(w, http.StatusOK, a.cursor.Status())
}

func (a *App) handleBYOK(w http.ResponseWriter, r *http.Request) {
	if err := a.relay.BYOK().ServeHTTP(w, r); err != nil {
		writeError(w, err)
	}
}

func (a *App) handleDecision(w http.ResponseWriter, r *http.Request) {
	var body DecisionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, apperrors.BadRequest("invalid decision request"))
		return
	}
	decision, err := a.Decision(body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, decision)
}

func runningState(running bool) string {
	if running {
		return "running"
	}
	return "stopped"
}

func certState(err error) string {
	if err != nil {
		return "error"
	}
	return "ready"
}

func certDetail(info certs.CAInfo, err error) string {
	if err != nil {
		return err.Error()
	}
	return "CA certificate: " + info.CertPath
}

func cursorState(plan cursor.SettingsPlan, err error) string {
	if err != nil {
		return "error"
	}
	if plan.Applied {
		return "applied"
	}
	if !plan.Exists {
		return "missing"
	}
	return "pending"
}

func cursorDetail(plan cursor.SettingsPlan, err error) string {
	if err != nil {
		return err.Error()
	}
	if plan.Applied {
		return "Cursor settings already point to " + plan.ProxyURL
	}
	if !plan.Exists {
		return "Settings file will be created at " + plan.SettingsPath
	}
	if plan.Current.HTTPProxy == "" {
		return "Cursor settings file exists but http.proxy is not set"
	}
	return "Current Cursor proxy is " + plan.Current.HTTPProxy
}

func adapterState(count int) string {
	if count == 0 {
		return "empty"
	}
	return "configured"
}

func adapterDetail(count int) string {
	if count == 0 {
		return "No enabled model route is available until an adapter is configured"
	}
	if count == 1 {
		return "1 adapter configured"
	}
	return strconv.Itoa(count) + " adapters configured"
}

func (a *App) setupStatus(warnings []string) SetupStatus {
	appStatus := a.Status()
	cursorPlan, err := a.CursorPlan()
	if err != nil {
		warnings = append(warnings, "读取 Cursor 设置失败: "+err.Error())
	}
	enabledAdapters := countEnabledAdapters(appStatus.Config.ModelAdapters)
	nextActions := []string{}
	if enabledAdapters == 0 {
		nextActions = append(nextActions, "配置 AI 模型")
	}
	if !appStatus.Proxy.Running {
		nextActions = append(nextActions, "启动本地桥接")
	}
	if err == nil && !cursorPlan.Applied {
		nextActions = append(nextActions, "应用 Cursor 设置")
	}
	return SetupStatus{
		Ready:           enabledAdapters > 0 && appStatus.Proxy.Running && err == nil && cursorPlan.Applied,
		ModelConfigured: enabledAdapters > 0,
		EnabledAdapters: enabledAdapters,
		Proxy:           appStatus.Proxy,
		Cursor:          cursorPlan,
		Warnings:        warnings,
		NextActions:     nextActions,
	}
}

func countEnabledAdapters(adapters []config.ModelAdapter) int {
	count := 0
	for _, adapter := range adapters {
		if adapter.Enabled {
			count++
		}
	}
	return count
}

func allHealthy(items []DiagnosticItem) bool {
	for _, item := range items {
		if !item.Healthy {
			return false
		}
	}
	return true
}

func maskAdapterSecrets(adapters []config.ModelAdapter) []config.ModelAdapter {
	masked := make([]config.ModelAdapter, len(adapters))
	for i, adapter := range adapters {
		masked[i] = adapter
		if masked[i].APIKey != "" {
			masked[i].APIKey = "********"
		}
	}
	return masked
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, err error) {
	appErr := apperrors.Public(err)
	writeJSON(w, appErr.HTTPStatus, appErr)
}
