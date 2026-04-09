import { ref, onUnmounted } from 'vue'
import type { AdminStats } from '@/api/types'

const MAX_RETRIES = 10

export function useAdminSSE() {
  const stats = ref<AdminStats | null>(null)
  const connected = ref(false)
  let eventSource: EventSource | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let retryCount = 0

  function connect() {
    const token = localStorage.getItem('admin_access_token')
    if (!token) return

    eventSource = new EventSource(`/api/admin/stats/stream?token=${token}`)

    eventSource.addEventListener('stats_update', (event: MessageEvent) => {
      try {
        stats.value = JSON.parse(event.data) as AdminStats
      } catch {
        // ignore parse errors
      }
    })

    eventSource.onopen = () => {
      connected.value = true
      retryCount = 0
    }

    eventSource.onerror = () => {
      connected.value = false
      disconnect()
      retryCount++
      if (retryCount > MAX_RETRIES) return
      const token = localStorage.getItem('admin_access_token')
      if (!token) return
      const delay = Math.min(5000 * Math.pow(1.5, retryCount - 1), 30000)
      reconnectTimer = setTimeout(connect, delay)
    }
  }

  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    eventSource?.close()
    eventSource = null
    connected.value = false
  }

  onUnmounted(disconnect)

  return { stats, connected, connect, disconnect }
}
