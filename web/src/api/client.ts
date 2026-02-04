import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('accessToken')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor to handle token refresh
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true

      try {
        const refreshToken = localStorage.getItem('refreshToken')
        if (refreshToken) {
          const response = await axios.post('/api/auth/refresh', { refresh_token: refreshToken })
          const { access_token, refresh_token } = response.data

          localStorage.setItem('accessToken', access_token)
          localStorage.setItem('refreshToken', refresh_token)

          originalRequest.headers.Authorization = `Bearer ${access_token}`
          return api(originalRequest)
        }
      } catch {
        localStorage.removeItem('accessToken')
        localStorage.removeItem('refreshToken')
        window.location.href = '/login'
      }
    }

    return Promise.reject(error)
  }
)

export interface LoginRequest {
  phone: string
  password: string
  totp_code?: string
}

export interface RegisterRequest {
  phone: string
  password: string
  invite_code: string
  display_name?: string
}

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
  reserved_domains: Domain[]
  max_domains: number
  token_count: number
  plan?: Plan
}

export interface Tunnel {
  id: string
  type: string
  name: string
  subdomain?: string
  remote_port?: number
  local_port: number
  created_at: string
}

export interface Domain {
  id: number
  subdomain: string
  created_at: string
}

export interface APIToken {
  id: number
  name: string
  allowed_subdomains: string[]
  max_tunnels: number
  last_used_at?: string
  created_at: string
}

export interface CreateTokenRequest {
  name: string
  allowed_subdomains: string[]
  max_tunnels: number
}

export interface CreateTokenResponse {
  token: string
  info: APIToken
}

export interface Download {
  platform: string
  os: string
  arch: string
  size: number
  url: string
  client_type: 'cli' | 'gui'
}

export interface DownloadsResponse {
  clients: Download[]
  cli: Download[]
  gui: Download[]
}

export const authApi = {
  login: (data: LoginRequest) => api.post<TokenPair & { user: User }>('/auth/login', data),
  register: (data: RegisterRequest) => api.post<TokenPair & { user: User }>('/auth/register', data),
  logout: () => api.post('/auth/logout'),
  deviceAuthorize: (sessionId: string) =>
    api.post('/auth/device/authorize', { session_id: sessionId }),
  refresh: (refreshToken: string) => api.post<TokenPair>('/auth/refresh', { refresh_token: refreshToken }),
}

export const profileApi = {
  get: () => api.get<ProfileResponse>('/profile'),
  update: (data: { display_name?: string }) => api.put<User>('/profile', data),
  changePassword: (data: { current_password: string; new_password: string }) =>
    api.put('/profile/password', data),
}

export const totpApi = {
  enable: () => api.post<{ secret: string; qr_code: string }>('/auth/totp/enable'),
  verify: (code: string) => api.post<{ backup_codes: string[] }>('/auth/totp/verify', { code }),
  disable: (code: string) => api.post('/auth/totp/disable', { code }),
}

export const tunnelsApi = {
  list: () => api.get<{ tunnels: Tunnel[] }>('/tunnels'),
  close: (id: string) => api.delete(`/tunnels/${id}`),
}

export const domainsApi = {
  list: () => api.get<{ domains: Domain[]; max_domains: number }>('/domains'),
  reserve: (subdomain: string) => api.post<Domain>('/domains', { subdomain }),
  release: (id: number) => api.delete(`/domains/${id}`),
  check: (subdomain: string) => api.get<{ available: boolean }>(`/domains/check/${subdomain}`),
}

export const tokensApi = {
  list: () => api.get<{ tokens: APIToken[]; max_tokens: number }>('/tokens'),
  create: (data: CreateTokenRequest) => api.post<CreateTokenResponse>('/tokens', data),
  delete: (id: number) => api.delete(`/tokens/${id}`),
}

export const downloadsApi = {
  list: () => api.get<DownloadsResponse>('/downloads'),
}

