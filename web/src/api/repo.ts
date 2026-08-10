import type { Repository, Branch, ProtectedBranchInfo, CommitInfo, FileEntry, FileContent, Tag, CodeSearchResponse } from '../lib/types'

type RequestFn = <T>(endpoint: string, options?: { method?: string; body?: any; token?: string | null }) => Promise<T>

export function createRepoApi(request: RequestFn, baseUrl: string) {
  return {
    async listRepos() {
      return request<Repository[]>('/repos')
    },

    async listUserRepos(username: string) {
      return request<Repository[]>(`/users/${username}/repos`)
    },

    async getMyRepos() {
      return request<Repository[]>(`/user/repos`)
    },

    async getRepo(owner: string, repo: string) {
      return request<Repository>(`/repos/${owner}/${repo}`)
    },

    async createRepo(name: string, description: string, isPrivate: boolean, owner?: string) {
      return request<Repository>('/repos', {
        method: 'POST',
        body: { name, description, private: isPrivate, owner },
      })
    },

    async createFromTemplate(data: {
      templateOwner: string
      templateRepo: string
      newOwner: string
      newRepo: string
      isPrivate: boolean
    }) {
      return request<Repository>('/repos/template/create', {
        method: 'POST',
        body: data,
      })
    },

    async updateRepo(owner: string, repo: string, data: Partial<Repository>) {
      return request<Repository>(`/repos/${owner}/${repo}`, {
        method: 'PATCH',
        body: data,
      })
    },

    async deleteRepo(owner: string, repo: string) {
      return request<void>(`/repos/${owner}/${repo}`, {
        method: 'DELETE',
      })
    },

    // Branches
    async listBranches(owner: string, repo: string) {
      return request<Branch[]>(`/repos/${owner}/${repo}/branches`)
    },

    async createBranch(owner: string, repo: string, branchName: string, fromBranch: string) {
      return request<Branch>(`/repos/${owner}/${repo}/branches`, {
        method: 'POST',
        body: { branchName, from: fromBranch },
      })
    },

    async deleteBranch(owner: string, repo: string, branchName: string) {
      return request<void>(`/repos/${owner}/${repo}/branches/${encodeURIComponent(branchName)}`, {
        method: 'DELETE',
      })
    },

    async renameBranch(owner: string, repo: string, oldName: string, newName: string) {
      return request<void>(`/repos/${owner}/${repo}/branches/${encodeURIComponent(oldName)}/rename`, {
        method: 'POST',
        body: { newName },
      })
    },

    // Branch protection
    async listProtectedBranches(owner: string, repo: string) {
      return request<ProtectedBranchInfo[]>(`/repos/${owner}/${repo}/branches/protection`)
    },

    async protectBranch(owner: string, repo: string, branchName: string) {
      return request<{ message: string }>(`/repos/${owner}/${repo}/branches/protection`, {
        method: 'POST',
        body: { branchName },
      })
    },

    async unprotectBranch(owner: string, repo: string, branchName: string) {
      return request<{ message: string }>(`/repos/${owner}/${repo}/branches/protection/${encodeURIComponent(branchName)}`, {
        method: 'DELETE',
      })
    },

    // Commits
    async listCommits(owner: string, repo: string, sha?: string, page = 1, limit = 20) {
      const params = new URLSearchParams({ page: String(page), limit: String(limit) })
      if (sha) params.set('sha', sha)
      return request<CommitInfo[]>(`/repos/${owner}/${repo}/commits?${params}`)
    },

    async getCommit(owner: string, repo: string, commitHash: string) {
      return request<CommitInfo>(`/repos/${owner}/${repo}/commits/${commitHash}`)
    },

    // Files
    async listFiles(owner: string, repo: string, ref?: string, path?: string) {
      const params = new URLSearchParams()
      if (ref) params.set('ref', ref)
      if (path) params.set('path', path)
      const query = params.toString()
      return request<FileEntry[]>(`/repos/${owner}/${repo}/files${query ? '?' + query : ''}`)
    },

    async listAllFiles(owner: string, repo: string, ref?: string) {
      const params = new URLSearchParams()
      if (ref) params.set('ref', ref)
      return request<FileEntry[]>(`/repos/${owner}/${repo}/files/all?${params.toString()}`)
    },

    async getFileContent(owner: string, repo: string, path: string, ref?: string) {
      const params = new URLSearchParams()
      if (ref) params.set('ref', ref)
      const query = params.toString()
      return request<FileContent>(`/repos/${owner}/${repo}/file?path=${encodeURIComponent(path)}${query ? '&' + query : ''}`)
    },

    async createFile(owner: string, repo: string, branch: string, path: string, content: string, message: string) {
      return request<{ message: string }>(`/repos/${owner}/${repo}/files`, {
        method: 'POST',
        body: { branch, path, content, message },
      })
    },

    async updateFile(owner: string, repo: string, branch: string, path: string, content: string, message: string) {
      return request<{ message: string }>(`/repos/${owner}/${repo}/files`, {
        method: 'PUT',
        body: { branch, path, content, message },
      })
    },

    async deleteFile(owner: string, repo: string, branch: string, path: string, message: string) {
      return request<{ message: string }>(`/repos/${owner}/${repo}/files`, {
        method: 'DELETE',
        body: { branch, path, message },
      })
    },

    async uploadFile(owner: string, repo: string, branch: string, path: string, message: string, file: File) {
      const formData = new FormData()
      formData.append('branch', branch)
      formData.append('path', path)
      formData.append('message', message)
      formData.append('file', file)

      const authToken = typeof window !== 'undefined' ? localStorage.getItem('token') : null
      const headers: Record<string, string> = {}
      if (authToken) headers['Authorization'] = authToken

      const response = await fetch(`${baseUrl}/repos/${owner}/${repo}/upload`, {
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

      return response.json() as Promise<{ message: string; filename: string; path: string }>
    },

    // Tags
    async listTags(owner: string, repo: string) {
      return request<Tag[]>(`/repos/${owner}/${repo}/tags`)
    },

    async createTag(owner: string, repo: string, data: { tagName: string; target?: string; message?: string }) {
      return request<{ message: string }>(`/repos/${owner}/${repo}/tags`, {
        method: 'POST',
        body: data,
      })
    },

    async deleteTag(owner: string, repo: string, tagName: string) {
      return request<{ message: string }>(`/repos/${owner}/${repo}/tags/${encodeURIComponent(tagName)}`, {
        method: 'DELETE',
      })
    },

    // Forks
    async listForks(owner: string, repo: string) {
      return request<Repository[]>(`/repos/${owner}/${repo}/forks`)
    },

    async getForkStatus(owner: string, repo: string) {
      return request<{
        isFork: boolean
        parentAvailable?: boolean
        parentOwner?: string
        parentRepo?: string
        ahead?: number
        behind?: number
      }>(`/repos/${owner}/${repo}/fork/status`)
    },

    async syncFork(owner: string, repo: string) {
      return request<{ message: string }>(`/repos/${owner}/${repo}/fork/sync`, {
        method: 'POST',
      })
    },

    // Code search - search file contents within a repository
    async searchCode(owner: string, repo: string, query: string, ref?: string) {
      const params = new URLSearchParams({ q: query })
      if (ref) params.set('ref', ref)
      return request<CodeSearchResponse>(`/repos/${owner}/${repo}/search?${params.toString()}`)
    },
  }
}
