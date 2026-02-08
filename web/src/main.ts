import { ViteSSG } from 'vite-ssg'
import { createPinia } from 'pinia'
import App from './App.vue'
import { routes } from './router'
import { i18n } from './i18n'
import './styles.css'

export const createApp = ViteSSG(
  App,
  { routes },
  ({ app, router }) => {
    app.use(createPinia())
    app.use(i18n)

    if (!import.meta.env.SSR) {
      router.beforeEach(async (to, _from, next) => {
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
