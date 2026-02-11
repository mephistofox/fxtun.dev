import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { authApi, profileApi, type User, type LoginRequest, type RegisterRequest } from '@/api/client'

export const useAuthStore = defineStore('auth', () => {
  const router = useRouter()
  const user = ref<User | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const initialized = ref(false)

  const isAuthenticated = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.is_admin ?? false)

  async function init() {
    if (initialized.value) return

    const token = localStorage.getItem('accessToken')
    if (token) {
      try {
        const response = await profileApi.get()
        user.value = response.data.user
      } catch {
        localStorage.removeItem('accessToken')
        localStorage.removeItem('refreshToken')
      }
    }
    initialized.value = true
  }

  async function login(data: LoginRequest) {
    loading.value = true
    error.value = null

    try {
      const response = await authApi.login(data)
      localStorage.setItem('accessToken', response.data.access_token)
      localStorage.setItem('refreshToken', response.data.refresh_token)
      user.value = response.data.user
      const redirect = router.currentRoute.value.query.redirect as string | undefined
      const safeRedirect = redirect && redirect.startsWith('/') && !redirect.startsWith('//') ? redirect : undefined
      router.push(safeRedirect || { name: 'dashboard' })
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      error.value = err.response?.data?.error || 'Login failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function register(data: RegisterRequest) {
    loading.value = true
    error.value = null

    try {
      const response = await authApi.register(data)
      localStorage.setItem('accessToken', response.data.access_token)
      localStorage.setItem('refreshToken', response.data.refresh_token)
      user.value = response.data.user
      router.push({ name: 'dashboard' })
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      error.value = err.response?.data?.error || 'Registration failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function logout() {
    try {
      await authApi.logout()
    } catch {
      // Ignore errors
    }

    localStorage.removeItem('accessToken')
    localStorage.removeItem('refreshToken')
    user.value = null
    router.push({ name: 'login' })
  }

  async function refreshProfile() {
    try {
      const response = await profileApi.get()
      user.value = response.data.user
    } catch {
      // Ignore errors
    }
  }

  return {
    user,
    loading,
    error,
    initialized,
    isAuthenticated,
    isAdmin,
    init,
    login,
    register,
    logout,
    refreshProfile,
  }
})
