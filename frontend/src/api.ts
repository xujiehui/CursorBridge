import type {
  AdapterImportResponse,
  AppStatus,
  CAInfo,
  CursorPlan,
  Diagnostics,
  InstallPlan,
  ModelAdapter,
  ProxyStatus,
  RuntimeConfigSnapshot,
  SetupStatus,
  UserConfig
} from './types'

const baseURL = import.meta.env.VITE_API_BASE_URL ?? ''
type DesktopService = {
  Status?: () => Promise<AppStatus>
  Diagnostics?: () => Promise<Diagnostics>
  SetupStatus?: () => Promise<SetupStatus>
  PrepareSetup?: () => Promise<SetupStatus>
  Config?: () => Promise<RuntimeConfigSnapshot>
  SaveConfig?: (config: UserConfig) => Promise<RuntimeConfigSnapshot>
  UpsertAdapter?: (adapter: ModelAdapter) => Promise<RuntimeConfigSnapshot>
  DeleteAdapter?: (id: string) => Promise<RuntimeConfigSnapshot>
  PreviewAdapterImport?: (source: string) => Promise<AdapterImportResponse>
  ImportAdapters?: (source: string) => Promise<AdapterImportResponse>
  StartProxy?: () => Promise<ProxyStatus>
  StopProxy?: () => Promise<ProxyStatus>
  CAInfo?: () => Promise<CAInfo>
  CAInstallPlan?: () => Promise<InstallPlan>
  CursorPlan?: () => Promise<CursorPlan>
  ApplyCursorSettings?: () => Promise<Record<string, unknown>>
  Decision?: (input: {
    method: string
    path: string
    headers: Record<string, string>
  }) => Promise<Record<string, unknown>>
}

type WailsRuntime = {
  Services?: {
    Service?: DesktopService
    DesktopService?: DesktopService
    desktop?: {
      Service?: DesktopService
      DesktopService?: DesktopService
    }
  }
}

declare global {
  interface Window {
    wails?: WailsRuntime
  }
}

function desktopService(): DesktopService | undefined {
  return (
    window.wails?.Services?.Service ??
    window.wails?.Services?.DesktopService ??
    window.wails?.Services?.desktop?.Service ??
    window.wails?.Services?.desktop?.DesktopService
  )
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${baseURL}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init.headers ?? {})
    }
  })
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ message: response.statusText }))
    throw new Error(payload.message ?? response.statusText)
  }
  return response.json() as Promise<T>
}

export const api = {
  status: () => desktopService()?.Status?.() ?? request<AppStatus>('/api/status'),
  diagnostics: () => desktopService()?.Diagnostics?.() ?? request<Diagnostics>('/api/diagnostics'),
  setupStatus: () => desktopService()?.SetupStatus?.() ?? request<SetupStatus>('/api/setup/status'),
  prepareSetup: () =>
    desktopService()?.PrepareSetup?.() ?? request<SetupStatus>('/api/setup/prepare', { method: 'POST' }),
  config: () => desktopService()?.Config?.() ?? request<RuntimeConfigSnapshot>('/api/config'),
  saveConfig: (config: UserConfig) =>
    desktopService()?.SaveConfig?.(config) ?? request<RuntimeConfigSnapshot>('/api/config', {
      method: 'PUT',
      body: JSON.stringify(config)
    }),
  upsertAdapter: (adapter: ModelAdapter) =>
    desktopService()?.UpsertAdapter?.(adapter) ?? request<RuntimeConfigSnapshot>('/api/adapters', {
      method: 'POST',
      body: JSON.stringify(adapter)
    }),
  deleteAdapter: (id: string) =>
    desktopService()?.DeleteAdapter?.(id) ?? request<RuntimeConfigSnapshot>(`/api/adapters/${encodeURIComponent(id)}`, {
      method: 'DELETE'
    }),
  previewAdapterImport: (source: string) =>
    desktopService()?.PreviewAdapterImport?.(source) ??
    request<AdapterImportResponse>('/api/adapters/import/preview', {
      method: 'POST',
      body: JSON.stringify({ source })
    }),
  importAdapters: (source: string) =>
    desktopService()?.ImportAdapters?.(source) ??
    request<AdapterImportResponse>('/api/adapters/import', {
      method: 'POST',
      body: JSON.stringify({ source })
    }),
  startProxy: () =>
    desktopService()?.StartProxy?.() ?? request<ProxyStatus>('/api/proxy/start', { method: 'POST' }),
  stopProxy: () =>
    desktopService()?.StopProxy?.() ?? request<ProxyStatus>('/api/proxy/stop', { method: 'POST' }),
  caInfo: () => desktopService()?.CAInfo?.() ?? request<CAInfo>('/api/certs/ca'),
  installPlan: () =>
    desktopService()?.CAInstallPlan?.() ?? request<InstallPlan>('/api/certs/install-plan'),
  cursorPlan: () => desktopService()?.CursorPlan?.() ?? request<CursorPlan>('/api/cursor/plan'),
  applyCursor: () =>
    desktopService()?.ApplyCursorSettings?.() ??
    request<Record<string, unknown>>('/api/cursor/apply', { method: 'POST' }),
  decisionWithInput: (input: { method: string; path: string; headers: Record<string, string> }) =>
    desktopService()?.Decision?.(input) ??
    request<Record<string, unknown>>('/api/relay/decision', {
      method: 'POST',
      body: JSON.stringify(input)
    }),
  decision: (model: string) => {
    const input = {
      method: 'POST',
      path: '/v1/chat/completions',
      headers: {
        'x-cursor-model': model
      }
    }
    return api.decisionWithInput(input)
  }
}
