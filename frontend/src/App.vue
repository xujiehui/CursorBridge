<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { api } from './api'
import type {
  AdapterImportResponse,
  AppStatus,
  CAInfo,
  CursorPlan,
  Diagnostics,
  InstallPlan,
  ModelAdapter,
  SetupStatus,
  UserConfig
} from './types'

const status = ref<AppStatus | null>(null)
const diagnostics = ref<Diagnostics | null>(null)
const setup = ref<SetupStatus | null>(null)
const caInfo = ref<CAInfo | null>(null)
const installPlan = ref<InstallPlan | null>(null)
const cursorPlan = ref<CursorPlan | null>(null)
const routePreview = ref<Record<string, unknown> | null>(null)
const importPreview = ref<AdapterImportResponse | null>(null)
const loading = ref(false)
const notice = ref('')
const error = ref('')
const showAdvanced = ref(false)
const showAdapterOptions = ref(false)
const configMode = ref<'import' | 'manual'>('import')
const lastRefresh = ref<Date | null>(null)
let refreshTimer: number | undefined

const configForm = reactive<UserConfig>({
  baseURL: 'http://127.0.0.1:8080',
  licenseCode: '',
  proxyURL: 'http://127.0.0.1:18080',
  observabilityLogEnabled: false,
  modelAdapters: []
})

const adapterForm = reactive<ModelAdapter>({
  id: '',
  displayName: '',
  type: 'openai',
  baseURL: 'https://api.openai.com/v1',
  apiKey: '',
  modelID: '',
  enabled: true
})

const routeForm = reactive({
  model: 'byok/gpt-4o',
  path: '/v1/chat/completions',
  rawTarget: ''
})

const importForm = reactive({
  source: ''
})

const adapters = computed(() => status.value?.config.modelAdapters ?? [])
const enabledAdapters = computed(() => adapters.value.filter((adapter) => adapter.enabled))
const readyLabel = computed(() => (setup.value?.ready ? '已就绪' : '待准备'))
const setupWarnings = computed(() => setup.value?.warnings ?? [])
const setupActions = computed(() => setup.value?.nextActions ?? [])
const importAdaptersPreview = computed(() => importPreview.value?.adapters ?? [])
const importWarnings = computed(() => importPreview.value?.warnings ?? [])
const installCommands = computed(() => installPlan.value?.commands ?? [])
const cursorWarnings = computed(() => cursorPlan.value?.warnings ?? [])
const diagnosticItems = computed(() => diagnostics.value?.items ?? [])
const formattedRoutePreview = computed(() =>
  routePreview.value ? JSON.stringify(routePreview.value, null, 2) : ''
)
const currentAdapter = computed(() => enabledAdapters.value[0] ?? adapters.value[0] ?? null)
const lastRefreshLabel = computed(() =>
  lastRefresh.value ? lastRefresh.value.toLocaleTimeString([], { hour12: false }) : '--'
)
const setupSteps = computed(() => [
  {
    id: 'model',
    label: '模型',
    value: enabledAdapters.value.length ? `${enabledAdapters.value.length} 个` : '未配置',
    done: Boolean(setup.value?.modelConfigured || enabledAdapters.value.length)
  },
  {
    id: 'bridge',
    label: '桥接',
    value: setup.value?.proxy.running ? '运行中' : '未启动',
    done: Boolean(setup.value?.proxy.running)
  },
  {
    id: 'cursor',
    label: 'Cursor',
    value: setup.value?.cursor.applied ? '已应用' : '待应用',
    done: Boolean(setup.value?.cursor.applied)
  }
])
const primaryAdapterLabel = computed(() => currentAdapter.value?.displayName ?? '未选择')

function syncForm(next: AppStatus) {
  configForm.baseURL = next.config.baseURL
  configForm.proxyURL = next.config.proxyURL
  configForm.licenseCode = next.config.licenseCodeConfigured ? '********' : ''
  configForm.observabilityLogEnabled = next.config.observabilityLogEnabled
  configForm.modelAdapters = next.config.modelAdapters
}

