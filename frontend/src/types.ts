export type AdapterType = 'openai' | 'anthropic'

export interface ModelAdapter {
  id: string
  displayName: string
  type: AdapterType
  baseURL: string
  apiKey: string
  modelID: string
  enabled: boolean
}

export interface RuntimeConfigSnapshot {
  baseURL: string
  licenseCodeConfigured: boolean
  proxyURL: string
  observabilityLogEnabled: boolean
  modelAdapters: ModelAdapter[]
}

export interface UserConfig {
  baseURL: string
  licenseCode: string
  proxyURL: string
  observabilityLogEnabled: boolean
  modelAdapters: ModelAdapter[]
}

export interface AdapterUpsertReport {
  imported: number
  updated: number
}

export interface AdapterImportResponse {
  sourceType: string
  adapters: ModelAdapter[]
  warnings: string[]
  report?: AdapterUpsertReport
  config?: RuntimeConfigSnapshot
}

export interface ProxyStatus {
  addr: string
  running: boolean
}

export interface AppStatus {
  health: string
  configPath: string
  dataDir: string
  proxy: ProxyStatus
  config: RuntimeConfigSnapshot
  cursor: Record<string, unknown>
}

export interface SetupStatus {
  ready: boolean
  modelConfigured: boolean
  enabledAdapters: number
  proxy: ProxyStatus
  cursor: CursorPlan
  warnings: string[]
  nextActions: string[]
}

export interface DiagnosticItem {
  id: string
  label: string
  state: string
  detail: string
  healthy: boolean
}

export interface Diagnostics {
  ready: boolean
  items: DiagnosticItem[]
  logs: {
    runUsage: string
    channelCalls: string
  }
  nextActions: string[]
}

export interface CAInfo {
  certPath: string
  keyPath: string
  fingerprint: string
  notAfter: string
}

export interface InstallPlan {
  supported: boolean
  platform: string
  commands: string[]
  note: string
}

export interface CursorPlan {
  supported: boolean
  settingsPath: string
  proxyURL: string
  changes: {
    'http.proxy': string
    'http.proxyStrictSSL': boolean
  }
  current: {
    'http.proxy': string
    'http.proxyStrictSSL': boolean
  }
  exists: boolean
  applied: boolean
  warnings: string[]
}

export interface ApiError {
  code: string
  message: string
  status: number
}
