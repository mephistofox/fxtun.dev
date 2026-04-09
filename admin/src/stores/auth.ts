import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/client'
import type { User } from '@/api/types'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const accessToken = ref<string | null>(
    localStorage.getItem('admin_access_token'),
  )
  const refreshTokenValue = ref<string | null>(
    localStorage.getItem('admin_refresh_token'),
  )

  const isAuthenticated = computed(() => !!accessToken.value)
  const isAdmin = computed(() => !!user.value?.is_admin)

  function setTokens(access: string, refresh: string) {
    accessToken.value = access
    refreshTokenValue.value = refresh
    localStorage.setItem('admin_access_token', access)
    localStorage.setItem('admin_refresh_token', refresh)
  }

  function clearTokens() {
    accessToken.value = null
    refreshTokenValue.value = null
    user.value = null
    localStorage.removeItem('admin_access_token')
    localStorage.removeItem('admin_refresh_token')
  }

  async function login(phone: string, password: string, totpCode?: string) {
    const response = await authApi.login(phone, password, totpCode)
    const { access_token, refresh_token, user: userData } = response.data

    if (!userData.is_admin) {
      throw new Error('Access denied: admin privileges required')
    }

    setTokens(access_token, refresh_token)
    user.value = userData
  }

  async function logout() {
    clearTokens()
    router.push('/login')
  }

  async function refreshToken() {
    if (!refreshTokenValue.value) {
      clearTokens()
      return
    }

    try {
      const response = await authApi.refresh(refreshTokenValue.value)
      const { access_token, refresh_token } = response.data
      setTokens(access_token, refresh_token)
    } catch {
      clearTokens()
    }
  }

  async function fetchProfile() {
    try {
      const response = await authApi.profile()
      user.value = response.data.user

      if (!user.value.is_admin) {
        clearTokens()
        throw new Error('Access denied: admin privileges required')
      }
    } catch (err) {
      clearTokens()
      throw err
    }
  }

  async function init() {
    if (!accessToken.value) return

    try {
      await fetchProfile()
    } catch {
      clearTokens()
    }
  }

  return {
    user,
    accessToken,
    isAuthenticated,
    isAdmin,
    login,
    logout,
    refreshToken,
    fetchProfile,
    init,
  }
})
