<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import {
  GetHistory,
  GetHistoryCount,
  ToggleFavorite,
  DeleteHistory,
  ClearHistory,
} from '../../bindings/google-translate/appservice.js'

interface HistoryEntry {
  id: number
  source_text: string
  translated_text: string
  source_lang: string
  target_lang: string
  is_favorite: boolean
  created_at: string
}

const entries = ref<HistoryEntry[]>([])
const keyword = ref('')
const total = ref(0)
const page = ref(0)
const pageSize = 20
const loading = ref(false)

async function loadHistory() {
  loading.value = true
  try {
    const [list, count] = await Promise.all([
      GetHistory(page.value * pageSize, pageSize, keyword.value),
      GetHistoryCount(keyword.value),
    ])
    entries.value = (list || []) as any
    total.value = count as number
  } catch (e) {
    console.error('Failed to load history:', e)
  } finally {
    loading.value = false
  }
}

onMounted(loadHistory)

let searchTimer: any
watch(keyword, () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 0
    loadHistory()
  }, 300)
})

async function toggleFav(id: number) {
  await ToggleFavorite(id)
  loadHistory()
}

async function deleteEntry(id: number) {
  await DeleteHistory(id)
  loadHistory()
}

async function clearAll() {
  if (!confirm('Clear all history?')) return
  await ClearHistory()
  loadHistory()
}

function prevPage() {
  if (page.value > 0) { page.value--; loadHistory() }
}

function nextPage() {
  if ((page.value + 1) * pageSize < total.value) { page.value++; loadHistory() }
}

function copyText(text: string) {
  navigator.clipboard.writeText(text)
}
</script>

<template>
  <div class="history-page">
    <div class="history-header">
      <h2 class="page-title">History</h2>
      <div class="header-actions">
        <input v-model="keyword" placeholder="Search..." class="search-input" />
        <button class="btn btn-danger" @click="clearAll" v-if="entries.length">Clear All</button>
      </div>
    </div>

    <div v-if="loading" class="loading">Loading...</div>

    <div v-else-if="entries.length === 0" class="empty">No history yet</div>

    <div v-else class="table-wrap">
      <table class="history-table">
        <thead>
          <tr>
            <th class="col-src">Source</th>
            <th class="col-tgt">Translation</th>
            <th class="col-lang">Lang</th>
            <th class="col-time">Time</th>
            <th class="col-act"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="entry in entries" :key="entry.id">
            <td class="cell-text" @click="copyText(entry.source_text)" title="Click to copy">
              {{ entry.source_text }}
            </td>
            <td class="cell-text cell-tgt" @click="copyText(entry.translated_text)" title="Click to copy">
              {{ entry.translated_text }}
            </td>
            <td class="cell-lang">{{ entry.source_lang }} &rarr; {{ entry.target_lang }}</td>
            <td class="cell-time">{{ entry.created_at }}</td>
            <td class="cell-act">
              <button class="icon-btn" @click="toggleFav(entry.id)">{{ entry.is_favorite ? '*' : 'o' }}</button>
              <button class="icon-btn del" @click="deleteEntry(entry.id)">x</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="total > pageSize" class="pagination">
      <button class="btn btn-ghost" @click="prevPage" :disabled="page === 0">Prev</button>
      <span class="page-info">{{ page + 1 }} / {{ Math.ceil(total / pageSize) }}</span>
      <button class="btn btn-ghost" @click="nextPage" :disabled="(page + 1) * pageSize >= total">Next</button>
    </div>
  </div>
</template>

<style scoped>
.history-page { display: flex; flex-direction: column; height: 100%; gap: 12px; }
.history-header { display: flex; align-items: center; justify-content: space-between; }
.header-actions { display: flex; gap: 8px; }
.search-input { width: 200px; }

.table-wrap { flex: 1; overflow-y: auto; }
.history-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.history-table th {
  text-align: left; padding: 6px 10px; font-weight: 500; font-size: 11px;
  color: var(--text-secondary); border-bottom: 1px solid var(--border);
  position: sticky; top: 0; background: var(--bg-primary); z-index: 1;
}
.history-table td { padding: 6px 10px; border-bottom: 1px solid var(--border); vertical-align: top; }
.history-table tbody tr:hover { background: var(--bg-secondary); }

.col-src, .col-tgt { width: 35%; }
.col-lang { width: 10%; }
.col-time { width: 12%; }
.col-act { width: 8%; }

.cell-text {
  max-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  cursor: pointer; line-height: 1.4;
}
.cell-text:hover { color: var(--accent); }
.cell-tgt { color: var(--accent); }
.cell-lang { font-size: 11px; color: var(--text-secondary); white-space: nowrap; }
.cell-time { font-size: 11px; color: var(--text-secondary); white-space: nowrap; }
.cell-act { white-space: nowrap; }

.icon-btn { width: 22px; height: 22px; border: none; background: transparent; color: var(--text-secondary); cursor: pointer; border-radius: 4px; font-size: 13px; display: inline-flex; align-items: center; justify-content: center; }
.icon-btn:hover { background: var(--bg-tertiary); color: var(--text-primary); }
.icon-btn.del:hover { color: var(--danger); }

.pagination { display: flex; align-items: center; justify-content: center; gap: 16px; }
.page-info { font-size: 13px; color: var(--text-secondary); }
.loading, .empty { text-align: center; color: var(--text-secondary); padding: 40px; }
</style>
