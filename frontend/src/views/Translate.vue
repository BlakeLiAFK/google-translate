<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import {
  Translate as doTranslate,
  GetLanguages,
  Correct,
  PlayTTS,
} from '../../bindings/google-translate/appservice.js'

interface LanguageItem {
  code: string
  name: string
}

const sourceText = ref('')
const translatedText = ref('')
const sourceLang = ref('auto')
const targetLang = ref('zh')
const languages = ref<LanguageItem[]>([])
const loading = ref(false)
const correction = ref('')
const ttsId = ref('')       // 当前播放的标识：'source' | 'target' | ''
const ttsPaused = ref(false)
const ttsLoading = ref(false)
let currentAudio: HTMLAudioElement | null = null

onMounted(async () => {
  try {
    const langs = await GetLanguages()
    languages.value = langs as any
  } catch (e) {
    console.error('Failed to load languages:', e)
  }
})

async function translate(skipHistory = false) {
  if (!sourceText.value.trim()) return
  loading.value = true
  const text = sourceText.value
  const src = sourceLang.value === 'auto' ? '' : sourceLang.value
  try {
    const [result, corrected] = await Promise.all([
      doTranslate(text, targetLang.value, src, skipHistory),
      Correct(text, src).catch(() => ''),
    ])
    translatedText.value = (result as any)?.translated || ''
    // 比较纠正结果：忽略大小写和首尾空白
    const c = (corrected as string || '').trim()
    const orig = text.trim()
    if (c && c.toLowerCase() !== orig.toLowerCase()) {
      correction.value = c
    } else {
      correction.value = ''
    }
  } catch (e: any) {
    translatedText.value = 'Error: ' + (e.message || e)
    correction.value = ''
  } finally {
    loading.value = false
  }
}

function applyCorrection() {
  sourceText.value = correction.value
  correction.value = ''
}

function swapLangs() {
  if (sourceLang.value === 'auto') return
  const tmp = sourceLang.value
  sourceLang.value = targetLang.value
  targetLang.value = tmp
  const tmpText = sourceText.value
  sourceText.value = translatedText.value
  translatedText.value = tmpText
}

function copyResult() {
  if (translatedText.value) {
    navigator.clipboard.writeText(translatedText.value)
  }
}

function clearAll() {
  sourceText.value = ''
  translatedText.value = ''
  correction.value = ''
}

function stopTTS() {
  if (currentAudio) {
    currentAudio.pause()
    currentAudio = null
  }
  ttsId.value = ''
  ttsPaused.value = false
  ttsLoading.value = false
}

async function toggleTTS(text: string, lang: string, id: string) {
  if (!text) return
  // 同一个音频：切换暂停/继续
  if (ttsId.value === id && currentAudio) {
    if (ttsPaused.value) {
      currentAudio.play()
      ttsPaused.value = false
    } else {
      currentAudio.pause()
      ttsPaused.value = true
    }
    return
  }
  // 不同音频：停止旧的，播放新的
  stopTTS()
  ttsLoading.value = true
  ttsId.value = id
  try {
    const data = await PlayTTS(text, lang || 'en')
    const audio = new Audio('data:audio/mpeg;base64,' + data)
    currentAudio = audio
    ttsLoading.value = false
    audio.onended = () => stopTTS()
    audio.play()
  } catch (e) {
    console.error('TTS failed:', e)
    stopTTS()
  }
}

// 自动翻译：停止输入 500ms 后触发
let debounceTimer: ReturnType<typeof setTimeout> | null = null
function scheduleAutoTranslate() {
  if (debounceTimer) clearTimeout(debounceTimer)
  if (!sourceText.value.trim()) {
    translatedText.value = ''
    return
  }
  debounceTimer = setTimeout(() => translate(true), 500)
}

watch(sourceText, scheduleAutoTranslate)
watch([sourceLang, targetLang], () => {
  if (sourceText.value.trim()) translate(true)
})

onUnmounted(() => {
  if (debounceTimer) clearTimeout(debounceTimer)
  stopTTS()
})
</script>

