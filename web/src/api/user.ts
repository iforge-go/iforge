import type { Repository, Collaborator, Notification, Activity, ContributionDay, RepositoryMirror, Priority, CommitStatus, CombinedCommitStatus, CustomField, CodeQualityReport, User, UserSearchResult, ParsedRepoURL, Participant } from '../lib/types'

type RequestFn = <T>(endpoint: string, options?: { method?: string; body?: any; token?: string | null }) => Promise<T>

export function createUserApi(request: RequestFn, baseUrl: string) {
  const serverBase = baseUrl.replace(/\/api\/v1$/, '')

  return {
    // Collaborators
    async listCollaborators(owner: string, repo: string) {
      return request<Collaborator[]>(`/repos/${owner}/${repo}/collaborators`)
    },

    async addCollaborator(owner: string, repo: string, collaboratorName: string, role: string) {
      return request<void>(`/repos/${owner}/${repo}/collaborators`, {
        method: 'POST',
        body: { collaboratorName, role },
      })
    },

    async updateCollaboratorRole(owner: string, repo: string, collaboratorName: string, role: string) {
      return request<void>(`/repos/${owner}/${repo}/collaborators/${collaboratorName}`, {
        method: 'PATCH',
        body: { role },
      })
    },

    async removeCollaborator(owner: string, repo: string, userName: string) {
      return request<void>(`/repos/${owner}/${repo}/collaborators/${userName}`, {
        method: 'DELETE',
      })
    },

    // Repository danger zone operations
    async transferRepo(owner: string, repo: string, newOwner: string) {
      return request<void>(`/repos/${owner}/${repo}/transfer`, {
        method: 'POST',
        body: { newOwner },
      })
    },

    async renameRepo(owner: string, repo: string, newName: string) {
      return request<void>(`/repos/${owner}/${repo}/rename`, {
        method: 'POST',
        body: { newName },
      })
    },

    async archiveRepo(owner: string, repo: string, isArchived: boolean) {
      return request<void>(`/repos/${owner}/${repo}/archive`, {
        method: 'POST',
        body: { isArchived },
      })
    },

    async setTemplateRepo(owner: string, repo: string, isTemplate: boolean) {
      return request<void>(`/repos/${owner}/${repo}/template`, {
        method: 'POST',
        body: { isTemplate },
      })
    },

    // User search
    async searchUsers(keyword: string, limit: number = 20) {
      return request<{ users: UserSearchResult[] }>(`/users/search?q=${encodeURIComponent(keyword)}&limit=${limit}`)
    },

    // Stars
    async starRepo(owner: string, repo: string) {
      return request<void>(`/repos/${owner}/${repo}/star`, {
        method: 'POST',
      })
    },

    async unstarRepo(owner: string, repo: string) {
      return request<void>(`/repos/${owner}/${repo}/star`, {
        method: 'DELETE',
      })
    },

    async isStarred(owner: string, repo: string) {
      return request<{ starred: boolean }>(`/repos/${owner}/${repo}/star`)
    },

    async getStargazers(owner: string, repo: string) {
      return request<{ stargazers: User[] }>(`/repos/${owner}/${repo}/stargazers`)
    },

    async listStarredRepos(username: string) {
      return request<{ repos: Repository[] }>(`/users/${username}/starred`)
    },

    // Watch
    async watchRepo(owner: string, repo: string, notification: boolean = true) {
      return request(`/repos/${owner}/${repo}/watch`, {
        method: 'POST',
        body: { notification },
      })
    },

    async unwatchRepo(owner: string, repo: string) {
      return request<void>(`/repos/${owner}/${repo}/watch`, {
        method: 'DELETE',
      })
    },

    async isWatching(owner: string, repo: string) {
      return request<{ watching: boolean }>(`/repos/${owner}/${repo}/watch`)
    },

    async getWatchers(owner: string, repo: string) {
      return request<User[]>(`/repos/${owner}/${repo}/watchers`)
    },

    async getUserRole(owner: string, repo: string) {
      return request<{ role: 'owner' | 'member' | 'viewer' | '' | null; canCreateIssue?: boolean }>(`/repos/${owner}/${repo}/role`)
    },

    // Fork
    async forkRepo(owner: string, repo: string) {
      return request<Repository>(`/repos/${owner}/${repo}/fork`, {
        method: 'POST',
      })
    },

    async getForks(owner: string, repo: string) {
      return request<Repository[]>(`/repos/${owner}/${repo}/forks`)
    },

    async getForkCount(owner: string, repo: string) {
      return request<{ count: number }>(`/repos/${owner}/${repo}/fork/count`)
    },

    async isForked(owner: string, repo: string) {
      return request<{ forked: boolean }>(`/repos/${owner}/${repo}/fork/isForked`)
    },

    // Notifications
    // 后端 sideload participants：批量预取 actor 头像信息，前端无需逐条 /users/:name
    async listNotifications(unreadOnly = false) {
      const query = unreadOnly ? '?unread=true' : ''
      return request<{ notifications: Notification[]; participants: Participant[] }>(`/notifications${query}`)
    },

    async getUnreadNotificationCount() {
      return request<{ count: number }>('/notifications/unread/count')
    },

    // Activities
    async listActivities(limit = 50, offset = 0) {
      return request<{ activities: Activity[]; participants: Participant[] }>(`/activities?limit=${limit}&offset=${offset}`)
    },

    async listRepoActivities(owner: string, repo: string, limit = 50, offset = 0) {
      return request<{ activities: Activity[]; participants: Participant[] }>(`/repos/${owner}/${repo}/activities?limit=${limit}&offset=${offset}`)
    },

    async listUserActivities(username: string, limit = 50, offset = 0) {
      return request<{ activities: Activity[]; participants: Participant[] }>(`/users/${username}/activities?limit=${limit}&offset=${offset}`)
    },

    async getUserContributions(username: string) {
      return request<ContributionDay[]>(`/users/${username}/contributions`)
    },

    async markNotificationAsRead(slug: string) {
      return request<void>(`/notifications/${slug}/read`, {
        method: 'POST',
      })
    },

    async markAllNotificationsAsRead() {
      return request<void>('/notifications/read-all', {
        method: 'POST',
      })
    },

    async deleteNotification(slug: string) {
      return request<void>(`/notifications/${slug}`, {
        method: 'DELETE',
      })
    },

    // Import
    async importRepo(data: {
      owner: string
      repoName: string
      sourceUrl: string
      isPrivate: boolean
      description?: string
    }) {
      return request<Repository>(`/repos/import`, {
        method: 'POST',
        body: data,
      })
    },

    async parseRepoUrl(url: string) {
      return request<ParsedRepoURL>(`/repos/parse-url?url=${encodeURIComponent(url)}`)
    },

    // Mirror
    async getMirror(owner: string, repo: string) {
      return request<RepositoryMirror | null>(`/repos/${owner}/${repo}/mirror`)
    },

    async createMirror(owner: string, repo: string, mirror: Partial<RepositoryMirror>) {
      return request<RepositoryMirror>(`/repos/${owner}/${repo}/mirror`, {
        method: 'POST',
        body: mirror,
      })
    },

    async updateMirror(owner: string, repo: string, mirror: Partial<RepositoryMirror>) {
      return request<RepositoryMirror>(`/repos/${owner}/${repo}/mirror`, {
        method: 'PUT',
        body: mirror,
      })
    },

    async deleteMirror(owner: string, repo: string) {
      return request<void>(`/repos/${owner}/${repo}/mirror`, { method: 'DELETE' })
    },

    async syncMirror(owner: string, repo: string) {
      return request<{ message: string }>(`/repos/${owner}/${repo}/mirror/sync`, {
        method: 'POST',
      })
    },

    // Priorities
    async listPriorities(owner: string, repo: string) {
      return request<Priority[]>(`/repos/${owner}/${repo}/priorities`)
    },

    async createPriority(owner: string, repo: string, data: { priorityName: string; description?: string; color: string }) {
      return request<Priority>(`/repos/${owner}/${repo}/priorities`, {
        method: 'POST',
        body: data,
      })
    },

    async updatePriority(owner: string, repo: string, id: number, data: { priorityName?: string; description?: string; color?: string }) {
      return request<void>(`/repos/${owner}/${repo}/priorities/${id}`, {
        method: 'PATCH',
        body: data,
      })
    },

    async deletePriority(owner: string, repo: string, id: number) {
      return request<void>(`/repos/${owner}/${repo}/priorities/${id}`, { method: 'DELETE' })
    },

    async reorderPriorities(owner: string, repo: string, priorityIds: number[]) {
      return request<void>(`/repos/${owner}/${repo}/priorities/reorder`, {
        method: 'POST',
        body: { priorityIds },
      })
    },

    async getDefaultPriority(owner: string, repo: string) {
      return request<{ priorityId: number }>(`/repos/${owner}/${repo}/priorities/default`)
    },

    async setDefaultPriority(owner: string, repo: string, priorityId: number | null) {
      return request<void>(`/repos/${owner}/${repo}/priorities/default`, {
        method: 'POST',
        body: { priorityId },
      })
    },

    // Commit Statuses
    async listCommitStatuses(owner: string, repo: string, commit: string) {
      return request<CommitStatus[]>(`/repos/${owner}/${repo}/statuses/${commit}`)
    },

    async getCombinedCommitStatus(owner: string, repo: string, commit: string) {
      return request<CombinedCommitStatus>(`/repos/${owner}/${repo}/commits/${commit}/status`)
    },

    // Custom Fields
    async listCustomFields(owner: string, repo: string) {
      return request<CustomField[]>(`/repos/${owner}/${repo}/custom_fields`)
    },

    async createCustomField(owner: string, repo: string, data: {
      fieldName: string
      fieldType: string
      constraints?: string | null
      enableForIssues: boolean
      enableForMergeRequests: boolean
    }) {
      return request<CustomField>(`/repos/${owner}/${repo}/custom_fields`, {
        method: 'POST',
        body: data,
      })
    },

    async updateCustomField(owner: string, repo: string, id: number, data: Partial<{
      fieldName: string
      fieldType: string
      constraints: string
      enableForIssues: boolean
      enableForMergeRequests: boolean
    }>) {
      return request<void>(`/repos/${owner}/${repo}/custom_fields/${id}`, {
        method: 'PATCH',
        body: data,
      })
    },

    async deleteCustomField(owner: string, repo: string, id: number) {
      return request<void>(`/repos/${owner}/${repo}/custom_fields/${id}`, { method: 'DELETE' })
    },

    // Code Quality
    async analyzeCodeQuality(owner: string, repo: string) {
      return request<CodeQualityReport>(`/repos/${owner}/${repo}/code-quality`)
    },

    // Archive & Atom Feed URLs (not under /api/v1)
    getArchiveUrl(owner: string, repo: string, ref: string, format: 'zip' | 'tar.gz') {
      return `${serverBase}/${owner}/${repo}/archive/${ref}.${format}`
    },

    getRepoAtomUrl(owner: string, repo: string) {
      return `${serverBase}/feeds/repos/${owner}/${repo}.atom`
    },

    getUserAtomUrl(username: string) {
      return `${serverBase}/feeds/users/${username}.atom`
    },
  }
}
