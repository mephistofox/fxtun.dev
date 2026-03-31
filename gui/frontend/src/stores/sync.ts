import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface SyncStatus {
  is_syncing: boolean
  last_synced?: string
  last_error?: string
}

export const useSyncStore = defineStore('sync', () => {
  const status = ref<SyncStatus>({
    is_syncing: false,
  })
  const isSyncing = computed(() => status.value.is_syncing)
  const lastSynced = computed(() => status.value.last_synced)
  const lastError = computed(() => status.value.last_error)

  async function getStatus(): Promise<void> {
    try {
      // Dynamic import to handle when bindings don't exist yet
      const SyncService = await import('@/wailsjs/wailsjs/go/gui/SyncService')
      const result = await SyncService.GetStatus()
      status.value = result
    } catch (e) {
      console.debug('Sync service not available:', e)
    }
  }

  async function pull(): Promise<boolean> {
    try {
      const SyncService = await import('@/wailsjs/wailsjs/go/gui/SyncService')
      status.value.is_syncing = true
      await SyncService.Pull()
      await getStatus()
      return true
    } catch (e) {
      console.error('Failed to pull:', e)
      status.value.last_error = e instanceof Error ? e.message : 'Pull failed'
      return false
    } finally {
      status.value.is_syncing = false
    }
  }

  async function push(): Promise<boolean> {
    try {
      const SyncService = await import('@/wailsjs/wailsjs/go/gui/SyncService')
      status.value.is_syncing = true
      await SyncService.Push()
      await getStatus()
      return true
    } catch (e) {
      console.error('Failed to push:', e)
      status.value.last_error = e instanceof Error ? e.message : 'Push failed'
      return false
    } finally {
      status.value.is_syncing = false
    }
  }

  function startPolling(): () => void {
    const interval = setInterval(() => {
      getStatus()
    }, 5000)

    return () => clearInterval(interval)
  }

  return {
    status,
    isSyncing,
    lastSynced,
    lastError,
    getStatus,
    pull,
    push,
    startPolling,
  }
})
