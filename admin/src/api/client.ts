import axios from 'axios'
import type {
  AdminStats,
  AdminUser,
  AdminUserDetail,
  AdminTunnel,
  AdminSubscription,
  AdminPayment,
  AuditLog,
  BulkResult,
  ChartDataResponse,
  CustomDomain,
  EdgeNode,
  InviteCode,
  Plan,
  ProfileResponse,
  ServerSettings,
  SystemInfo,
  TokenPair,
  User,
  UserStats,
} from './types'
import router from '@/router'

const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
})

// Refresh token mutex to prevent concurrent refresh attempts
let isRefreshing = false
let failedQueue: Array<{ resolve: (token: string) => void; reject: (err: unknown) => void }> = []

function processQueue(error: unknown, token: string | null = null) {
  failedQueue.forEach(({ resolve, reject }) => {
    if (error) reject(error)
    else resolve(token!)
  })
  failedQueue = []
}

// Request interceptor: add Authorization header from localStorage
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('admin_access_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error),
)

// Response interceptor: handle 401 by trying refresh, then redirect to login
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true

      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({
            resolve: (token: string) => {
              originalRequest.headers.Authorization = `Bearer ${token}`
              resolve(api(originalRequest))
            },
            reject,
          })
        })
      }

      isRefreshing = true

      try {
        const refreshToken = localStorage.getItem('admin_refresh_token')
        if (refreshToken) {
          const response = await axios.post('/api/auth/refresh', {
            refresh_token: refreshToken,
          })
          const { access_token, refresh_token } = response.data

          localStorage.setItem('admin_access_token', access_token)
          localStorage.setItem('admin_refresh_token', refresh_token)

          processQueue(null, access_token)

          originalRequest.headers.Authorization = `Bearer ${access_token}`
          return api(originalRequest)
        }
      } catch (refreshError) {
        processQueue(refreshError)
        // Refresh failed, clear tokens and redirect
      } finally {
        isRefreshing = false
      }

      localStorage.removeItem('admin_access_token')
      localStorage.removeItem('admin_refresh_token')
      router.push('/login')
    }

    return Promise.reject(error)
  },
)

// Auth API
export const authApi = {
  login: (phone: string, password: string, totp_code?: string) =>
    api.post<TokenPair & { user: User }>('/auth/login', { phone, password, ...(totp_code ? { totp_code } : {}) }),

  refresh: (refreshToken: string) =>
    api.post<TokenPair>('/auth/refresh', { refresh_token: refreshToken }),

  profile: () => api.get<ProfileResponse>('/profile'),
}