async function refresh(silent = false) {
  if (!silent) {
    loading.value = true
  }
  error.value = ''
  try {
    const [nextStatus, nextSetup, nextDiagnostics, nextCAInfo, nextInstallPlan, nextCursorPlan] =
      await Promise.all([
        api.status(),
        api.setupStatus(),
        api.diagnostics(),
        api.caInfo(),
        api.installPlan(),
        api.cursorPlan()
      ])
    status.value = nextStatus
    setup.value = nextSetup
    diagnostics.value = nextDiagnostics
    caInfo.value = nextCAInfo
    installPlan.value = nextInstallPlan
    cursorPlan.value = nextCursorPlan
    syncForm(nextStatus)
    lastRefresh.value = new Date()
  } catch (err) {
    error.value = toMessage(err)
  } finally {
    loading.value = false
  }
}

async function saveAdapter() {
  await action('', async () => {
    const modelID = stripByok(adapterForm.modelID)
    const adapter = {
      ...adapterForm,
      id: adapterForm.id.trim() || adapterIDPlaceholder(),
      displayName: adapterForm.displayName.trim() || modelID,
      modelID
    }
    await api.upsertAdapter(adapter)
    const model = modelID ? `byok/${modelID}` : routeForm.model
    setup.value = await api.prepareSetup()
    notice.value = setup.value.ready ? '模型配置已保存，本地桥接已准备好' : '模型配置已保存'
    resetAdapter()
    routeForm.model = model
    await refresh(true)
  })
}

async function previewAdapterImport() {
  await action('导入预览已更新', async () => {
    importPreview.value = await api.previewAdapterImport(importForm.source)
  })
}

async function importAdaptersFromSource() {
  await action('', async () => {
    const result = await api.importAdapters(importForm.source)
    importPreview.value = result
    const imported = result.report?.imported ?? 0
    const updated = result.report?.updated ?? 0
    const first = result.adapters[0]
    if (first) {
      routeForm.model = `byok/${first.modelID}`
    }
    setup.value = await api.prepareSetup()
    notice.value = `已导入 ${imported} 个，更新 ${updated} 个模型配置`
    await refresh(true)
  })
}

async function prepareLocalBridge() {
  await action('', async () => {
    setup.value = await api.prepareSetup()
    notice.value = setup.value.ready ? '本地桥接已准备好' : '已尝试准备本地桥接，请查看高级状态'
    await refresh(true)
  })
}

async function saveConfig() {
  await action('高级配置已保存', async () => {
    await api.saveConfig({ ...configForm, modelAdapters: status.value?.config.modelAdapters ?? [] })
    setup.value = await api.prepareSetup()
    await refresh(true)
  })
}

async function toggleProxy() {
  await action(status.value?.proxy.running ? '桥接已停止' : '桥接已启动', async () => {
    if (status.value?.proxy.running) {
      await api.stopProxy()
    } else {
      await api.startProxy()
    }
    await refresh(true)
  })
}

async function removeAdapter(id: string) {
  await action('模型配置已删除', async () => {
    await api.deleteAdapter(id)
    await refresh(true)
  })
}

async function applyCursorSettings() {
  await action('Cursor 设置已写入', async () => {
    await api.applyCursor()
    await refresh(true)
  })
}

async function previewRoute(showNotice = true) {
  await action(showNotice ? '路由预览已更新' : '', async () => {
    const headers: Record<string, string> = {
      'x-cursor-model': routeForm.model
    }
    if (routeForm.rawTarget.trim()) {
      headers['x-raw-cursor-server-url'] = routeForm.rawTarget.trim()
    }
    routePreview.value = await api.decisionWithInput({
      method: 'POST',
      path: routeForm.path || '/v1/chat/completions',
      headers
    })
  })
}

async function copyText(text: string, message: string) {
  await action(message, async () => {
    await navigator.clipboard.writeText(text)
  })
}

async function action(message: string, work: () => Promise<void>) {
  loading.value = true
  notice.value = ''
  error.value = ''
  try {
    await work()
    if (message) {
      notice.value = message
    }
  } catch (err) {
    error.value = toMessage(err)
  } finally {
    loading.value = false
  }
}

function editAdapter(adapter: ModelAdapter) {
  Object.assign(adapterForm, adapter, { apiKey: '' })
  routeForm.model = `byok/${adapter.modelID}`
  configMode.value = 'manual'
  showAdapterOptions.value = true
}

