import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as HistoryService from '@/wailsjs/wailsjs/go/gui/HistoryService'
import { storage } from '@/wailsjs/wailsjs/go/models'
import { useTunnelsStore } from '@/stores/tunnels'
import type { TunnelType } from '@/types'

export interface HistoryEntry {
  id: number
  bundleId?: number
  bundleName?: string
  tunnelType: TunnelType
  localPort: number
  remoteAddr?: string
  url?: string
  connectedAt: string
  disconnectedAt?: string
  bytesSent: number
  bytesReceived: number
}

export const useHistoryStore = defineStore('history', () => {
  const entries = ref<HistoryEntry[]>([])
  const isLoading = ref(false)
  const totalCount = ref(0)

  async function loadHistory(limit = 50, offset = 0): Promise<void> {
    isLoading.value = true
    try {
      const result = await HistoryService.List(limit, offset)
      entries.value = result.entries.map((e: storage.HistoryEntry) => ({
        id: e.id,
        bundleId: e.bundle_id,
        bundleName: e.bundle_name,
        tunnelType: e.tunnel_type as TunnelType,
        localPort: e.local_port,
        remoteAddr: e.remote_addr,
        url: e.url,
        connectedAt: e.connected_at,
        disconnectedAt: e.disconnected_at,
        bytesSent: e.bytes_sent,
        bytesReceived: e.bytes_received,
      }))
      totalCount.value = result.total
    } catch (e) {
      console.error('Failed to load history:', e)
    } finally {
      isLoading.value = false
    }
  }

  async function getRecent(limit = 5): Promise<HistoryEntry[]> {
    try {
      const result = await HistoryService.GetRecent(limit)
      return result.map((e: storage.HistoryEntry) => ({
        id: e.id,
        bundleId: e.bundle_id,
        bundleName: e.bundle_name,
        tunnelType: e.tunnel_type as TunnelType,
        localPort: e.local_port,
        remoteAddr: e.remote_addr,
        url: e.url,
        connectedAt: e.connected_at,
        disconnectedAt: e.disconnected_at,
        bytesSent: e.bytes_sent,
        bytesReceived: e.bytes_received,
      }))
    } catch (e) {
      console.error('Failed to get recent history:', e)
      return []
    }
  }

  async function clearHistory(): Promise<boolean> {
    try {
      await HistoryService.Clear()
      entries.value = []
      totalCount.value = 0
      return true
    } catch (e) {
      console.error('Failed to clear history:', e)
      return false
    }
  }

  // Verify active entries: if no matching tunnel is running, mark as stale
  function verifyActiveEntries(): void {
    const tunnelsStore = useTunnelsStore()
    for (const entry of entries.value) {
      if (entry.disconnectedAt) continue
      const isRunning = tunnelsStore.tunnels.some(
        t => t.name === entry.bundleName && t.localPort === entry.localPort
      )
      if (!isRunning) {
        // No matching tunnel — mark as disconnected on the frontend
        entry.disconnectedAt = new Date().toISOString()
      }
    }
  }

  // For active entries (no disconnectedAt), pull live traffic from tunnels store
  function getLiveTraffic(entry: HistoryEntry): { bytesSent: number; bytesReceived: number } {
    if (entry.disconnectedAt) {
      return { bytesSent: entry.bytesSent, bytesReceived: entry.bytesReceived }
    }
    const tunnelsStore = useTunnelsStore()
    const tunnel = tunnelsStore.tunnels.find(
      t => t.name === entry.bundleName && t.localPort === entry.localPort
    )
    if (tunnel) {
      return { bytesSent: tunnel.bytesSent, bytesReceived: tunnel.bytesReceived }
    }
    return { bytesSent: entry.bytesSent, bytesReceived: entry.bytesReceived }
  }

  return {
    entries,
    isLoading,
    totalCount,
    loadHistory,
    getRecent,
    clearHistory,
    getLiveTraffic,
    verifyActiveEntries,
  }
})
