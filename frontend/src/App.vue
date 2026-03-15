<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const isMini = computed(() => router.currentRoute.value.path === '/mini')

const navItems = [
  { path: '/translate', label: 'Translate', icon: 'T' },
  { path: '/history', label: 'History', icon: 'H' },
  { path: '/i18n', label: 'i18n', icon: 'I' },
  { path: '/settings', label: 'Settings', icon: 'S' },
]
</script>

<template>
  <div class="app">
    <!-- macOS 标题栏拖拽区域（迷你窗口不显示） -->
    <div v-if="!isMini" class="titlebar" style="--wails-draggable: drag"></div>

    <div class="layout">
      <!-- 侧边导航（迷你窗口不显示） -->
      <nav v-if="!isMini" class="sidebar">
        <div class="nav-brand"><img src="/icon.png" alt="GT" class="nav-logo" /></div>
        <div class="nav-items">
          <button
            v-for="item in navItems"
            :key="item.path"
            :class="['nav-item', { active: router.currentRoute.value.path === item.path }]"
            @click="router.push(item.path)"
            :title="item.label"
          >
            <span class="nav-icon">{{ item.icon }}</span>
            <span class="nav-label">{{ item.label }}</span>
          </button>
        </div>
      </nav>

      <!-- 主内容区 -->
      <main class="content">
        <router-view v-slot="{ Component }">
          <keep-alive>
            <component :is="Component" />
          </keep-alive>
        </router-view>
      </main>
    </div>
  </div>
</template>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

:root {
  --bg-primary: #1b2636;
  --bg-secondary: #233044;
  --bg-tertiary: #2a3a52;
  --bg-input: #1e2d42;
  --text-primary: #e8edf3;
  --text-secondary: #8899aa;
  --accent: #4a9eff;
  --accent-hover: #3a8eef;
  --border: #2a3a52;
  --success: #4caf50;
  --danger: #f44336;
  --radius: 8px;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Inter', sans-serif;
  background: var(--bg-primary);
  color: var(--text-primary);
  overflow: hidden;
  height: 100vh;
}

#app {
  height: 100vh;
}

.app {
  height: 100vh;
  display: flex;
  flex-direction: column;
}

.titlebar {
  height: 40px;
  flex-shrink: 0;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
}

.layout {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.sidebar {
  width: 72px;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px 0;
  flex-shrink: 0;
}

.nav-brand {
  margin-bottom: 16px;
  display: flex;
  justify-content: center;
}

.nav-logo {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  transform: scale(2);
}

.nav-items {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
  padding: 0 8px;
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 10px 4px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: var(--radius);
  transition: all 0.15s;
  font-size: 10px;
}

.nav-item:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.nav-item.active {
  background: var(--accent);
  color: #fff;
}

.nav-icon {
  font-size: 18px;
  font-weight: 600;
  line-height: 1;
}

.nav-label {
  font-size: 9px;
  letter-spacing: 0.3px;
}

.content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

/* 通用组件样式 */
.btn {
  padding: 8px 16px;
  border: none;
  border-radius: var(--radius);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.15s;
}

.btn-primary {
  background: var(--accent);
  color: #fff;
}

.btn-primary:hover {
  background: var(--accent-hover);
}

.btn-danger {
  background: var(--danger);
  color: #fff;
}

.btn-ghost {
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border);
}

.btn-ghost:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

input, textarea, select {
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--text-primary);
  padding: 8px 12px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.15s;
  font-family: inherit;
}

input:focus, textarea:focus, select:focus {
  border-color: var(--accent);
}

textarea {
  resize: none;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 16px;
}
</style>