function resetAdapter() {
  Object.assign(adapterForm, {
    id: '',
    displayName: '',
    type: 'openai',
    baseURL: 'https://api.openai.com/v1',
    apiKey: '',
    modelID: '',
    enabled: true
  })
  showAdapterOptions.value = false
}

function useImportedAdapter(adapter: ModelAdapter) {
  Object.assign(adapterForm, adapter, { apiKey: '' })
  routeForm.model = `byok/${adapter.modelID}`
  configMode.value = 'manual'
  showAdapterOptions.value = true
}

function clearImportSource() {
  importForm.source = ''
  importPreview.value = null
}

function useAdapterRoute(adapter: ModelAdapter) {
  routeForm.model = `byok/${adapter.modelID}`
  showAdvanced.value = true
  void previewRoute()
}

function toMessage(err: unknown) {
  return err instanceof Error ? err.message : String(err)
}

function stripByok(model: string) {
  return model.trim().replace(/^byok\//, '')
}

function adapterIDPlaceholder() {
  const model = stripByok(adapterForm.modelID)
  const name = adapterForm.displayName || model || 'model'
  return name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'model'
}

function proxyLabel() {
  return status.value?.proxy.running ? '停止桥接' : '启动桥接'
}

onMounted(() => {
  void refresh()
  refreshTimer = window.setInterval(() => {
    void refresh(true)
  }, 5000)
})

onBeforeUnmount(() => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer)
  }
})
</script>

