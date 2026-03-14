<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import {
  GetSettings,
  SetSetting,
  GetCacheStats,
  ClearCache,
  IsHTTPAPIRunning,
  IsMCPServerRunning,
  StartHTTPAPI,
  StopHTTPAPI,
  StartMCPServer,
  StopMCPServer,
  SetProxy,
  CheckUpdate,
  GetVersion,
  DownloadUpdate,
} from '../../bindings/google-translate/appservice.js'

const settings = ref<Record<string, string>>({})
const cacheCount = ref(0)
const httpRunning = ref(false)
const mcpRunning = ref(false)
const loading = ref(true)
const proxyInput = ref('')
const proxySaving = ref(false)
const proxyMsg = ref('')
const currentVersion = ref('')
const updateChecking = ref(false)
const updateInfo = ref<any>(null)
const updateError = ref('')
const downloading = ref(false)
const downloadDone = ref(false)

async function loadSettings() {
  loading.value = true
  try {
    const [s, count, httpStatus, mcpStatus, ver] = await Promise.all([
      GetSettings(),
      GetCacheStats(),
      IsHTTPAPIRunning(),
      IsMCPServerRunning(),
      GetVersion(),
    ])
    settings.value = (s || {}) as Record<string, string>
    proxyInput.value = settings.value.proxy_url || ''
    cacheCount.value = count as number
    httpRunning.value = httpStatus as boolean
    mcpRunning.value = mcpStatus as boolean
    currentVersion.value = ver as string
  } catch (e) {
    console.error('Failed to load settings:', e)
  } finally {
    loading.value = false
  }
}

// 监听后台自动检查更新的通知
function onUpdateAvailable(e: Event) {
  const info = (e as CustomEvent).detail
  if (info && info.has_update) {
    updateInfo.value = info
  }
}

onMounted(() => {
  loadSettings()
  window.addEventListener('update-available', onUpdateAvailable)
})
onUnmounted(() => {
  window.removeEventListener('update-available', onUpdateAvailable)
})

async function saveSetting(key: string, value: string) {
  await SetSetting(key, value)
  settings.value[key] = value
}

async function toggleHTTP() {
  if (httpRunning.value) {
    await StopHTTPAPI()
  } else {
    await StartHTTPAPI()
  }
  setTimeout(async () => {
    httpRunning.value = await IsHTTPAPIRunning() as boolean
  }, 500)
}

async function toggleMCP() {
  if (mcpRunning.value) {
    await StopMCPServer()
  } else {
    await StartMCPServer()
  }
  setTimeout(async () => {
    mcpRunning.value = await IsMCPServerRunning() as boolean
  }, 500)
}

async function doClearCache() {
  if (!confirm('Clear all translation cache?')) return
  await ClearCache()
  cacheCount.value = 0
}

async function checkForUpdate() {
  updateChecking.value = true
  updateError.value = ''
  updateInfo.value = null
  try {
    const info = await CheckUpdate()
    updateInfo.value = info
  } catch (e: any) {
    updateError.value = e?.message || 'Check failed'
  } finally {
    updateChecking.value = false
  }
}

async function doDownloadUpdate() {
  if (!updateInfo.value?.download_url) return
  downloading.value = true
  updateError.value = ''
  try {
    await DownloadUpdate(updateInfo.value.download_url)
    downloadDone.value = true
  } catch (e: any) {
    updateError.value = e?.message || 'Download failed'
  } finally {
    downloading.value = false
  }
}

async function saveProxy() {
  proxySaving.value = true
  proxyMsg.value = ''
  try {
    await SetProxy(proxyInput.value.trim())
    settings.value.proxy_url = proxyInput.value.trim()
    proxyMsg.value = 'ok'
    setTimeout(() => { proxyMsg.value = '' }, 2000)
  } catch (e: any) {
    proxyMsg.value = e?.message || 'Failed'
  } finally {
    proxySaving.value = false
  }
}
</script>

