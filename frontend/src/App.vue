<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { api } from './api'
import type {
  AppStatus,
  CAInfo,
  CursorPlan,
  Diagnostics,
  InstallPlan,
  ModelAdapter,
  UserConfig
} from './types'

const status = ref<AppStatus | null>(null)
const diagnostics = ref<Diagnostics | null>(null)
const caInfo = ref<CAInfo | null>(null)
const installPlan = ref<InstallPlan | null>(null)
const cursorPlan = ref<CursorPlan | null>(null)
const routePreview = ref<Record<string, unknown> | null>(null)
const loading = ref(false)
const notice = ref('')
const error = ref('')
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

const proxyLabel = computed(() => (status.value?.proxy.running ? '停止代理' : '启动代理'))
const readyLabel = computed(() => (diagnostics.value?.ready ? '就绪' : '待配置'))
const adapters = computed(() => status.value?.config.modelAdapters ?? [])
const diagnosticItems = computed(() => diagnostics.value?.items ?? [])
const nextActions = computed(() => diagnostics.value?.nextActions ?? [])
const installCommands = computed(() => installPlan.value?.commands ?? [])
const cursorWarnings = computed(() => cursorPlan.value?.warnings ?? [])
const enabledAdapters = computed(() => adapters.value.filter((adapter) => adapter.enabled).length)
const formattedRoutePreview = computed(() =>
  routePreview.value ? JSON.stringify(routePreview.value, null, 2) : ''
)
const lastRefreshLabel = computed(() =>
  lastRefresh.value ? lastRefresh.value.toLocaleTimeString([], { hour12: false }) : '--'
)

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
    const [nextStatus, nextDiagnostics, nextCAInfo, nextInstallPlan, nextCursorPlan] = await Promise.all([
      api.status(),
      api.diagnostics(),
      api.caInfo(),
      api.installPlan(),
      api.cursorPlan()
    ])
    status.value = nextStatus
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

async function saveConfig() {
  await action('配置已保存', async () => {
    await api.saveConfig({ ...configForm, modelAdapters: status.value?.config.modelAdapters ?? [] })
    await refresh(true)
  })
}

async function toggleProxy() {
  await action(status.value?.proxy.running ? '代理已停止' : '代理已启动', async () => {
    if (status.value?.proxy.running) {
      await api.stopProxy()
    } else {
      await api.startProxy()
    }
    await refresh(true)
  })
}

async function saveAdapter() {
  await action('模型适配器已保存', async () => {
    await api.upsertAdapter({ ...adapterForm })
    const model = adapterForm.modelID ? `byok/${stripByok(adapterForm.modelID)}` : routeForm.model
    resetAdapter()
    routeForm.model = model
    await refresh(true)
    await previewRoute(false)
  })
}