<template>
  <main class="shell">
    <header class="topbar">
      <div class="brand">
        <p class="eyebrow">CursorBridge</p>
        <h1>模型配置</h1>
      </div>
      <div class="topbar-actions">
        <span class="status-chip" :class="setup?.ready ? 'good' : 'pending'">{{ readyLabel }}</span>
        <button class="icon-button" :disabled="loading" title="刷新" aria-label="刷新" @click="refresh()">↻</button>
      </div>
    </header>

    <section class="status-strip" aria-label="准备状态">
      <article v-for="item in setupSteps" :key="item.id" class="status-item" :class="{ complete: item.done }">
        <span>{{ item.label }}</span>
        <strong>{{ item.value }}</strong>
      </article>
      <article class="status-item current">
        <span>当前渠道</span>
        <strong>{{ primaryAdapterLabel }}</strong>
      </article>
    </section>

    <section class="alerts" aria-live="polite">
      <div v-if="notice" class="notice">{{ notice }}</div>
      <div v-if="error" class="error">{{ error }}</div>
      <div v-if="setupWarnings.length" class="warning-list inline">
        <span v-for="item in setupWarnings" :key="item">{{ item }}</span>
      </div>
    </section>

    <section v-if="setupActions.length" class="action-bar">
      <div>
        <strong>待处理</strong>
        <span>{{ setupActions.join(' · ') }}</span>
      </div>
      <button class="secondary" :disabled="loading" @click="prepareLocalBridge">一键准备</button>
    </section>

    <section class="panel setup-panel">
      <div class="panel-head">
        <div>
          <p class="section-kicker">快速配置</p>
          <h2>中转站与模型</h2>
        </div>
        <div class="segmented" role="tablist" aria-label="配置方式">
          <button
            type="button"
            :class="{ active: configMode === 'import' }"
            @click="configMode = 'import'"
          >
            粘贴导入
          </button>
          <button
            type="button"
            :class="{ active: configMode === 'manual' }"
            @click="configMode = 'manual'"
          >
            手动填写
          </button>
        </div>
      </div>

      <div v-if="configMode === 'import'" class="mode-panel">
        <label>
          配置内容
          <textarea
            v-model="importForm.source"
            placeholder='{"baseURL":"https://api.example.com/v1","apiKey":"sk-...","models":["gpt-4o"]}'
            autocomplete="off"
          />
        </label>
        <div class="button-row">
          <button class="secondary" :disabled="loading || !importForm.source.trim()" @click="previewAdapterImport">
            预览
          </button>
          <button :disabled="loading || !importAdaptersPreview.length" @click="importAdaptersFromSource">
            导入并应用
          </button>
          <button class="ghost" :disabled="loading || !importForm.source" @click="clearImportSource">清空</button>
        </div>

        <div v-if="importPreview" class="import-summary">
          <span>来源 {{ importPreview.sourceType }}</span>
          <span>{{ importAdaptersPreview.length }} 个模型</span>
        </div>
        <div v-if="importWarnings.length" class="warning-list">
          <span v-for="item in importWarnings" :key="item">{{ item }}</span>
        </div>
        <div v-if="importAdaptersPreview.length" class="adapter-list compact">
          <article v-for="adapter in importAdaptersPreview" :key="adapter.id" class="adapter-row">
            <div class="adapter-main">
              <div>
                <strong>{{ adapter.displayName }}</strong>
                <span>{{ adapter.type }} · {{ adapter.baseURL }}</span>
              </div>
              <code>byok/{{ adapter.modelID }}</code>
            </div>
            <div class="row-actions">
              <button class="secondary" @click="useImportedAdapter(adapter)">编辑</button>
            </div>
          </article>
        </div>
        <div v-else class="empty">暂无预览</div>
      </div>

      <div v-else class="mode-panel">
        <div class="form-grid">
          <label>
            渠道名称
            <input v-model="adapterForm.displayName" placeholder="我的中转站" autocomplete="off" />
          </label>
          <label>
            类型
            <select v-model="adapterForm.type">
              <option value="openai">OpenAI Compatible</option>
              <option value="anthropic">Anthropic Compatible</option>
            </select>
          </label>
        </div>
        <div class="form-grid">
          <label>
            Base URL
            <input v-model="adapterForm.baseURL" placeholder="https://api.example.com/v1" autocomplete="off" />
          </label>
          <label>
            API Key
            <input v-model="adapterForm.apiKey" type="password" placeholder="sk-..." autocomplete="off" />
          </label>
        </div>
        <label>
          模型 ID
          <input v-model="adapterForm.modelID" placeholder="gpt-4o" autocomplete="off" />
        </label>

        <button type="button" class="option-toggle" @click="showAdapterOptions = !showAdapterOptions">
          {{ showAdapterOptions ? '收起选项' : '更多选项' }}
        </button>
        <div v-if="showAdapterOptions" class="adapter-options">
          <label>
            配置 ID
            <input v-model="adapterForm.id" :placeholder="adapterIDPlaceholder()" autocomplete="off" />
          </label>
          <label class="check">
            <input v-model="adapterForm.enabled" type="checkbox" />
            <span>启用这个渠道</span>
          </label>
        </div>

        <div class="button-row">
          <button :disabled="loading" @click="saveAdapter">保存并应用</button>
          <button class="ghost" :disabled="loading" @click="resetAdapter">清空</button>
        </div>
      </div>
    </section>

    <section class="panel channels-panel">
      <div class="panel-head">
        <div>
          <p class="section-kicker">已配置</p>
          <h2>模型渠道</h2>
        </div>
        <button class="secondary" :disabled="loading" @click="prepareLocalBridge">应用到 Cursor</button>
      </div>
      <div v-if="adapters.length" class="adapter-list">
        <article v-for="adapter in adapters" :key="adapter.id" class="adapter-row">
          <div class="adapter-main">
            <div>
              <strong>{{ adapter.displayName }}</strong>
              <span>{{ adapter.enabled ? '启用' : '停用' }} · {{ adapter.type }} · {{ adapter.baseURL }}</span>
            </div>
            <code>byok/{{ adapter.modelID }}</code>
          </div>
          <div class="row-actions">
            <button class="secondary" @click="editAdapter(adapter)">编辑</button>
            <button class="secondary" @click="useAdapterRoute(adapter)">预览</button>
            <button class="danger" @click="removeAdapter(adapter.id)">删除</button>
          </div>
        </article>
      </div>
      <div v-else class="empty">暂无模型配置</div>
    </section>

    <section class="advanced">
      <button class="ghost advanced-toggle" @click="showAdvanced = !showAdvanced">
        {{ showAdvanced ? '收起高级' : '高级与诊断' }}
      </button>
      <div v-if="showAdvanced" class="advanced-grid">
        <section class="panel">
          <div class="panel-head">
            <div>
              <p class="section-kicker">运行</p>
              <h2>本地桥接</h2>
            </div>
            <button :disabled="loading" @click="toggleProxy">{{ proxyLabel() }}</button>
          </div>
          <div class="metric-list">
            <div>
              <span>监听地址</span>
              <strong>{{ status?.proxy.addr ?? '127.0.0.1:18080' }}</strong>
            </div>
            <div>
              <span>Cursor 设置</span>
              <strong>{{ cursorPlan?.settingsPath }}</strong>
            </div>
            <div>
              <span>当前代理</span>
              <strong>{{ cursorPlan?.current?.['http.proxy'] || '未设置' }}</strong>
            </div>
          </div>
          <div class="button-row">
            <button class="secondary" :disabled="loading" @click="applyCursorSettings">写入 Cursor 设置</button>
            <button
              class="ghost"
              :disabled="!cursorPlan?.settingsPath"
              @click="copyText(cursorPlan?.settingsPath ?? '', 'Cursor 设置路径已复制')"
            >
              复制路径
            </button>
          </div>
          <p v-if="cursorWarnings.length" class="hint">{{ cursorWarnings.join('；') }}</p>
        </section>

        <section class="panel">
          <div class="panel-head">
            <div>
              <p class="section-kicker">高级</p>
              <h2>运行参数</h2>
            </div>
            <button :disabled="loading" @click="saveConfig">保存</button>
          </div>
          <label>
            Bridge Base URL
            <input v-model="configForm.baseURL" autocomplete="off" />
          </label>
          <label>
            Proxy URL
            <input v-model="configForm.proxyURL" autocomplete="off" />
          </label>
          <label class="check">
            <input v-model="configForm.observabilityLogEnabled" type="checkbox" />
            <span>启用 JSONL 观测日志</span>
          </label>
          <div class="path-stack">
            <span>{{ status?.configPath }}</span>
            <span>{{ diagnostics?.logs.runUsage }}</span>
            <span>{{ diagnostics?.logs.channelCalls }}</span>
          </div>
        </section>

        <section class="panel">
          <div class="panel-head">
            <div>
              <p class="section-kicker">证书</p>
              <h2>本地 CA</h2>
            </div>
            <button
              class="secondary"
              :disabled="!caInfo?.certPath"
              @click="copyText(caInfo?.certPath ?? '', 'CA 证书路径已复制')"
            >
              复制证书路径
            </button>
          </div>
          <div class="metric-list">
            <div>
              <span>证书文件</span>
              <strong>{{ caInfo?.certPath }}</strong>
            </div>
            <div>
              <span>指纹</span>
              <strong>{{ caInfo?.fingerprint }}</strong>
            </div>
            <div>
              <span>有效期至</span>
              <strong>{{ caInfo?.notAfter ? new Date(caInfo.notAfter).toLocaleString() : '--' }}</strong>
            </div>
          </div>
          <div class="command-list">
            <code v-for="command in installCommands" :key="command">{{ command }}</code>
          </div>
        </section>

        <section class="panel">
          <div class="panel-head">
            <div>
              <p class="section-kicker">诊断</p>
              <h2>路由预览</h2>
            </div>
            <button class="secondary" :disabled="loading" @click="previewRoute()">预览</button>
          </div>
          <div class="form-grid">
            <label>
              模型
              <input v-model="routeForm.model" placeholder="byok/gpt-4o" autocomplete="off" />
            </label>
            <label>
              路径
              <input v-model="routeForm.path" placeholder="/v1/chat/completions" autocomplete="off" />
            </label>
          </div>
          <label>
            Raw Cursor Target
            <input v-model="routeForm.rawTarget" placeholder="https://api2.cursor.sh" autocomplete="off" />
          </label>
          <pre v-if="routePreview">{{ formattedRoutePreview }}</pre>
          <div v-else class="empty">暂无预览</div>
        </section>

        <section class="panel full-span">
          <div class="panel-head">
            <div>
              <p class="section-kicker">诊断</p>
              <h2>系统状态</h2>
            </div>
          </div>
          <div class="workflow">
            <article v-for="item in diagnosticItems" :key="item.id" class="step" :class="item.healthy ? 'good' : 'pending'">
              <span>{{ item.label }}</span>
              <strong>{{ item.state }}</strong>
              <p>{{ item.detail }}</p>
            </article>
          </div>
        </section>
      </div>
    </section>

    <footer>
      <span>最后刷新 {{ lastRefreshLabel }}</span>
      <span>数据目录 {{ status?.dataDir }}</span>
    </footer>
  </main>
</template>
