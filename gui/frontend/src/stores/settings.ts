import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as SettingsService from '@/wailsjs/wailsjs/go/gui/SettingsService'
import { setLocale, getLocale } from '@/i18n'

export type Theme = 'light' | 'dark' | 'system'
export type Locale = 'en' | 'ru'

export const useSettingsStore = defineStore('settings', () => {
  const theme = ref<Theme>('system')
  const locale = ref<Locale>(getLocale())
  const minimizeToTray = ref(true)
  const notifications = ref(true)
  const serverAddress = ref('')

  async function init(): Promise<void> {
    try {
      // Load settings from Wails backend
      const savedTheme = await SettingsService.GetTheme()
      if (savedTheme) {
        theme.value = savedTheme as Theme
      }

      minimizeToTray.value = await SettingsService.GetMinimizeToTray()
      notifications.value = await SettingsService.GetNotifications()

      const savedServer = await SettingsService.GetDefaultServerAddress()
      if (savedServer) {
        serverAddress.value = savedServer
      }

      // Apply theme on init
      applyTheme(theme.value)
    } catch (e) {
      console.error('Failed to load settings:', e)
    }
  }

  async function saveTheme(newTheme: Theme): Promise<void> {
    theme.value = newTheme
    applyTheme(newTheme)
    try {
      await SettingsService.SetTheme(newTheme)
    } catch (e) {
      console.error('Failed to save theme:', e)
    }
  }

  async function saveMinimizeToTray(value: boolean): Promise<void> {
    minimizeToTray.value = value
    try {
      await SettingsService.SetMinimizeToTray(value)
    } catch (e) {
      console.error('Failed to save minimizeToTray:', e)
    }
  }

  async function saveNotifications(value: boolean): Promise<void> {
    notifications.value = value
    try {
      await SettingsService.SetNotifications(value)
    } catch (e) {
      console.error('Failed to save notifications:', e)
    }
  }

  async function saveServerAddress(address: string): Promise<void> {
    serverAddress.value = address
    try {
      await SettingsService.SetDefaultServerAddress(address)
    } catch (e) {
      console.error('Failed to save serverAddress:', e)
    }
  }

  function saveLocale(newLocale: Locale): void {
    locale.value = newLocale
    setLocale(newLocale)
  }

  function applyTheme(newTheme: Theme): void {
    const isDark = newTheme === 'dark' ||
      (newTheme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)

    if (isDark) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }

  // Watch for system theme changes
  if (typeof window !== 'undefined') {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (theme.value === 'system') {
        applyTheme('system')
      }
    })
  }

  return {
    theme,
    locale,
    minimizeToTray,
    notifications,
    serverAddress,
    init,
    saveTheme,
    saveLocale,
    saveMinimizeToTray,
    saveNotifications,
    saveServerAddress,
  }
})
