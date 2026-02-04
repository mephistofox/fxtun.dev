import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from './stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'landing',
      component: () => import('./views/LandingView.vue'),
      meta: { requiresGuest: true },
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('./views/LoginView.vue'),
      meta: { requiresGuest: true },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('./views/RegisterView.vue'),
      meta: { requiresGuest: true },
    },
    {
      path: '/offer',
      name: 'offer',
      component: () => import('./views/OfferView.vue'),
    },
    {
      path: '/checkout',
      name: 'checkout',
      component: () => import('./views/CheckoutView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/payment/success',
      name: 'payment-success',
      component: () => import('./views/PaymentSuccessView.vue'),
    },
    {
      path: '/payments/success',
      name: 'payments-success',
      component: () => import('./views/PaymentSuccessView.vue'),
    },
    {
      path: '/payment/fail',
      name: 'payment-fail',
      component: () => import('./views/PaymentFailView.vue'),
    },
    {
      path: '/payments/fail',
      name: 'payments-fail',
      component: () => import('./views/PaymentFailView.vue'),
    },
    {
      path: '/auth/callback',
      name: 'auth-callback',
      component: () => import('./views/AuthCallbackView.vue'),
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
      path: '/domains',
      name: 'domains',
      component: () => import('./views/DomainsView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/tokens',
      name: 'tokens',
      component: () => import('./views/TokensView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/downloads',
      name: 'downloads',
      component: () => import('./views/DownloadsView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/profile',
      name: 'profile',
      component: () => import('./views/ProfileView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/auth/cli',
      name: 'cli-auth',
      component: () => import('./views/CliAuthView.vue'),
      meta: { requiresAuth: true },
    },
    // Admin routes
    {
      path: '/admin',
      name: 'admin',
      component: () => import('./views/admin/AdminDashboardView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/admin/users',
      name: 'admin-users',
      component: () => import('./views/admin/AdminUsersView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/admin/invites',
      name: 'admin-invites',
      component: () => import('./views/admin/AdminInvitesView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/admin/tunnels',
      name: 'admin-tunnels',
      component: () => import('./views/admin/AdminTunnelsView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/admin/custom-domains',
      name: 'admin-custom-domains',
      component: () => import('./views/admin/AdminCustomDomainsView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/admin/plans',
      name: 'admin-plans',
      component: () => import('./views/admin/AdminPlansView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/admin/audit',
      name: 'admin-audit',
      component: () => import('./views/admin/AdminAuditView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/admin/subscriptions',
      name: 'admin-subscriptions',
      component: () => import('./views/admin/AdminSubscriptionsView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
  ],
})

router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore()

  // Wait for auth initialization
  if (!authStore.initialized) {
    await authStore.init()
  }

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next({ name: 'login', query: { redirect: to.fullPath } })
  } else if (to.meta.requiresGuest && authStore.isAuthenticated) {
    // Redirect authenticated users from guest pages to dashboard
    next({ name: 'dashboard' })
  } else if (to.meta.requiresAdmin && !authStore.isAdmin) {
    next({ name: 'dashboard' })
  } else {
    next()
  }
})

export default router
