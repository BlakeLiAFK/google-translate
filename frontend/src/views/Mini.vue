<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import {
  Translate,
  Correct,
  GetLanguages,
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
const ttsId = ref('')
const ttsPaused = ref(false)
const ttsLoading = ref(false)
let currentAudio: HTMLAudioElement | null = null

onMounted(async () => {
  const langs = await GetLanguages()
  languages.value = (langs || []) as any
  window.addEventListener('clipboard-change', onClipboard)
})

onUnmounted(() => {
  window.removeEventListener('clipboard-change', onClipboard)
  stopTTS()
})

function onClipboard(e: any) {
  sourceText.value = e.detail || ''
}

let timer: any
watch(sourceText, () => {
  clearTimeout(timer)
  correction.value = ''
  if (!sourceText.value.trim()) {
    translatedText.value = ''
    return
  }
  timer = setTimeout(doTranslate, 300)
})

watch([sourceLang, targetLang], () => {
  if (sourceText.value.trim()) doTranslate()
})

async function doTranslate() {
  if (!sourceText.value.trim()) return
  loading.value = true
  const text = sourceText.value
  const src = sourceLang.value === 'auto' ? '' : sourceLang.value
  try {
    const [result, corrected] = await Promise.all([
      Translate(text, targetLang.value, src, true),
      Correct(text, src).catch(() => ''),
    ])
    translatedText.value = (result as any)?.translated || ''
    const c = (corrected as string || '').trim()
    if (c && c.toLowerCase() !== text.trim().toLowerCase()) {
      correction.value = c
    } else {
      correction.value = ''
    }
  } catch {
    translatedText.value = ''
  } finally {
    loading.value = false
  }
}

function applyCorrection() {
  sourceText.value = correction.value
  correction.value = ''
}

function stopTTS() {
  if (currentAudio) { currentAudio.pause(); currentAudio = null }
  ttsId.value = ''; ttsPaused.value = false; ttsLoading.value = false
}

async function toggleTTS(text: string, lang: string, id: string) {
  if (!text) return
  if (ttsId.value === id && currentAudio) {
    if (ttsPaused.value) { currentAudio.play(); ttsPaused.value = false }
    else { currentAudio.pause(); ttsPaused.value = true }
    return
  }
  stopTTS()
  ttsLoading.value = true; ttsId.value = id
  try {
    const data = await PlayTTS(text, lang || 'en')
    const audio = new Audio('data:audio/mpeg;base64,' + data)
    currentAudio = audio; ttsLoading.value = false
    audio.onended = () => stopTTS()
    audio.play()
  } catch { stopTTS() }
}

function copyText(text: string) {
  navigator.clipboard.writeText(text)
}
</script>

<template>
  <div class="mini-page">
    <div class="mini-header" style="--wails-draggable: drag">
      <select v-model="sourceLang" class="mini-select">
        <option value="auto">Auto</option>
        <option v-for="l in languages.filter((l: any) => l.code !== 'auto')" :key="l.code" :value="l.code">{{ l.name }}</option>
      </select>
      <span class="mini-arrow">&rarr;</span>
      <select v-model="targetLang" class="mini-select">
        <option v-for="l in languages.filter((l: any) => l.code !== 'auto')" :key="l.code" :value="l.code">{{ l.name }}</option>
      </select>
      <span v-if="loading" class="mini-loading">...</span>
    </div>

    <div class="mini-body">
      <div class="mini-panel mini-source">
        <textarea v-model="sourceText" placeholder="Type or copy text..." rows="3"></textarea>
        <div class="mini-bar">
          <button :class="['mini-btn', { active: ttsId === 'source' }]" @click="toggleTTS(sourceText, sourceLang === 'auto' ? 'en' : sourceLang, 'source')">{{ ttsId === 'source' && !ttsPaused ? '&#9646;&#9646;' : '&#9654;' }}</button>
          <button class="mini-btn" @click="copyText(sourceText)" title="Copy">&#9998;</button>
        </div>
      </div>

      <div v-if="correction" class="mini-correction" @click="applyCorrection">
        Did you mean: <span>{{ correction }}</span>
      </div>

      <div class="mini-panel mini-target">
        <div class="mini-result" @click="copyText(translatedText)">{{ translatedText || '...' }}</div>
        <div class="mini-bar">
          <button :class="['mini-btn', { active: ttsId === 'target' }]" @click="toggleTTS(translatedText, targetLang, 'target')">{{ ttsId === 'target' && !ttsPaused ? '&#9646;&#9646;' : '&#9654;' }}</button>
          <button class="mini-btn" @click="copyText(translatedText)" title="Copy">&#9998;</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mini-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

.mini-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  cursor: grab;
  flex-shrink: 0;
}

.mini-select {
  flex: 1;
  padding: 4px 8px;
  font-size: 12px;
}

.mini-arrow { color: var(--text-secondary); font-size: 14px; flex-shrink: 0; }
.mini-loading { color: var(--accent); font-size: 12px; flex-shrink: 0; }

.mini-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.mini-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.mini-source textarea {
  flex: 1;
  border: none;
  padding: 8px 12px;
  font-size: 13px;
  line-height: 1.5;
  background: transparent;
  color: var(--text-primary);
  resize: none;
}

.mini-target {
  border-top: 1px solid var(--border);
  background: var(--bg-secondary);
}

.mini-result {
  flex: 1;
  padding: 8px 12px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--accent);
  overflow-y: auto;
  cursor: pointer;
}
.mini-result:hover { opacity: 0.8; }

.mini-bar {
  display: flex;
  gap: 4px;
  padding: 4px 8px;
  justify-content: flex-end;
  flex-shrink: 0;
}

.mini-btn {
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.mini-btn:hover { background: var(--bg-tertiary); color: var(--text-primary); }
.mini-btn.active { color: var(--accent); }

.mini-correction {
  padding: 4px 12px;
  font-size: 12px;
  color: var(--text-secondary);
  background: var(--bg-tertiary);
  cursor: pointer;
  flex-shrink: 0;
}
.mini-correction span { color: var(--accent); text-decoration: underline; }
.mini-correction:hover { background: var(--bg-secondary); }
</style>
