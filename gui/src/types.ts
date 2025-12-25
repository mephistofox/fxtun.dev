// Tunnel types
export type TunnelType = 'http' | 'tcp' | 'udp'

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
  localAddr?: string
  subdomain?: string
  remotePort?: number
}

// Bundle types
export interface Bundle {
  id: number
  name: string
  type: TunnelType
  localPort: number
  subdomain?: string
  remotePort?: number
  autoConnect: boolean
  createdAt?: string
  updatedAt?: string
}

// History types
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

// Auth types
export type AuthMethod = 'token' | 'password'

export interface AuthCredentials {
  method: AuthMethod
  serverAddress: string
  token?: string
  phone?: string
  password?: string
  totpCode?: string
  remember: boolean
}

// Settings types
export type Theme = 'light' | 'dark' | 'system'

export interface Settings {
  theme: Theme
  minimizeToTray: boolean
  notifications: boolean
  serverAddress: string
}

// Event types from Go backend
export type ClientEventType =
  | 'connecting'
  | 'connected'
  | 'disconnected'
  | 'reconnecting'
  | 'tunnel_created'
  | 'tunnel_closed'
  | 'tunnel_error'
  | 'error'

export interface ClientEvent {
  type: ClientEventType
  payload?: Record<string, unknown>
}

// Log entry
export interface LogEntry {
  timestamp: string
  level: 'debug' | 'info' | 'warn' | 'error'
  message: string
}
