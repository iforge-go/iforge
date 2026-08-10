import type { PublicGeneralSettings } from '../lib/types'

type RequestFn = <T>(endpoint: string, options?: { method?: string; body?: any; token?: string | null }) => Promise<T>

export function createSettingsApi(request: RequestFn, baseUrl: string) {
  return {
    // Public general settings (no auth required) — siteName / description / timezone / allowRegistration
    async getPublicGeneralSettings() {
      return request<PublicGeneralSettings>('/settings/general')
    },

    // Public SSH settings (no auth required)
    async getSSHSettings() {
      return request<{ enabled: boolean; host: string; port: number }>('/settings/ssh')
    },

    // Public clone URL templates (no auth required)
    async getCloneUrlTemplates() {
      return request<{ httpsUrlTemplate: string; sshUrlTemplate: string }>('/settings/clone-url-templates')
    },
  }
}