export const plansApi = {
  listPublic: () => api.get<{ plans: Plan[] }>('/plans/public'),
}

// Custom domains
export interface CustomDomain {
  id: number
  user_id: number
  domain: string
  target_subdomain: string
  verified: boolean
  verified_at?: string
  created_at: string
}

export interface CustomDomainListResponse {
  domains: CustomDomain[]
  total: number
  max_domains: number
  base_domain: string
  server_ip: string
}

export interface VerifyResponse {
  verified: boolean
  error?: string
  expected?: string
}

export const customDomainsApi = {
  list: () => api.get<CustomDomainListResponse>('/custom-domains'),
  add: (domain: string, target_subdomain: string) =>
    api.post<CustomDomain>('/custom-domains', { domain, target_subdomain }),
  delete: (id: number) => api.delete(`/custom-domains/${id}`),
  verify: (id: number) => api.post<VerifyResponse>(`/custom-domains/${id}/verify`),
}

// Admin API types
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
}

export interface InviteCode {
  id: number
  code: string
  used: boolean
  used_at?: string
  expires_at?: string
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

export interface Plan {
  id: number
  slug: string
  name: string
  price: number       // Price in USD
  price_rub?: number  // Price in RUB (calculated on backend, readonly)
  max_tunnels: number
  max_domains: number
  max_custom_domains: number
  max_tokens: number
  max_tunnels_per_token: number
  inspector_enabled: boolean
  is_public: boolean
  is_recommended: boolean
}

// Admin subscription and payment types
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

export interface AdminSubscriptionsResponse {
  subscriptions: AdminSubscription[]
  total: number
  page: number
  limit: number
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

export interface AdminPaymentsResponse {
  payments: AdminPayment[]
  total: number
  page: number
  limit: number
}

export const adminApi = {
  // Stats
  getStats: () => api.get<AdminStats>('/admin/stats'),

  // Users
  listUsers: (page = 1, limit = 20) =>
    api.get<{ users: AdminUser[]; total: number; page: number; limit: number }>('/admin/users', {
      params: { page, limit },
    }),
  updateUser: (id: number, data: { is_admin?: boolean; is_active?: boolean; plan_id?: number }) =>
    api.put<AdminUser>(`/admin/users/${id}`, data),
  deleteUser: (id: number) => api.delete(`/admin/users/${id}`),

  // Invite codes
  listInvites: (page = 1, limit = 20, unused?: boolean) =>
    api.get<{ codes: InviteCode[]; total: number }>('/admin/invite-codes', {
      params: { page, limit, unused: unused ? 'true' : undefined },
    }),
  createInvite: (expiresInDays?: number) =>
    api.post<InviteCode>('/admin/invite-codes', { expires_in_days: expiresInDays }),
  deleteInvite: (id: number) => api.delete(`/admin/invite-codes/${id}`),

  // Audit logs
  listAuditLogs: (page = 1, limit = 20, userId?: number) =>
    api.get<{ logs: AuditLog[]; total: number }>('/admin/audit-logs', {
      params: { page, limit, user_id: userId },
    }),

  // Tunnels
  listTunnels: () => api.get<{ tunnels: AdminTunnel[]; total: number }>('/admin/tunnels'),
  closeTunnel: (id: string) => api.delete(`/admin/tunnels/${id}`),

  // Merge users
  mergeUsers: (primaryId: number, secondaryId: number) =>
    api.post('/admin/users/merge', { primary_user_id: primaryId, secondary_user_id: secondaryId }),

  // Reset password
  resetPassword: (id: number, newPassword: string) =>
    api.post(`/admin/users/${id}/reset-password`, { new_password: newPassword }),

  // Custom domains
  listCustomDomains: (page = 1, limit = 20) =>
    api.get<{ domains: Array<CustomDomain & { user_phone: string; tls_expiry?: string }>; total: number }>('/admin/custom-domains', {
      params: { page, limit },
    }),
  deleteCustomDomain: (id: number) => api.delete(`/admin/custom-domains/${id}`),

  // Plans
  listPlans: () => api.get<{ plans: Plan[]; total: number }>('/admin/plans'),
  createPlan: (data: Omit<Plan, 'id'>) => api.post<Plan>('/admin/plans', data),
  updatePlan: (id: number, data: Partial<Omit<Plan, 'id' | 'slug'>>) => api.put<Plan>(`/admin/plans/${id}`, data),
  deletePlan: (id: number) => api.delete(`/admin/plans/${id}`),

  // Subscriptions
  listSubscriptions: (page = 1, limit = 20) =>
    api.get<AdminSubscriptionsResponse>('/admin/subscriptions', { params: { page, limit } }),
  cancelSubscription: (id: number) =>
    api.post<{ success: boolean; message: string }>(`/admin/subscriptions/${id}/cancel`),
  extendSubscription: (id: number, days: number) =>
    api.post<{ success: boolean; message: string }>(`/admin/subscriptions/${id}/extend`, { days }),

  // Payments
  listPayments: (page = 1, limit = 20) =>
    api.get<AdminPaymentsResponse>('/admin/payments', { params: { page, limit } }),
}

// Inspect API types
export interface ExchangeSummary {
  id: string
  tunnel_id: string
  timestamp: string
  duration_ns: number
  method: string
  path: string
  host: string
  status_code: number
  request_body_size: number
  response_body_size: number
  remote_addr: string
}

export interface CapturedExchange extends ExchangeSummary {
  request_headers: Record<string, string[]>
  request_body: string | null
  response_headers: Record<string, string[]>
  response_body: string | null
}

export interface ExchangeListResponse {
  exchanges: ExchangeSummary[]
  total: number
}

export const inspectApi = {
  list: (tunnelId: string, offset = 0, limit = 50) =>
    api.get<ExchangeListResponse>(`/tunnels/${tunnelId}/inspect`, { params: { offset, limit } }).then(r => r.data),
  get: (tunnelId: string, exchangeId: string) =>
    api.get<CapturedExchange>(`/tunnels/${tunnelId}/inspect/${exchangeId}`).then(r => r.data),
  clear: (tunnelId: string) =>
    api.delete(`/tunnels/${tunnelId}/inspect`).then(r => r.data),
  replay: (tunnelId: string, exchangeId: string) =>
    api.post(`/tunnels/${tunnelId}/inspect/${exchangeId}/replay`).then(r => r.data),
}

// Subscription types
export interface Subscription {
  id: number
  plan_id: number
  plan?: Plan
  next_plan_id?: number
  next_plan?: Plan
  status: 'pending' | 'active' | 'cancelled' | 'expired'
  recurring: boolean
  current_period_start?: string
  current_period_end?: string
  created_at: string
}

export interface SubscriptionResponse {
  subscription: Subscription | null
  has_active: boolean
}

export interface CheckoutResponse {
  payment_url: string
  invoice_id: number
}

export interface Payment {
  id: number
  invoice_id: number
  amount: number
  status: 'pending' | 'success' | 'failed'
  is_recurring: boolean
  created_at: string
}

export interface PaymentsListResponse {
  payments: Payment[]
  total: number
}

export const subscriptionApi = {
  get: () => api.get<SubscriptionResponse>('/subscription'),
  checkout: (planId: number, recurring: boolean) =>
    api.post<CheckoutResponse>('/subscription/checkout', { plan_id: planId, recurring }),
  cancel: () => api.post<{ success: boolean; message: string }>('/subscription/cancel'),
  changePlan: (planId: number) =>
    api.post<{ success: boolean; message: string }>('/subscription/change', { plan_id: planId }),
  getPayments: () => api.get<PaymentsListResponse>('/subscription/payments'),
}

export default api