async function removeAdapter(id: string) {
  await action('模型适配器已删除', async () => {
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
}

function useAdapterRoute(adapter: ModelAdapter) {
  routeForm.model = `byok/${adapter.modelID}`
  void previewRoute()
}

function stateClass(healthy?: boolean) {
  return healthy ? 'good' : 'pending'
}

function toMessage(err: unknown) {
  return err instanceof Error ? err.message : String(err)
}

function stripByok(model: string) {
  return model.trim().replace(/^byok\//, '')
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
    <section class="hero">
      <div class="hero-copy">
        <p class="eyebrow">Cursor助手</p>
        <h1>本地 Cursor 桥接工作台</h1>
        <p class="subtle">通过本机代理、显式证书信任和 BYOK 模型适配器，把 Cursor 请求路由到你配置的模型渠道。</p>
      </div>
      <div class="hero-actions">
        <span class="pill" :class="diagnostics?.ready ? 'good' : 'pending'">{{ readyLabel }}</span>
        <button class="icon-button" :disabled="loading" title="刷新" @click="refresh()">↻</button>
      </div>
    </section>

    <div class="alerts">
      <div v-if="notice" class="notice">{{ notice }}</div>
      <div v-if="error" class="error">{{ error }}</div>
    </div>

    <section class="status-grid">
      <article>
        <span>桥接服务</span>
        <strong>{{ status?.health ?? '...' }}</strong>
      </article>
      <article>
        <span>本地代理</span>
        <strong :class="status?.proxy.running ? 'ok' : 'warn'">
          {{ status?.proxy.running ? '运行中' : '未运行' }}
        </strong>
      </article>
      <article>
        <span>监听地址</span>
        <strong>{{ status?.proxy.addr ?? '127.0.0.1:18080' }}</strong>
      </article>
      <article>
        <span>启用适配器</span>
        <strong>{{ enabledAdapters }}/{{ adapters.length }}</strong>
      </article>
    </section>

    <section class="workflow">
      <article
        v-for="item in diagnosticItems"
        :key="item.id"
        class="step"
        :class="stateClass(item.healthy)"
      >
        <span>{{ item.label }}</span>
        <strong>{{ item.state }}</strong>
        <p>{{ item.detail }}</p>
      </article>
    </section>

    <section v-if="nextActions.length" class="next-actions">
      <strong>下一步</strong>
      <span v-for="item in nextActions" :key="item">{{ item }}</span>
    </section>

    <section class="layout">
      <div class="column">
        <section class="panel">
          <div class="panel-head">
            <div>
              <p class="section-kicker">运行</p>
              <h2>代理与 Cursor</h2>
            </div>
            <button :disabled="loading" @click="toggleProxy">{{ proxyLabel }}</button>
          </div>
          <div class="metric-list">
            <div>
              <span>Cursor 设置文件</span>
              <strong>{{ cursorPlan?.settingsPath }}</strong>
            </div>
            <div>
              <span>目标代理</span>
              <strong>{{ cursorPlan?.proxyURL }}</strong>
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
              <p class="section-kicker">配置</p>
              <h2>运行参数</h2>
            </div>
            <button :disabled="loading" @click="saveConfig">保存</button>
          </div>
          <label>
            Base URL
            <input v-model="configForm.baseURL" autocomplete="off" />
          </label>
          <label>
            License Code
            <input v-model="configForm.licenseCode" type="password" autocomplete="off" />
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
          <div v-if="installCommands.length" class="button-row">
            <button class="ghost" @click="copyText(installCommands.join('\n'), '安装命令已复制')">
              复制安装命令
            </button>
          </div>
          <p v-if="installPlan?.note" class="hint">{{ installPlan.note }}</p>
        </section>
      </div>

      <div class="column">
        <section class="panel">
          <div class="panel-head">
            <div>
              <p class="section-kicker">模型</p>
              <h2>BYOK 适配器</h2>
            </div>
            <button :disabled="loading" @click="saveAdapter">保存适配器</button>
          </div>
          <div class="form-grid">
            <label>
              ID
              <input v-model="adapterForm.id" placeholder="openai-main" autocomplete="off" />
            </label>
            <label>
              名称
              <input v-model="adapterForm.displayName" placeholder="GPT-4o" autocomplete="off" />
            </label>
            <label>
              类型
              <select v-model="adapterForm.type">
                <option value="openai">OpenAI</option>
                <option value="anthropic">Anthropic</option>
              </select>
            </label>
            <label>
              模型 ID
              <input v-model="adapterForm.modelID" placeholder="gpt-4o" autocomplete="off" />
            </label>
          </div>
          <label>
            Base URL
            <input v-model="adapterForm.baseURL" autocomplete="off" />
          </label>
          <label>
            API Key
            <input v-model="adapterForm.apiKey" type="password" autocomplete="off" />
          </label>
          <label class="check">
            <input v-model="adapterForm.enabled" type="checkbox" />
            <span>启用适配器</span>
          </label>
          <button class="ghost" @click="resetAdapter">清空表单</button>
        </section>

        <section class="panel">
          <div class="panel-head">
            <div>
              <p class="section-kicker">列表</p>
              <h2>已配置渠道</h2>
            </div>
          </div>
          <div v-if="adapters.length" class="adapter-list">
            <article v-for="adapter in adapters" :key="adapter.id" class="adapter-row">
              <div class="adapter-main">
                <div>
                  <strong>{{ adapter.displayName }}</strong>
                  <span>{{ adapter.enabled ? '启用' : '停用' }} · {{ adapter.type }}</span>
                </div>
                <code>byok/{{ adapter.modelID }}</code>
              </div>
              <div class="row-actions">
                <button class="secondary" @click="useAdapterRoute(adapter)">路由</button>
                <button class="secondary" @click="editAdapter(adapter)">编辑</button>
                <button class="danger" @click="removeAdapter(adapter.id)">删除</button>
              </div>
            </article>
          </div>
          <div v-else class="empty">尚未配置模型适配器</div>
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
          <div v-else class="empty">选择适配器或输入模型后预览路由</div>
        </section>
      </div>
    </section>

    <footer>
      <span>最后刷新 {{ lastRefreshLabel }}</span>
      <span>数据目录 {{ status?.dataDir }}</span>
    </footer>
  </main>
</template>
