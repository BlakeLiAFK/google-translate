<script setup lang="ts">
import { ref } from 'vue'
import { TranslateI18nContent } from '../../bindings/google-translate/appservice.js'

const inputContent = ref('')
const targetLangs = ref('zh,ja,ko')
const sourceLang = ref('en')
const format = ref('auto')
const results = ref<Record<string, string>>({})
const loading = ref(false)
const error = ref('')

const formats = [
  { value: 'auto', label: 'Auto Detect' },
  { value: 'json', label: 'JSON' },
  { value: 'ts', label: 'TypeScript' },
  { value: 'md', label: 'Markdown' },
  { value: 'ini', label: 'INI' },
]

async function translateI18n() {
  if (!inputContent.value.trim() || !targetLangs.value.trim()) return

  loading.value = true
  error.value = ''
  results.value = {}

  try {
    const langs = targetLangs.value.split(',').map(s => s.trim()).filter(Boolean)
    const fmt = format.value === 'auto' ? '' : format.value
    const result = await TranslateI18nContent(inputContent.value, langs, sourceLang.value, fmt)
    results.value = (result || {}) as Record<string, string>
  } catch (e: any) {
    error.value = e.message || String(e)
  } finally {
    loading.value = false
  }
}

function copyResult(lang: string) {
  navigator.clipboard.writeText(results.value[lang])
}

const samples: Record<string, string> = {
  json: JSON.stringify({
    "app": { "title": "My Application", "description": "A sample application" },
    "button": { "submit": "Submit", "cancel": "Cancel", "save": "Save Changes" },
    "message": { "welcome": "Welcome back!", "error": "Something went wrong" }
  }, null, 2),
  ts: `export default {
  app: {
    title: "My Application",
    description: "A sample application",
  },
  button: {
    submit: "Submit",
    cancel: "Cancel",
  },
}`,
  md: `# My Application

Welcome to the documentation.

## Getting Started

This guide will help you get started with the application.

- Install dependencies
- Configure settings
- Run the application

## Features

The application supports multiple languages and real-time translation.

> Note: This is a sample document for testing.`,
  ini: `[app]
title = My Application
description = A sample application

[button]
submit = Submit
cancel = Cancel
save = Save Changes

[message]
welcome = Welcome back!
error = Something went wrong`,
}

function loadSample() {
  const fmt = format.value === 'auto' ? 'json' : format.value
  inputContent.value = samples[fmt] || samples.json
  if (format.value === 'auto') format.value = fmt
}
</script>

<template>
  <div class="i18n-page">
    <div class="i18n-header">
      <h2 class="page-title">i18n Tool</h2>
      <button class="btn btn-ghost" @click="loadSample">Load Sample</button>
    </div>

    <div class="config-bar">
      <div class="config-item">
        <label>Format</label>
        <select v-model="format">
          <option v-for="f in formats" :key="f.value" :value="f.value">{{ f.label }}</option>
        </select>
      </div>
      <div class="config-item">
        <label>Source Language</label>
        <input v-model="sourceLang" placeholder="en" />
      </div>
      <div class="config-item">
        <label>Target Languages</label>
        <input v-model="targetLangs" placeholder="zh,ja,ko" />
      </div>
      <button class="btn btn-primary" @click="translateI18n" :disabled="loading || !inputContent.trim()">
        {{ loading ? 'Translating...' : 'Translate' }}
      </button>
    </div>

    <div v-if="error" class="error-msg">{{ error }}</div>

    <div class="i18n-area">
      <div class="json-panel">
        <div class="panel-header">Source ({{ format === 'auto' ? 'Auto' : formats.find(f => f.value === format)?.label }})</div>
        <textarea v-model="inputContent" placeholder="Paste your i18n content here (JSON, TypeScript, Markdown, or INI)..."></textarea>
      </div>

      <div class="results-panel" v-if="Object.keys(results).length > 0">
        <div v-for="(content, lang) in results" :key="lang" class="result-item">
          <div class="result-header">
            <span class="result-lang">{{ lang }}</span>
            <button class="btn btn-ghost" @click="copyResult(lang as string)">Copy</button>
          </div>
          <pre class="result-json">{{ content }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.i18n-page { display: flex; flex-direction: column; height: 100%; gap: 12px; }
.i18n-header { display: flex; align-items: center; justify-content: space-between; }
.config-bar { display: flex; gap: 12px; align-items: flex-end; }
.config-item { display: flex; flex-direction: column; gap: 4px; }
.config-item label { font-size: 12px; color: var(--text-secondary); }
.config-item input, .config-item select { width: 160px; }
.error-msg { color: var(--danger); font-size: 13px; padding: 8px 12px; background: rgba(244,67,54,0.1); border-radius: var(--radius); }
.i18n-area { flex: 1; display: flex; gap: 12px; min-height: 0; overflow: hidden; }
.json-panel { flex: 1; display: flex; flex-direction: column; border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
.panel-header { padding: 8px 12px; font-size: 12px; font-weight: 600; color: var(--text-secondary); border-bottom: 1px solid var(--border); background: var(--bg-secondary); }
.json-panel textarea { flex: 1; border: none; padding: 12px; font-family: 'SF Mono','Menlo',monospace; font-size: 13px; line-height: 1.6; background: var(--bg-input); }
.results-panel { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 8px; }
.result-item { border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
.result-header { display: flex; align-items: center; justify-content: space-between; padding: 6px 12px; background: var(--bg-secondary); border-bottom: 1px solid var(--border); }
.result-lang { font-size: 13px; font-weight: 600; color: var(--accent); }
.result-json { padding: 12px; font-family: 'SF Mono','Menlo',monospace; font-size: 12px; line-height: 1.5; background: var(--bg-input); overflow-x: auto; white-space: pre; margin: 0; }
</style>
