import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'system'

function getSystemTheme(): 'light' | 'dark' {
  if (import.meta.env.SSR) return 'dark'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function applyTheme(theme: 'light' | 'dark') {
  if (import.meta.env.SSR) return
  if (theme === 'dark') {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
}

export const useThemeStore = defineStore('theme', () => {
  const saved = import.meta.env.SSR ? null : localStorage.getItem('theme')
  const mode = ref<ThemeMode>((saved as ThemeMode) || 'system')

  function setMode(newMode: ThemeMode) {
    mode.value = newMode
    localStorage.setItem('theme', newMode)
    applyTheme(newMode === 'system' ? getSystemTheme() : newMode)
  }

  function init() {
    applyTheme(mode.value === 'system' ? getSystemTheme() : mode.value)

    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
      if (mode.value === 'system') {
        applyTheme(e.matches ? 'dark' : 'light')
      }
    })
  }

  watch(mode, (newMode) => {
    applyTheme(newMode === 'system' ? getSystemTheme() : newMode)
  })

  return {
    mode,
    setMode,
    init,
  }
})
