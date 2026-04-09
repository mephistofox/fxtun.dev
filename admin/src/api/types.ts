// Admin panel TypeScript types
// Mirrors backend DTOs from internal/server/api/dto/ and web/src/api/client.ts

export interface AdminStats {
  active_clients: number
  active_tunnels: number
  http_tunnels: number
  tcp_tunnels: number
  udp_tunnels: number
  total_users: number
}

export interface AdminUser {
  id: number
  phone: string
  email?: string
  display_name: string
  is_admin: boolean
  is_active: boolean
  plan_id: number
  plan?: Plan
  created_at: string
  last_login_at?: string
  avatar_url?: string
  github_id?: number
  google_id?: string
}

export interface UserStats {
  total: number
  active: number
  blocked: number
  admins: number
}

export interface Plan {
  id: number
  slug: string
  name: string
  price: number
  price_rub?: number
  max_tunnels: number
  max_domains: number
  max_custom_domains: number
  max_tokens: number
  max_tunnels_per_token: number
  inspector_enabled: boolean
  is_public: boolean
  is_recommended: boolean
  rate_limit_tcp: number
  rate_limit_udp: number
  rate_limit_http: number
  creem_product_id: string
}

export interface AdminTunnel {
  id: string
  type: string
  name: string
  subdomain?: string
  remote_port?: number
  local_port: number
  url?: string
  client_id: string
  user_id: number
  user_phone: string
  created_at: string
}

export interface AdminSubscription {
  id: number
  user_id: number
  user_phone: string
  user_email: string
  plan_id: number
  plan?: Plan
  next_plan?: Plan
  status: 'pending' | 'active' | 'cancelled' | 'expired'
  recurring: boolean
  current_period_start?: string
  current_period_end?: string
  created_at: string
}

export interface AdminPayment {
  id: number
  user_id: number
  user_phone: string
  user_email: string
  subscription_id?: number
  invoice_id: number
  amount: number
  status: 'pending' | 'success' | 'failed'
  is_recurring: boolean
  created_at: string
}

export interface Payment {
  id: number
  invoice_id: number
  amount: number
  currency: string
  provider: string
  status: 'pending' | 'success' | 'failed'
  is_recurring: boolean
  created_at: string
}

export interface AuditLog {
  id: number
  user_id?: number
  user_phone?: string
  action: string
  details?: Record<string, unknown>
  ip_address: string
  created_at: string
}

export interface CustomDomain {
  id: number
  user_id: number
  domain: string
  target_subdomain: string
  verified: boolean
  verified_at?: string
  created_at: string
  user_phone?: string
  tls_expiry?: string
}

export interface EdgeNode {
  id: number
  node_id: string
  name: string
  region: string
  public_addr: string
  http_addr: string
  status: string
  approved_at?: string
  approved_by?: number
  last_heartbeat_at?: string
  version: string
  metadata: string
  created_at: string
  updated_at: string
}

export interface InviteCode {
  id: number
  code: string
  created_by_user_id?: number
  used_by_user_id?: number
  used_at?: string
  expires_at?: string
  max_uses?: number
  use_count: number
  created_at: string
}

export interface TunnelHistoryEntry {
  id: number
  bundle_name?: string
  tunnel_type: string
  local_port: number
  remote_addr?: string
  url?: string
  connected_at: string
  disconnected_at?: string
  bytes_sent: number
  bytes_received: number
}

export interface TunnelHistoryStats {
  total_connections: number
  total_bytes_sent: number
  total_bytes_received: number
}

export interface AdminUserDetail {
  user: AdminUser
  payments: Payment[]
  subscriptions: AdminSubscription[]
  tunnel_history: TunnelHistoryEntry[]
  tunnel_stats: TunnelHistoryStats | null
  token_count: number
  domain_count: number
}

// Chart data types
export interface ChartDataPoint {
  date: string
  value: number
}

export interface ChartDataResponse {
  points: ChartDataPoint[]
  metric: string
  period: string
}

// Bulk operations
export interface BulkResult {
  success_count: number
  error_count: number
  errors: string[]
}

// Server settings
export interface ServerSettings {
  server: {
    control_port: number
    http_port: number
    [key: string]: unknown
  }
  web: {
    port: number
    cors_origins: string[]
    [key: string]: unknown
  }
  domain: {
    base: string
    [key: string]: unknown
  }
  features: {
    registration_enabled: boolean
    totp_enabled: boolean
    [key: string]: unknown
  }
  [key: string]: unknown
}

// System info
export interface SystemInfo {
  version: string
  go_version: string
  os: string
  arch: string
  num_cpu: number
  goroutines: number
}

// Generic paginated response
export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  limit: number
}

// Auth types
export interface TokenPair {
  access_token: string
  refresh_token: string
}

export interface User {
  id: number
  phone: string
  email?: string
  display_name: string
  is_admin: boolean
  github_id?: number
  google_id?: string
  created_at: string
}

export interface ProfileResponse {
  user: User
  totp_enabled: boolean
  reserved_domains: { id: number; subdomain: string; created_at: string }[]
  max_domains: number
  token_count: number
  tunnel_count: number
  plan?: Plan
}