<template>
  <div class="settings-page">
    <h2 class="page-title">Settings</h2>

    <div v-if="loading" class="loading">Loading...</div>

    <div v-else class="settings-list">
      <div class="settings-group">
        <h3 class="group-title">Services</h3>

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">HTTP API</span>
            <span class="setting-desc">Port: {{ settings.http_port || '9700' }}</span>
          </div>
          <div class="setting-control">
            <input
              :value="settings.http_port"
              @change="(e: any) => saveSetting('http_port', e.target.value)"
              class="port-input"
              placeholder="9700"
            />
            <button :class="['btn', httpRunning ? 'btn-danger' : 'btn-primary']" @click="toggleHTTP">
              {{ httpRunning ? 'Stop' : 'Start' }}
            </button>
            <span :class="['status-dot', { active: httpRunning }]"></span>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">MCP Server (SSE)</span>
            <span class="setting-desc">Port: {{ settings.mcp_port || '9701' }}</span>
          </div>
          <div class="setting-control">
            <input
              :value="settings.mcp_port"
              @change="(e: any) => saveSetting('mcp_port', e.target.value)"
              class="port-input"
              placeholder="9701"
            />
            <button :class="['btn', mcpRunning ? 'btn-danger' : 'btn-primary']" @click="toggleMCP">
              {{ mcpRunning ? 'Stop' : 'Start' }}
            </button>
            <span :class="['status-dot', { active: mcpRunning }]"></span>
          </div>
        </div>
      </div>

      <div class="settings-group">
        <h3 class="group-title">Translation</h3>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Default Target Language</span>
          </div>
          <select :value="settings.target_lang" @change="(e: any) => saveSetting('target_lang', e.target.value)">
            <option value="zh">Chinese (Simplified)</option>
            <option value="en">English</option>
            <option value="ja">Japanese</option>
            <option value="ko">Korean</option>
            <option value="fr">French</option>
            <option value="de">German</option>
            <option value="es">Spanish</option>
          </select>
        </div>
      </div>

      <div class="settings-group">
        <h3 class="group-title">Proxy</h3>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Proxy URL</span>
            <span class="setting-desc">socks5://127.0.0.1:1080 or http://127.0.0.1:7890</span>
          </div>
          <div class="setting-control">
            <input
              v-model="proxyInput"
              class="proxy-input"
              placeholder="socks5://127.0.0.1:1080"
              @keydown.enter="saveProxy"
            />
            <button class="btn btn-primary" @click="saveProxy" :disabled="proxySaving">
              {{ proxySaving ? '...' : 'Apply' }}
            </button>
            <span v-if="proxyMsg" :class="['proxy-msg', proxyMsg === 'ok' ? 'ok' : 'err']">{{ proxyMsg }}</span>
          </div>
        </div>
      </div>

      <div class="settings-group">
        <h3 class="group-title">Cache</h3>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Translation Cache</span>
            <span class="setting-desc">{{ cacheCount }} cached translations</span>
          </div>
          <button class="btn btn-danger" @click="doClearCache">Clear Cache</button>
        </div>
      </div>

      <div class="settings-group">
        <h3 class="group-title">Application</h3>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Clipboard Monitor</span>
            <span class="setting-desc">Auto-translate copied text in mini window</span>
          </div>
          <label class="toggle">
            <input type="checkbox" :checked="settings.clipboard_monitor === 'true'" @change="(e: any) => saveSetting('clipboard_monitor', e.target.checked ? 'true' : 'false')" />
            <span class="toggle-slider"></span>
          </label>
        </div>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Start Minimized</span>
            <span class="setting-desc">Start app minimized to system tray</span>
          </div>
          <label class="toggle">
            <input type="checkbox" :checked="settings.start_minimized === 'true'" @change="(e: any) => saveSetting('start_minimized', e.target.checked ? 'true' : 'false')" />
            <span class="toggle-slider"></span>
          </label>
        </div>
      </div>

      <div class="settings-group">
        <h3 class="group-title">About</h3>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Version</span>
            <span class="setting-desc">{{ currentVersion || 'dev' }}</span>
          </div>
          <div class="setting-control">
            <button class="btn btn-primary" @click="checkForUpdate" :disabled="updateChecking">
              {{ updateChecking ? 'Checking...' : 'Check Update' }}
            </button>
          </div>
        </div>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Auto Check Update</span>
            <span class="setting-desc">Check for updates on startup</span>
          </div>
          <label class="toggle">
            <input type="checkbox" :checked="settings.auto_update === 'true'" @change="(e: any) => saveSetting('auto_update', e.target.checked ? 'true' : 'false')" />
            <span class="toggle-slider"></span>
          </label>
        </div>
        <div v-if="updateInfo" class="update-result">
          <template v-if="updateInfo.has_update">
            <div class="update-available">
              <span class="update-tag">{{ updateInfo.tag_name }}</span>
              <span class="update-hint">New version available!</span>
            </div>
            <div class="update-actions">
              <button
                v-if="!downloadDone && updateInfo.download_url"
                class="btn btn-primary"
                @click="doDownloadUpdate"
                :disabled="downloading"
              >{{ downloading ? 'Downloading...' : 'Download & Install' }}</button>
              <span v-if="downloadDone" class="update-done">Updated! Restart to apply.</span>
              <a :href="updateInfo.html_url" target="_blank" class="btn btn-ghost">Release Page</a>
            </div>
          </template>
          <template v-else>
            <span class="update-latest">Already up to date</span>
          </template>
        </div>
        <div v-if="updateError" class="update-error">{{ updateError }}</div>
      </div>

      <div class="settings-group">
        <h3 class="group-title">MCP Client Config</h3>
        <div class="config-hint">
          <pre>{{ JSON.stringify({ "mcpServers": { "google-translate": { "url": `http://localhost:${settings.mcp_port || '9701'}/sse` } } }, null, 2) }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-page { max-width: 700px; }
.settings-list { display: flex; flex-direction: column; gap: 20px; }
.settings-group { background: var(--bg-secondary); border: 1px solid var(--border); border-radius: var(--radius); padding: 16px; }
.group-title { font-size: 14px; font-weight: 600; color: var(--text-secondary); margin-bottom: 12px; }
.setting-item { display: flex; align-items: center; justify-content: space-between; padding: 10px 0; border-bottom: 1px solid var(--border); }
.setting-item:last-child { border-bottom: none; padding-bottom: 0; }
.setting-info { display: flex; flex-direction: column; gap: 2px; }
.setting-label { font-size: 14px; font-weight: 500; }
.setting-desc { font-size: 12px; color: var(--text-secondary); }
.setting-control { display: flex; align-items: center; gap: 8px; }
.port-input { width: 80px; text-align: center; }
.proxy-input { width: 260px; }
.proxy-msg { font-size: 12px; font-weight: 500; }
.proxy-msg.ok { color: var(--success); }
.proxy-msg.err { color: var(--danger); }
.status-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--text-secondary); }
.status-dot.active { background: var(--success); }
.toggle { position: relative; display: inline-block; width: 44px; height: 24px; }
.toggle input { opacity: 0; width: 0; height: 0; }
.toggle-slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background: var(--bg-tertiary); border-radius: 12px; transition: 0.2s; }
.toggle-slider::before { content: ''; position: absolute; height: 18px; width: 18px; left: 3px; bottom: 3px; background: var(--text-primary); border-radius: 50%; transition: 0.2s; }
.toggle input:checked + .toggle-slider { background: var(--accent); }
.toggle input:checked + .toggle-slider::before { transform: translateX(20px); }
.config-hint pre { background: var(--bg-input); padding: 12px; border-radius: var(--radius); font-family: 'SF Mono','Menlo',monospace; font-size: 12px; line-height: 1.5; overflow-x: auto; }
.loading { text-align: center; color: var(--text-secondary); padding: 40px; }
.update-result { padding: 10px 0; display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.update-available { display: flex; align-items: center; gap: 8px; }
.update-tag { font-weight: 600; color: var(--accent); font-size: 14px; }
.update-hint { font-size: 13px; color: var(--success); }
.update-actions { display: flex; align-items: center; gap: 8px; }
.update-done { font-size: 13px; color: var(--success); font-weight: 500; }
.update-latest { font-size: 13px; color: var(--text-secondary); }
.update-error { font-size: 13px; color: var(--danger); padding: 6px 0; }
</style>
