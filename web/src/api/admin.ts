import type {
  SSHKey, GeneralSettings, PublicGeneralSettings, SMTPSettings, LDAPSettings,
  Release, ReleaseAsset, AccountPreference, ExtraMailAddress, WikiPage,
  AccessToken, DeployKey, GPGKey, Contributor,
  WebhookSettings, UploadSettings, RepositorySettings, OIDCSettings, AISettings,
  AIModelConfig, AIModelConfigInput,
  SystemInfo, Organization, OrganizationMember, Repository, User, Issue, LFSObject, Participant,
  AuditLog, AuditLogListResponse
} from '../lib/types'

type RequestFn = <T>(endpoint: string, options?: { method?: string; body?: any; token?: string | null }) => Promise<T>

export function createAdminApi(request: RequestFn, baseUrl: string) {
  return {
    // SSH Keys
    async listSSHKeys() {
      return request<SSHKey[]>('/user/sshkeys')
    },

    async createSSHKey(title: string, publicKey: string) {
      return request<SSHKey>('/user/sshkeys', {
        method: 'POST',
        body: { title, publicKey },
      })
    },

    async deleteSSHKey(sshKeyId: number) {
      return request<void>(`/user/sshkeys/${sshKeyId}`, {
        method: 'DELETE',
      })
    },



    // Admin settings
    async getGeneralSettings() {
      return request<GeneralSettings>('/admin/settings/general')
    },

    async updateGeneralSettings(settings: Partial<GeneralSettings>) {
      return request<void>('/admin/settings/general', {
        method: 'PUT',
        body: settings,
      })
    },

    async getSMTPSettings() {
      return request<SMTPSettings>('/admin/settings/smtp')
    },

    async updateSMTPSettings(settings: Partial<SMTPSettings>) {
      return request<void>('/admin/settings/smtp', {
        method: 'PUT',
        body: settings,
      })
    },

    async testSMTPSettings() {
      return request<{ message: string }>('/admin/settings/smtp/test', {
        method: 'POST',
      })
    },

    async getLDAPSettings() {
      return request<LDAPSettings>('/admin/settings/ldap')
    },

    async updateLDAPSettings(settings: Partial<LDAPSettings>) {
      return request<void>('/admin/settings/ldap', {
        method: 'PUT',
        body: settings,
      })
    },

    async testLDAPSettings() {
      return request<{ message: string }>('/admin/settings/ldap/test', {
        method: 'POST',
      })
    },

    // Releases
    async listReleases(owner: string, repo: string) {
      return request<Release[]>(`/repos/${owner}/${repo}/releases`)
    },

    async getRelease(owner: string, repo: string, tag: string) {
      return request<Release>(`/repos/${owner}/${repo}/releases/${encodeURIComponent(tag)}`)
    },

    async createRelease(owner: string, repo: string, data: { tag: string; name: string; content?: string }) {
      return request<Release>(`/repos/${owner}/${repo}/releases`, {
        method: 'POST',
        body: data,
      })
    },

    async updateRelease(owner: string, repo: string, tag: string, data: { name: string; content?: string }) {
      return request<Release>(`/repos/${owner}/${repo}/releases/${encodeURIComponent(tag)}`, {
        method: 'PATCH',
        body: data,
      })
    },

    async deleteRelease(owner: string, repo: string, tag: string) {
      return request<void>(`/repos/${owner}/${repo}/releases/${encodeURIComponent(tag)}`, {
        method: 'DELETE',
      })
    },

    // Release Assets
    async listReleaseAssets(owner: string, repo: string, tag: string) {
      return request<ReleaseAsset[]>(`/repos/${owner}/${repo}/releases/${encodeURIComponent(tag)}/assets`)
    },

    async uploadReleaseAsset(owner: string, repo: string, tag: string, file: File, label?: string) {
      const formData = new FormData()
      formData.append('file', file)
      if (label) formData.append('label', label)

      const authToken = typeof window !== 'undefined' ? localStorage.getItem('token') : null
      const headers: Record<string, string> = {}
      if (authToken) headers['Authorization'] = authToken

      const response = await fetch(`${baseUrl}/repos/${owner}/${repo}/releases/${encodeURIComponent(tag)}/assets`, {
        method: 'POST',
        headers,
        body: formData,
        credentials: 'include',
      })

      if (!response.ok) {
        const errorBody = await response.json().catch(() => ({ error: response.statusText }))
        const err = new Error(errorBody.error || errorBody.message || `API error: ${response.status}`)
        ;(err as any).status = response.status
        throw err
      }

      return response.json() as Promise<ReleaseAsset>
    },

    async deleteReleaseAsset(owner: string, repo: string, tag: string, assetId: number) {
      return request<{ message: string }>(`/repos/${owner}/${repo}/releases/${encodeURIComponent(tag)}/assets/${assetId}`, {
        method: 'DELETE',
      })
    },

    getAssetDownloadUrl(owner: string, repo: string, tag: string, assetId: number) {
      return `${baseUrl}/repos/${owner}/${repo}/releases/${encodeURIComponent(tag)}/assets/${assetId}`
    },

    // Account Preferences
    async getUserPreferences() {
      return request<AccountPreference>('/user/preferences')
    },

    async updateUserPreferences(data: { highlighterTheme?: string; notification?: boolean; timezone?: string }) {
      return request<{ message: string }>('/user/preferences', {
        method: 'PUT',
        body: data,
      })
    },

    // Extra Mail Addresses
    async listExtraEmails() {
      return request<{ addresses: ExtraMailAddress[] }>('/user/emails')
    },

    async addExtraEmail(mailAddress: string) {
      return request<{ message: string }>('/user/emails', {
        method: 'POST',
        body: { mailAddress },
      })
    },

    async deleteExtraEmail(mailAddress: string) {
      return request<{ message: string }>(`/user/emails/${encodeURIComponent(mailAddress)}`, {
        method: 'DELETE',
      })
    },

    async getPrimaryEmail() {
      return request<{ mailAddress: string }>('/user/emails/primary')
    },

    // Wiki
    async listWikiPages(owner: string, repo: string) {
      return request<WikiPage[]>(`/repos/${owner}/${repo}/wiki`)
    },

    async getWikiPage(owner: string, repo: string, pageName: string) {
      return request<WikiPage>(`/repos/${owner}/${repo}/wiki/${pageName}`)
    },

    async createWikiPage(owner: string, repo: string, data: { pageName: string; title: string; content: string }) {
      return request<WikiPage>(`/repos/${owner}/${repo}/wiki`, {
        method: 'POST',
        body: data,
      })
    },

    async updateWikiPage(owner: string, repo: string, pageName: string, data: { title: string; content: string }) {
      return request<WikiPage>(`/repos/${owner}/${repo}/wiki/${pageName}`, {
        method: 'PUT',
        body: data,
      })
    },

    async deleteWikiPage(owner: string, repo: string, pageName: string) {
      return request<void>(`/repos/${owner}/${repo}/wiki/${pageName}`, {
        method: 'DELETE',
      })
    },

    // Access Tokens
    async listAccessTokens() {
      return request<AccessToken[]>('/user/tokens')
    },

    async createAccessToken(note: string) {
      return request<{ token: AccessToken; value: string }>('/user/tokens', {
        method: 'POST',
        body: { note },
      })
    },

    async deleteAccessToken(id: number) {
      return request<void>(`/user/tokens/${id}`, {
        method: 'DELETE',
      })
    },

    // Deploy Keys
    async listDeployKeys(owner: string, repo: string) {
      return request<DeployKey[]>(`/repos/${owner}/${repo}/deploy_keys`)
    },

    async createDeployKey(owner: string, repo: string, data: { title: string; publicKey: string; allowWrite: boolean }) {
      return request<DeployKey>(`/repos/${owner}/${repo}/deploy_keys`, {
        method: 'POST',
        body: data,
      })
    },

    async deleteDeployKey(owner: string, repo: string, id: number) {
      return request<void>(`/repos/${owner}/${repo}/deploy_keys/${id}`, {
        method: 'DELETE',
      })
    },

    // LFS objects
    async listLFSObjects(owner: string, repo: string, page = 1, limit = 50) {
      return request<{ objects: LFSObject[]; total: number }>(
        `/repos/${owner}/${repo}/lfs/objects?page=${page}&limit=${limit}`
      )
    },

    async deleteLFSObject(owner: string, repo: string, oid: string) {
      return request<{ message: string }>(
        `/repos/${owner}/${repo}/lfs/objects/${oid}`,
        { method: 'DELETE' }
      )
    },

    // Contributors
    async getContributors(owner: string, repo: string) {
      return request<Contributor[]>(`/repos/${owner}/${repo}/contributors`)
    },

    // Search
    async searchAllIssues(query: string, limit = 25, offset = 0) {
      return request<{ issues: Issue[]; total: number; limit: number; offset: number; participants: Participant[] }>(
        `/search/issues?q=${encodeURIComponent(query)}&limit=${limit}&offset=${offset}`
      )
    },

    // GPG Keys
    async listGPGKeys() {
      return request<GPGKey[]>('/user/gpgkeys')
    },

    async createGPGKey(title: string, publicKey: string) {
      return request<GPGKey>('/user/gpgkeys', {
        method: 'POST',
        body: { title, publicKey },
      })
    },

    async deleteGPGKey(id: number) {
      return request<void>(`/user/gpgkeys/${id}`, {
        method: 'DELETE',
      })
    },

    async searchRepoIssues(owner: string, repo: string, query: string, limit = 25, offset = 0) {
      return request<{ issues: Issue[]; total: number; limit: number; offset: number }>(
        `/repos/${owner}/${repo}/issues/search?q=${encodeURIComponent(query)}&limit=${limit}&offset=${offset}`
      )
    },

    // Admin SSH settings - REMOVED: SSH settings page removed

    async getWebhookSettings() {
      return request<WebhookSettings>('/admin/settings/webhook')
    },

    async updateWebhookSettings(settings: Partial<WebhookSettings>) {
      return request<void>('/admin/settings/webhook', {
        method: 'PUT',
        body: settings,
      })
    },

    async getUploadSettings() {
      return request<UploadSettings>('/admin/settings/upload')
    },

    async updateUploadSettings(settings: Partial<UploadSettings>) {
      return request<void>('/admin/settings/upload', {
        method: 'PUT',
        body: settings,
      })
    },

    async getRepositorySettings() {
      return request<RepositorySettings>('/admin/settings/repository')
    },

    async updateRepositorySettings(settings: Partial<RepositorySettings>) {
      return request<void>('/admin/settings/repository', {
        method: 'PUT',
        body: settings,
      })
    },

    async getOIDCSettings() {
      return request<OIDCSettings>('/admin/settings/oidc')
    },

    async updateOIDCSettings(settings: Partial<OIDCSettings>) {
      return request<void>('/admin/settings/oidc', {
        method: 'PUT',
        body: settings,
      })
    },

    async testOIDCSettings() {
      return request<{ message: string }>('/admin/settings/oidc/test', {
        method: 'POST',
      })
    },

    async getAISettings() {
      return request<AISettings>('/admin/settings/ai')
    },

    async updateAISettings(settings: Partial<AISettings>) {
      return request<void>('/admin/settings/ai', {
        method: 'PUT',
        body: settings,
      })
    },

    async testAISettings() {
      return request<{ message: string }>('/admin/settings/ai/test', {
        method: 'POST',
      })
    },

    // AI 模型配置（多模型管理）
    async listAIModelConfigs() {
      return request<AIModelConfig[]>('/admin/settings/ai/models')
    },

    async getAIModelConfig(id: number) {
      return request<AIModelConfig>(`/admin/settings/ai/models/${id}`)
    },

    async createAIModelConfig(input: AIModelConfigInput) {
      return request<AIModelConfig>('/admin/settings/ai/models', {
        method: 'POST',
        body: input,
      })
    },

    async updateAIModelConfig(id: number, input: AIModelConfigInput) {
      return request<AIModelConfig>(`/admin/settings/ai/models/${id}`, {
        method: 'PUT',
        body: input,
      })
    },

    async deleteAIModelConfig(id: number) {
      return request<{ message: string }>(`/admin/settings/ai/models/${id}`, {
        method: 'DELETE',
      })
    },

    async setDefaultAIModelConfig(id: number) {
      return request<AIModelConfig>(`/admin/settings/ai/models/${id}/default`, {
        method: 'POST',
      })
    },

    async testAIModelConfig(id: number, input?: Partial<AIModelConfigInput>) {
      return request<{ message: string }>(`/admin/settings/ai/models/${id}/test`, {
        method: 'POST',
        body: input || {},
      })
    },

    // Admin
    async listUsers() {
      return request<User[]>('/admin/users')
    },

    async createUser(user: { userName: string; password: string; fullName: string; mailAddress: string; isAdmin: boolean }) {
      return request<User>('/admin/users', {
        method: 'POST',
        body: user,
      })
    },

    async adminUpdateUser(username: string, data: { fullName?: string; mailAddress?: string; isAdmin?: boolean; url?: string; description?: string; password?: string }) {
      return request<User>(`/admin/users/${username}`, {
        method: 'PATCH',
        body: data,
      })
    },

    async adminDeleteUser(username: string) {
      return request<void>(`/admin/users/${username}`, {
        method: 'DELETE',
      })
    },

    async getSystemInfo() {
      return request<SystemInfo>('/admin/system')
    },

    // Admin Organizations
    async adminListOrganizations() {
      return request<Organization[]>('/admin/organizations')
    },

    async adminCreateOrganization(data: { organizationName: string; description?: string }) {
      return request<Organization>('/admin/organizations', {
        method: 'POST',
        body: data,
      })
    },

    async adminUpdateOrganization(organizationName: string, data: Partial<Organization>) {
      return request<Organization>(`/admin/organizations/${organizationName}`, {
        method: 'PATCH',
        body: data,
      })
    },

    async adminDeleteOrganization(organizationName: string) {
      return request<void>(`/admin/organizations/${organizationName}`, {
        method: 'DELETE',
      })
    },

    // Admin Repositories
    async adminListRepos(page = 1, limit = 20) {
      return request<{ repositories: Repository[]; total: number; page: number; limit: number }>(
        `/admin/repos?page=${page}&limit=${limit}`
      )
    },

    async adminGetRepo(owner: string, repo: string) {
      return request<Repository>(`/admin/repos/${owner}/${repo}`)
    },

    async adminDeleteRepo(owner: string, repo: string) {
      return request<void>(`/admin/repos/${owner}/${repo}`, {
        method: 'DELETE',
      })
    },

    // Organizations
    async listOrganizations() {
      return request<Organization[]>('/organizations')
    },

    async listMyOrganizations() {
      return request<Organization[]>('/user/organizations')
    },

    async listManagedOrganizations() {
      return request<string[]>('/user/managed-organizations')
    },

    async createOrganization(name: string, description: string, isPrivate: boolean) {
      return request<Organization>('/organizations', {
        method: 'POST',
        body: { organizationName: name, description, isPrivate },
      })
    },

    async getOrganization(organizationName: string) {
      return request<Organization>(`/organizations/${organizationName}`)
    },

    async updateOrganization(organizationName: string, data: Partial<Organization>) {
      return request<Organization>(`/organizations/${organizationName}`, {
        method: 'PUT',
        body: data,
      })
    },

    async deleteOrganization(organizationName: string) {
      return request<void>(`/organizations/${organizationName}`, {
        method: 'DELETE',
      })
    },

    async getOrganizationMembers(organizationName: string) {
      return request<OrganizationMember[]>(`/organizations/${organizationName}/members`)
    },

    async addOrganizationMember(organizationName: string, userName: string, role: string) {
      return request<void>(`/organizations/${organizationName}/members`, {
        method: 'POST',
        body: { userName, role },
      })
    },

    async removeOrganizationMember(organizationName: string, userName: string) {
      return request<void>(`/organizations/${organizationName}/members/${userName}`, {
        method: 'DELETE',
      })
    },

    async getOrganizationRepos(organizationName: string) {
      return request<Repository[]>(`/repos?owner=${organizationName}`)
    },

    // Audit Logs (admin only)
    async listAuditLogs(params: {
      actor?: string
      action?: string
      resourceType?: string
      startTime?: string
      endTime?: string
      page?: number
      pageSize?: number
    } = {}) {
      const query = new URLSearchParams()
      if (params.actor) query.set('actor', params.actor)
      if (params.action) query.set('action', params.action)
      if (params.resourceType) query.set('resourceType', params.resourceType)
      if (params.startTime) query.set('startTime', params.startTime)
      if (params.endTime) query.set('endTime', params.endTime)
      if (params.page) query.set('page', String(params.page))
      if (params.pageSize) query.set('pageSize', String(params.pageSize))
      const qs = query.toString()
      return request<AuditLogListResponse>(`/admin/audit-logs${qs ? '?' + qs : ''}`)
    },

    // Plugin Management
    async listPlugins() {
      return request<any[]>('/admin/plugins')
    },

    async listPluginTemplates() {
      return request<any[]>('/admin/plugin-templates')
    },

    async installPlugin(data: { template_id: string; name: string; config?: Record<string, any> }) {
      return request<any>('/admin/plugins/install', {
        method: 'POST',
        body: data,
      })
    },

    async enablePlugin(id: number) {
      return request<any>(`/admin/plugins/${id}/enable`, {
        method: 'POST',
      })
    },

    async disablePlugin(id: number) {
      return request<any>(`/admin/plugins/${id}/disable`, {
        method: 'POST',
      })
    },

    async updatePlugin(id: number, data: { config: Record<string, any> }) {
      return request<any>(`/admin/plugins/${id}`, {
        method: 'PATCH',
        body: data,
      })
    },

    async uninstallPlugin(id: number) {
      return request<any>(`/admin/plugins/${id}`, {
        method: 'DELETE',
      })
    },
  }
}
