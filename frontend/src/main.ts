import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import App from './App.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/translate' },
    { path: '/translate', component: () => import('./views/Translate.vue') },
    { path: '/history', component: () => import('./views/History.vue') },
    { path: '/i18n', component: () => import('./views/I18nTool.vue') },
    { path: '/settings', component: () => import('./views/Settings.vue') },
    { path: '/mini', component: () => import('./views/Mini.vue') },
  ],
})

createApp(App).use(router).mount('#app')
