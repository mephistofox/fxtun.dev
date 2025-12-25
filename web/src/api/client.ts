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
  display_name: string
  is_admin: boolean
  totp_enabled: boolean
  created_at: string
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
  refresh: (refreshToken: string) => api.post<TokenPair>('/auth/refresh', { refresh_token: refreshToken }),
}

export const profileApi = {
  get: () => api.get<User>('/profile'),
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
  list: () => api.get<{ domains: Domain[] }>('/domains'),
  reserve: (subdomain: string) => api.post<Domain>('/domains', { subdomain }),
  release: (id: number) => api.delete(`/domains/${id}`),
  check: (subdomain: string) => api.get<{ available: boolean }>(`/domains/check/${subdomain}`),
}

export const tokensApi = {
  list: () => api.get<{ tokens: APIToken[] }>('/tokens'),
  create: (data: CreateTokenRequest) => api.post<CreateTokenResponse>('/tokens', data),
  delete: (id: number) => api.delete(`/tokens/${id}`),
}

export const downloadsApi = {
  list: () => api.get<DownloadsResponse>('/downloads'),
}

export default api
