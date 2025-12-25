import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as TunnelService from '@/wailsjs/wailsjs/go/gui/TunnelService'
import { gui } from '@/wailsjs/wailsjs/go/models'
import { EventsOn, BrowserOpenURL } from '@/wailsjs/wailsjs/runtime/runtime'
import type { TunnelType } from '@/types'

export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected'

export interface TunnelInfo {
  id: string
  name: string
  type: TunnelType
  localPort: number
  remoteAddr?: string
  url?: string
  connected: string
}

export interface TunnelConfig {
  name: string
  type: TunnelType
  localPort: number
  subdomain?: string
  remotePort?: number
}

export const useTunnelsStore = defineStore('tunnels', () => {
  const tunnels = ref<TunnelInfo[]>([])
  const status = ref<ConnectionStatus>('disconnected')
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  const activeTunnels = computed(() => tunnels.value)

  function init() {
    // Subscribe to Wails events
    EventsOn('connected', () => {
      status.value = 'connected'
    })

    EventsOn('disconnected', () => {
      status.value = 'disconnected'
      tunnels.value = []
    })

    EventsOn('tunnel_created', (data: any) => {
      const payload = data.payload || data
      // Check if tunnel already exists (prevent duplicates)
      if (!payload.id || tunnels.value.find(t => t.id === payload.id)) {
        return
      }
      const tunnel: TunnelInfo = {
        id: payload.id,
        name: payload.name,
        type: payload.type as TunnelType,
        localPort: payload.local_port,
        remoteAddr: payload.remote_addr,
        url: payload.url,
        connected: payload.connected,
      }
      tunnels.value.push(tunnel)
    })

    EventsOn('tunnel_closed', (data: any) => {
      const payload = data.payload || data
      tunnels.value = tunnels.value.filter(t => t.id !== payload.tunnel_id)
    })

    EventsOn('error', (data: any) => {
      const payload = data.payload || data
      error.value = payload.error || payload.message || 'An error occurred'
    })

    // Load initial status
    loadStatus()
  }

  async function loadStatus(): Promise<void> {
    try {
      const connectionStatus = await TunnelService.GetConnectionStatus()
      status.value = connectionStatus as ConnectionStatus
      if (status.value === 'connected') {
        await loadTunnels()
      }
    } catch (e) {
      console.error('Failed to load status:', e)
    }
  }

  async function loadTunnels(): Promise<void> {
    try {
      const result = await TunnelService.GetActiveTunnels()
      tunnels.value = result.map((t: gui.TunnelInfo) => ({
        id: t.id,
        name: t.name,
        type: t.type as TunnelType,
        localPort: t.local_port,
        remoteAddr: t.remote_addr,
        url: t.url,
        connected: t.connected,
      }))
    } catch (e) {
      console.error('Failed to load tunnels:', e)
    }
  }

  async function createTunnel(config: TunnelConfig): Promise<TunnelInfo | null> {
    isLoading.value = true
    error.value = null

    try {
      const tunnelConfig = new gui.TunnelConfig({
        name: config.name,
        type: config.type,
        local_port: config.localPort,
        subdomain: config.subdomain,
        remote_port: config.remotePort,
      })

      const result = await TunnelService.CreateTunnel(tunnelConfig)

      const tunnel: TunnelInfo = {
        id: result.id,
        name: result.name,
        type: result.type as TunnelType,
        localPort: result.local_port,
        remoteAddr: result.remote_addr,
        url: result.url,
        connected: result.connected,
      }

      // Tunnel will be added via 'tunnel_created' event, just return the result
      return tunnel
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create tunnel'
      return null
    } finally {
      isLoading.value = false
    }
  }

  async function closeTunnel(tunnelId: string): Promise<boolean> {
    try {
      await TunnelService.CloseTunnel(tunnelId)
      tunnels.value = tunnels.value.filter(t => t.id !== tunnelId)
      return true
    } catch (e) {
      console.error('Failed to close tunnel:', e)
      return false
    }
  }

  async function disconnect(): Promise<void> {
    try {
      await TunnelService.Disconnect()
      status.value = 'disconnected'
      tunnels.value = []
    } catch (e) {
      console.error('Failed to disconnect:', e)
    }
  }

  function openUrl(url: string): void {
    if (url) {
      BrowserOpenURL(url)
    }
  }

  return {
    tunnels,
    activeTunnels,
    status,
    isLoading,
    error,
    init,
    loadStatus,
    loadTunnels,
    createTunnel,
    closeTunnel,
    disconnect,
    openUrl,
  }
})