<template>
  <div class="translate-page">
    <div class="lang-bar">
      <select v-model="sourceLang" class="lang-select">
        <option value="auto">Auto Detect</option>
        <option
          v-for="lang in languages.filter((l: any) => l.code !== 'auto')"
          :key="lang.code"
          :value="lang.code"
        >{{ lang.name }}</option>
      </select>

      <button class="swap-btn" @click="swapLangs" :disabled="sourceLang === 'auto'" title="Swap">
        &#8644;
      </button>

      <select v-model="targetLang" class="lang-select">
        <option
          v-for="lang in languages.filter((l: any) => l.code !== 'auto')"
          :key="lang.code"
          :value="lang.code"
        >{{ lang.name }}</option>
      </select>
    </div>

    <div class="translate-area">
      <div class="text-panel source-panel">
        <textarea
          v-model="sourceText"
          placeholder="Enter text to translate..."
          @keydown.ctrl.enter="translate"
          @keydown.meta.enter="translate"
        ></textarea>
        <div v-if="correction" class="correction-bar" @click="applyCorrection">
          Did you mean: <span class="correction-text">{{ correction }}</span>
        </div>
        <div class="panel-actions">
          <span class="char-count">{{ sourceText.length }}</span>
          <button
            :class="['tts-btn', { active: ttsId === 'source' }]"
            @click="toggleTTS(sourceText, sourceLang === 'auto' ? 'en' : sourceLang, 'source')"
            v-if="sourceText"
            :title="ttsId === 'source' && !ttsPaused ? 'Pause' : 'Play'"
          >{{ ttsLoading && ttsId === 'source' ? '...' : ttsId === 'source' && !ttsPaused ? '&#9646;&#9646;' : '&#9654;' }}</button>
          <button class="btn btn-ghost" @click="clearAll" v-if="sourceText">Clear</button>
        </div>
      </div>

      <div class="text-panel target-panel">
        <textarea
          v-model="translatedText"
          placeholder="Translation will appear here..."
          readonly
        ></textarea>
        <div class="panel-actions">
          <button
            :class="['tts-btn', { active: ttsId === 'target' }]"
            @click="toggleTTS(translatedText, targetLang, 'target')"
            v-if="translatedText"
            :title="ttsId === 'target' && !ttsPaused ? 'Pause' : 'Play'"
          >{{ ttsLoading && ttsId === 'target' ? '...' : ttsId === 'target' && !ttsPaused ? '&#9646;&#9646;' : '&#9654;' }}</button>
          <button class="btn btn-ghost" @click="copyResult" v-if="translatedText">Copy</button>
        </div>
      </div>
    </div>

    <button
      class="btn btn-primary translate-btn"
      @click="translate(false)"
      :disabled="loading || !sourceText.trim()"
    >
      {{ loading ? 'Translating...' : 'Translate' }}
    </button>
  </div>
</template>

<style scoped>
.translate-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 12px;
}

.lang-bar {
  display: flex;
  align-items: center;
  gap: 12px;
}

.lang-select {
  flex: 1;
  padding: 8px 12px;
}

.swap-btn {
  width: 40px;
  height: 36px;
  border: 1px solid var(--border);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border-radius: var(--radius);
  cursor: pointer;
  font-size: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}

.swap-btn:hover:not(:disabled) {
  background: var(--accent);
  color: #fff;
}

.swap-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.translate-area {
  display: flex;
  gap: 12px;
  flex: 1;
  min-height: 0;
}

.text-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
  background: var(--bg-input);
}

.text-panel textarea {
  flex: 1;
  border: none;
  padding: 14px;
  font-size: 15px;
  line-height: 1.6;
  background: transparent;
}

.panel-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding: 6px 10px;
  border-top: 1px solid var(--border);
}

.correction-bar {
  padding: 6px 14px;
  font-size: 13px;
  color: var(--text-secondary);
  background: var(--bg-tertiary);
  cursor: pointer;
  transition: background 0.15s;
}
.correction-bar:hover { background: var(--bg-secondary); }
.correction-text { color: var(--accent); font-weight: 500; text-decoration: underline; }

.char-count {
  font-size: 11px;
  color: var(--text-secondary);
  margin-right: auto;
}

.tts-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}
.tts-btn:hover { background: var(--bg-tertiary); color: var(--accent); }
.tts-btn.active { color: var(--accent); }

.translate-btn {
  padding: 12px;
  font-size: 14px;
  font-weight: 600;
}
</style>
