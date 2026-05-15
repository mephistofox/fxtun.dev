import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const AdminLayout = () => import('@/layouts/AdminLayout.vue')

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/',
    component: AdminLayout,
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'dashboard',
        component: () => import('@/views/DashboardView.vue'),
      },
      {
        path: 'users',
        name: 'users',
        component: () => import('@/views/UsersView.vue'),
      },
      {
        path: 'users/:id',
        name: 'user-detail',
        component: () => import('@/views/UserDetailView.vue'),
      },
      {
        path: 'tunnels',
        name: 'tunnels',
        component: () => import('@/views/TunnelsView.vue'),
      },
      {
        path: 'nodes',
        name: 'nodes',
        component: () => import('@/views/NodesView.vue'),
      },
      {
        path: 'plans',
        name: 'plans',
        component: () => import('@/views/PlansView.vue'),
      },
      {
        path: 'subscriptions',
        name: 'subscriptions',
        component: () => import('@/views/SubscriptionsView.vue'),
      },
      {
        path: 'payments',
        name: 'payments',
        component: () => import('@/views/PaymentsView.vue'),
      },
      {
        path: 'domains',
        name: 'domains',
        component: () => import('@/views/DomainsView.vue'),
      },
      {
        path: 'audit',
        name: 'audit',
        component: () => import('@/views/AuditView.vue'),
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('@/views/SettingsView.vue'),
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

let authInitialized = false

router.beforeEach(async (to) => {
  const authStore = useAuthStore()

  // Initialize auth on first navigation
  if (!authInitialized) {
    try {
      await authStore.init()
      authInitialized = true
    } catch {
      // init failed, will retry on next navigation
    }
  }

  const requiresAuth = to.meta.requiresAuth !== false

  if (requiresAuth && (!authStore.isAuthenticated || !authStore.isAdmin)) {
    return { name: 'login' }
  }

  if (to.name === 'login' && authStore.isAuthenticated && authStore.isAdmin) {
    return { name: 'dashboard' }
  }
})

export default router
