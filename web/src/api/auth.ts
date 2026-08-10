import type { User } from '../lib/types'

type RequestFn = <T>(endpoint: string, options?: { method?: string; body?: any; token?: string | null }) => Promise<T>

export function createAuthApi(request: RequestFn, _baseUrl: string) {
  return {
    async login(userName: string, password: string) {
      return request<{ token: string; user: User }>('/login', {
        method: 'POST',
        body: { userName, password },
      })
    },

    async register(userName: string, password: string, fullName: string, mailAddress: string) {
      return request<User>('/register', {
        method: 'POST',
        body: { userName, password, fullName, mailAddress },
      })
    },

    // 检查用户名是否可注册（公开端点，注册页实时校验用）
    async checkUsername(username: string) {
      return request<{ available: boolean; reason: string }>(
        `/register/check-username?username=${encodeURIComponent(username)}`,
      )
    },

    async getCurrentUser() {
      return request<User>('/user')
    },

    async updateUser(data: { fullName?: string; mailAddress?: string; url?: string; description?: string }) {
      return request<User>('/user', {
        method: 'PATCH',
        body: data,
      })
    },

    async changePassword(currentPassword: string, newPassword: string) {
      return request<{ message: string }>('/user/change-password', {
        method: 'POST',
        body: { currentPassword, newPassword },
      })
    },

    async getUserByUsername(username: string) {
      return request<User>(`/users/${username}`)
    },

    // Password reset flow
    async requestPasswordReset(email: string) {
      return request<{ message: string }>('/password/reset/request', {
        method: 'POST',
        body: { email },
      })
    },

    async validateResetToken(token: string) {
      return request<{ valid: boolean }>(`/password/reset/validate?token=${encodeURIComponent(token)}`, {
        method: 'GET',
      })
    },

    async resetPassword(token: string, newPassword: string) {
      return request<{ message: string }>('/password/reset', {
        method: 'POST',
        body: { token, newPassword },
      })
    },

    // OIDC authentication
    async getOIDCStatus() {
      return request<{ enabled: boolean }>('/settings/oidc/status', {
        method: 'GET',
      })
    },

    // Setup wizard
    async getSetupStatus() {
      return request<{ initialized: boolean }>('/setup/status')
    },

    async setup(data: {
      adminUsername: string
      adminPassword: string
      adminEmail: string
    }) {
      return request<{ message: string }>('/setup', {
        method: 'POST',
        body: data,
      })
    },

    async getOIDCAuthURL() {
      return request<{ authUrl: string; state: string }>('/auth/oidc/url', {
        method: 'GET',
      })
    },

    async oidcCallback(code: string, state: string) {
      return request<{ token: string; user: User }>(`/auth/oidc/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`, {
        method: 'POST',
      })
    },
  }
}
