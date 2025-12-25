import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as HistoryService from '@/wailsjs/wailsjs/go/gui/HistoryService'
import { storage } from '@/wailsjs/wailsjs/go/models'
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

  return {
    entries,
    isLoading,
    totalCount,
    loadHistory,
    getRecent,
    clearHistory,
  }
})
