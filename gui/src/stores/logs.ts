import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { LogEntry } from '@/types'

export const useLogsStore = defineStore('logs', () => {
  const logs = ref<LogEntry[]>([])
  const maxLogs = 1000
  const filter = ref<'all' | 'debug' | 'info' | 'warn' | 'error'>('all')

  const filteredLogs = computed(() => {
    if (filter.value === 'all') {
      return logs.value
    }
    return logs.value.filter(log => log.level === filter.value)
  })

  function addLog(entry: LogEntry): void {
    logs.value.push(entry)
    if (logs.value.length > maxLogs) {
      logs.value = logs.value.slice(-maxLogs)
    }
  }

  function clearLogs(): void {
    logs.value = []
  }

  function setFilter(newFilter: typeof filter.value): void {
    filter.value = newFilter
  }

  function init(): void {
    // Subscribe to log events from Wails
    // EventsOn('log', (entry: LogEntry) => addLog(entry))
  }

  return {
    logs,
    filteredLogs,
    filter,
    addLog,
    clearLogs,
    setFilter,
    init,
  }
})
