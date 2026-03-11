<script setup lang="ts">
import { ref, onMounted } from 'vue'
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
} from '../../bindings/google-translate/appservice.js'

const settings = ref<Record<string, string>>({})
const cacheCount = ref(0)
const httpRunning = ref(false)
const mcpRunning = ref(false)
const loading = ref(true)
const proxyInput = ref('')
const proxySaving = ref(false)
const proxyMsg = ref('')

async function loadSettings() {
  loading.value = true
  try {
    const [s, count, httpStatus, mcpStatus] = await Promise.all([
      GetSettings(),
      GetCacheStats(),
      IsHTTPAPIRunning(),
      IsMCPServerRunning(),
    ])
    settings.value = (s || {}) as Record<string, string>
    proxyInput.value = settings.value.proxy_url || ''
    cacheCount.value = count as number
    httpRunning.value = httpStatus as boolean
    mcpRunning.value = mcpStatus as boolean
  } catch (e) {
    console.error('Failed to load settings:', e)
  } finally {
    loading.value = false
  }
}

onMounted(loadSettings)

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
</style>
