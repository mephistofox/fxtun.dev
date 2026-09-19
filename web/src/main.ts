import { ViteSSG } from 'vite-ssg'
import { createPinia } from 'pinia'
import App from './App.vue'
import { routes } from './router'
import { i18n } from './i18n'
import { useAuthStore } from './stores/auth'
import './styles.css'

export const createApp = ViteSSG(
  App,
  { routes },
  ({ app, router }) => {
    app.use(createPinia())
    app.use(i18n)

    // Set locale from route meta — works during both SSG and client
    router.beforeEach((to, _from, next) => {
      if (to.meta.forcedLocale) {
        // @ts-expect-error vue-i18n composition api
        i18n.global.locale.value = to.meta.forcedLocale as 'en' | 'ru'
      }
      next()
    })

    // Routes carry requiresAuth/requiresAdmin meta, but nothing enforced it:
    // the admin views were reachable in the browser and only failed later, on
    // the API call. The API is the real boundary — this keeps the UI honest.
    // Skipped during SSG, where there is no session.
    if (!import.meta.env.SSR) {
      router.beforeEach(async (to) => {
        if (!to.meta.requiresAuth && !to.meta.requiresAdmin) return true
        const auth = useAuthStore()
        if (!auth.isAuthenticated) {
          return { name: 'login', query: { redirect: to.fullPath } }
        }
        if (to.meta.requiresAdmin && !auth.isAdmin) {
          return { name: 'dashboard' }
        }
        return true
      })
    }

    // GA4: track SPA page views
    if (!import.meta.env.SSR) {
      router.afterEach((to) => {
        if (typeof window.gtag === 'function') {
          // Send to whatever property analytics.js configured for this host.
          // Naming one here configured a second container on every navigation:
          // extra weight on load, and fxtun.ru traffic reported into the
          // fxtun.dev property.
          window.gtag('event', 'page_view', {
            page_path: to.fullPath,
            page_title: document.title,
          })
        }
      })
    }

    if (!import.meta.env.SSR) {
      router.beforeEach(async (to, _from, next) => {
        if (to.meta.forcedLocale) {
          const { setLocale } = await import('./i18n')
          setLocale(to.meta.forcedLocale as 'en' | 'ru')
        }

        const { useAuthStore } = await import('./stores/auth')
        const authStore = useAuthStore()

        if (!authStore.initialized) {
          await authStore.init()
        }

        if (to.meta.requiresAuth && !authStore.isAuthenticated) {
          next({ name: 'login', query: { redirect: to.fullPath } })
        } else if (to.meta.requiresGuest && authStore.isAuthenticated) {
          next({ name: 'dashboard' })
        } else if (to.meta.requiresAdmin && !authStore.isAdmin) {
          next({ name: 'dashboard' })
        } else {
          next()
        }
      })
    }
  },
)