// Admin API
export const adminApi = {
  // Stats
  getStats: () => api.get<AdminStats>('/admin/stats'),

  getChartData: (metric: string, period: string) =>
    api.get<ChartDataResponse>('/admin/stats/chart', {
      params: { metric, period },
    }),

  // SSE stream URL (for EventSource — use with token query param)
  getStatsStreamUrl: () => {
    const token = localStorage.getItem('admin_access_token')
    return `/api/admin/stats/stream${token ? `?token=${token}` : ''}`
  },

  // Users
  listUsers: (
    page = 1,
    limit = 20,
    filter = 'all',
    search = '',
    sortBy?: string,
    order?: string,
  ) =>
    api.get<{
      users: AdminUser[]
      total: number
      page: number
      limit: number
      stats: UserStats
    }>('/admin/users', {
      params: { page, limit, filter, search, sort_by: sortBy, order },
    }),

  getUserDetail: (id: number) =>
    api.get<AdminUserDetail>(`/admin/users/${id}`),

  updateUser: (
    id: number,
    data: { is_admin?: boolean; is_active?: boolean; plan_id?: number },
  ) => api.put<AdminUser>(`/admin/users/${id}`, data),

  deleteUser: (id: number) => api.delete(`/admin/users/${id}`),

  resetPassword: (id: number, newPassword: string) =>
    api.post(`/admin/users/${id}/reset-password`, {
      new_password: newPassword,
    }),

  mergeUsers: (primaryId: number, secondaryId: number) =>
    api.post('/admin/users/merge', {
      primary_user_id: primaryId,
      secondary_user_id: secondaryId,
    }),

  bulkUsers: (action: string, userIds: number[], planId?: number) =>
    api.post<BulkResult>('/admin/users/bulk', {
      action,
      user_ids: userIds,
      plan_id: planId,
    }),

  // Tunnels
  listTunnels: (params?: {
    type?: string
    user_id?: number
    node_id?: string
  }) =>
    api.get<{ tunnels: AdminTunnel[]; total: number }>('/admin/tunnels', {
      params,
    }),

  closeTunnel: (id: string) => api.delete(`/admin/tunnels/${id}`),

  bulkCloseTunnels: (tunnelIds: string[]) =>
    api.post<BulkResult>('/admin/tunnels/bulk-close', {
      tunnel_ids: tunnelIds,
    }),

  // Edge Nodes
  listNodes: (status?: string) =>
    api.get<{ nodes: EdgeNode[]; total: number }>('/admin/nodes', {
      params: { status },
    }),

  approveNode: (id: number) => api.post(`/admin/nodes/${id}/approve`),

  disableNode: (id: number) => api.post(`/admin/nodes/${id}/disable`),

  deleteNode: (id: number) => api.delete(`/admin/nodes/${id}`),

  // Plans
  listPlans: () =>
    api.get<{ plans: Plan[]; total: number }>('/admin/plans'),

  createPlan: (data: Omit<Plan, 'id'>) => api.post<Plan>('/admin/plans', data),

  updatePlan: (id: number, data: Partial<Omit<Plan, 'id' | 'slug'>>) =>
    api.put<Plan>(`/admin/plans/${id}`, data),

  deletePlan: (id: number) => api.delete(`/admin/plans/${id}`),

  // Subscriptions
  listSubscriptions: (page = 1, limit = 20, status?: string) =>
    api.get<{
      subscriptions: AdminSubscription[]
      total: number
      page: number
      limit: number
    }>('/admin/subscriptions', { params: { page, limit, ...(status ? { status } : {}) } }),

  cancelSubscription: (id: number) =>
    api.post<{ success: boolean; message: string }>(
      `/admin/subscriptions/${id}/cancel`,
    ),

  extendSubscription: (id: number, days: number) =>
    api.post<{ success: boolean; message: string }>(
      `/admin/subscriptions/${id}/extend`,
      { days },
    ),

  // Payments
  listPayments: (page = 1, limit = 20, status?: string) =>
    api.get<{
      payments: AdminPayment[]
      total: number
      page: number
      limit: number
    }>('/admin/payments', { params: { page, limit, ...(status ? { status } : {}) } }),

  // Custom Domains
  listCustomDomains: (page = 1, limit = 20) =>
    api.get<{
      domains: Array<CustomDomain>
      total: number
    }>('/admin/custom-domains', { params: { page, limit } }),

  deleteCustomDomain: (id: number) =>
    api.delete(`/admin/custom-domains/${id}`),

  // Audit Logs
  listAuditLogs: (page = 1, limit = 20, userId?: number) =>
    api.get<{ logs: AuditLog[]; total: number }>('/admin/audit-logs', {
      params: { page, limit, user_id: userId },
    }),

  // Settings
  getSettings: () => api.get<ServerSettings>('/admin/settings'),

  getSystemInfo: () => api.get<SystemInfo>('/admin/settings/system-info'),

  // Invite Codes
  listInviteCodes: () =>
    api.get<{ codes: InviteCode[]; total: number }>('/admin/invite-codes'),

  createInviteCode: (code?: string, maxUses?: number) =>
    api.post<InviteCode>('/admin/invite-codes', {
      code,
      max_uses: maxUses,
    }),

  deleteInviteCode: (id: number) =>
    api.delete(`/admin/invite-codes/${id}`),
}

export default api
