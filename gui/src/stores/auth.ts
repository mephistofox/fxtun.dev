import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as AuthService from '@/wailsjs/wailsjs/go/gui/AuthService'
import { gui } from '@/wailsjs/wailsjs/go/models'

export const useAuthStore = defineStore('auth', () => {
  const isAuthenticated = ref(false)
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const serverAddress = ref('')
  const authMethod = ref<'token' | 'password'>('token')
  const totpRequired = ref(false)
  const isBlocked = ref(false)

  async function checkAuth(): Promise<boolean> {
    try {
      const status = await AuthService.CheckAuth()
      isAuthenticated.value = status.has_credentials
      if (status.server_address) {
        serverAddress.value = status.server_address
      }
      if (status.auth_method) {
        authMethod.value = status.auth_method as 'token' | 'password'
      }
      return status.has_credentials
    } catch (e) {
      console.error('Failed to check auth:', e)
      return false
    }
  }

  async function autoLogin(): Promise<boolean> {
    isLoading.value = true
    error.value = null
    try {
      const response = await AuthService.AutoLogin()
      isAuthenticated.value = response.success
      if (!response.success && response.error) {
        error.value = response.error
      }
      return response.success
    } catch (e) {
      console.error('Auto login failed:', e)
      return false
    } finally {
      isLoading.value = false
    }
  }

  async function loginWithToken(server: string, token: string, remember: boolean): Promise<boolean> {
    isLoading.value = true
    error.value = null

    try {
      const request = new gui.LoginRequest({
        method: 'token',
        server_address: server,
        token: token,
        remember: remember,
      })

      const response = await AuthService.Login(request)

      if (response.success) {
        isAuthenticated.value = true
        serverAddress.value = server
        authMethod.value = 'token'
      } else {
        error.value = response.error || 'Login failed'
      }

      return response.success
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Login failed'
      return false
    } finally {
      isLoading.value = false
    }
  }

  async function loginWithPassword(
    server: string,
    phone: string,
    password: string,
    totpCode: string | undefined,
    remember: boolean
  ): Promise<boolean> {
    isLoading.value = true
    error.value = null

    try {
      const request = new gui.LoginRequest({
        method: 'password',
        server_address: server,
        phone: phone,
        password: password,
        totp_code: totpCode,
        remember: remember,
      })

      const response = await AuthService.Login(request)

      if (response.success) {
        isAuthenticated.value = true
        serverAddress.value = server
        authMethod.value = 'password'
        totpRequired.value = false
      } else if (response.totp_required) {
        totpRequired.value = true
        error.value = null // Clear error, show TOTP field instead
      } else {
        error.value = response.error || 'Login failed'
      }

      return response.success
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Login failed'
      return false
    } finally {
      isLoading.value = false
    }
  }

  async function logout(): Promise<void> {
    try {
      await AuthService.Logout()
      isAuthenticated.value = false
      serverAddress.value = ''
      totpRequired.value = false
      isBlocked.value = false
    } catch (e) {
      console.error('Logout failed:', e)
    }
  }

  function setBlocked(): void {
    isBlocked.value = true
    isAuthenticated.value = false
  }

  function resetTotpRequired(): void {
    totpRequired.value = false
  }

  return {
    isAuthenticated,
    isLoading,
    error,
    serverAddress,
    authMethod,
    totpRequired,
    isBlocked,
    setBlocked,
    checkAuth,
    autoLogin,
    loginWithToken,
    loginWithPassword,
    logout,
    resetTotpRequired,
  }
})
