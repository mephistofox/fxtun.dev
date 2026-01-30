import { createRouter, createWebHashHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

let authInitialized = false

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/dashboard',
  },
  {
    path: '/auth',
    name: 'auth',
    component: () => import('./views/AuthView.vue'),
    meta: { requiresGuest: true },
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: () => import('./views/DashboardView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/inspect/:tunnelId',
    name: 'inspect',
    component: () => import('./views/InspectView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/bundles',
    name: 'bundles',
    component: () => import('./views/BundlesView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/domains',
    name: 'domains',
    component: () => import('./views/DomainsView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/history',
    name: 'history',
    component: () => import('./views/HistoryView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('./views/SettingsView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/logs',
    name: 'logs',
    component: () => import('./views/LogsView.vue'),
    meta: { requiresAuth: true },
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

// Navigation guard for authentication
router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore()

  // Initialize auth state and auto-login on first navigation
  if (!authInitialized) {
    const hasCredentials = await authStore.checkAuth()
    if (hasCredentials) {
      // Try to connect with saved credentials
      const success = await authStore.autoLogin()
      if (!success) {
        // Auto-login failed, clear auth state
        authStore.isAuthenticated = false
      }
    }
    authInitialized = true
  }

  // Check route requirements
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    // Redirect to auth page if not authenticated
    next({ name: 'auth' })
  } else if (to.meta.requiresGuest && authStore.isAuthenticated) {
    // Redirect to dashboard if already authenticated
    next({ name: 'dashboard' })
  } else {
    next()
  }
})

export default router
