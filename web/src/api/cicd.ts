import type { Pipeline, CICDJob, JobLog, Artifact, Secret, Environment, Deployment, Runner, CronSchedule } from '@/lib/types'
import { API_BASE } from '@/lib/api'

type RequestFn = <T>(endpoint: string, options?: { method?: string; body?: any }) => Promise<T>

export function createCICDApi(request: RequestFn, _baseUrl: string) {
  return {
    // Pipelines
    listPipelines: (owner: string, repo: string, page = 1, pageSize = 20) =>
      request<{ items: Pipeline[]; total: number; page: number; pageSize: number }>(
        `/repos/${owner}/${repo}/pipelines?page=${page}&page_size=${pageSize}`
      ),
    getPipeline: (owner: string, repo: string, id: number) =>
      request<Pipeline>(`/repos/${owner}/${repo}/pipelines/${id}`),
    cancelPipeline: (owner: string, repo: string, id: number) =>
      request<{ ok: boolean }>(`/repos/${owner}/${repo}/pipelines/${id}/cancel`, { method: 'POST' }),
    retryPipeline: (owner: string, repo: string, id: number) =>
      request<{ ok: boolean }>(`/repos/${owner}/${repo}/pipelines/${id}/retry`, { method: 'POST' }),

    // Jobs
    listJobs: (owner: string, repo: string, pipelineId: number) =>
      request<{ items: CICDJob[] }>(`/repos/${owner}/${repo}/pipelines/${pipelineId}/jobs`),
    getJob: (owner: string, repo: string, jobId: number) =>
      request<CICDJob>(`/repos/${owner}/${repo}/jobs/${jobId}`),
    getJobLogs: (owner: string, repo: string, jobId: number, fromLine = 0, limit = 5000) =>
      request<{ items: JobLog[]; fromLine: number; limit: number }>(
        `/repos/${owner}/${repo}/jobs/${jobId}/logs?from_line=${fromLine}&limit=${limit}`
      ),
    triggerManualJob: (owner: string, repo: string, jobId: number, variables?: Record<string, string>) =>
      request<{ ok: boolean }>(`/repos/${owner}/${repo}/jobs/${jobId}/trigger`, {
        method: 'POST',
        body: { variables: variables || {} },
      }),

    // Artifacts
    listArtifacts: (owner: string, repo: string, pipelineId: number) =>
      request<{ items: Artifact[] }>(`/repos/${owner}/${repo}/pipelines/${pipelineId}/artifacts`),
    // 下载 artifact:跨域请求需带 cookie,用 fetch + blob 触发浏览器下载
    downloadArtifact: async (owner: string, repo: string, artifactId: number, filename?: string) => {
      const resp = await fetch(`${API_BASE}/repos/${owner}/${repo}/artifacts/${artifactId}/download`, {
        credentials: 'include',
      })
      if (!resp.ok) throw new Error(`Download failed: ${resp.status}`)
      const blob = await resp.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename || `artifact-${artifactId}`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    },

    // Secrets
    listSecrets: (owner: string, repo: string) =>
      request<{ items: Secret[] }>(`/repos/${owner}/${repo}/secrets`),
    setSecret: (owner: string, repo: string, key: string, value: string) =>
      request<{ ok: boolean }>(`/repos/${owner}/${repo}/secrets`, { method: 'POST', body: { key, value } }),
    deleteSecret: (owner: string, repo: string, key: string) =>
      request<{ ok: boolean }>(`/repos/${owner}/${repo}/secrets/${encodeURIComponent(key)}`, { method: 'DELETE' }),

    // Environments & Deployments
    listEnvironments: (owner: string, repo: string) =>
      request<{ items: Environment[] }>(`/repos/${owner}/${repo}/environments`),
    listDeployments: (owner: string, repo: string, environment?: string, page = 1, pageSize = 20) => {
      const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) })
      if (environment) params.set('environment', environment)
      return request<{ items: Deployment[]; total: number; page: number; pageSize: number }>(
        `/repos/${owner}/${repo}/deployments?${params.toString()}`
      )
    },
    listDeploymentsByCommit: (owner: string, repo: string, commitSha: string) =>
      request<{ items: Deployment[] }>(`/repos/${owner}/${repo}/commits/${commitSha}/deployments`),

    // Runners (admin)
    listRunners: () => request<{ items: Runner[] }>(`/admin/runners`),
    deleteRunner: (id: number) => request<{ ok: boolean }>(`/admin/runners/${id}`, { method: 'DELETE' }),

    // Cron Schedules (定时触发)
    listCrons: (owner: string, repo: string) =>
      request<{ items: CronSchedule[] }>(`/repos/${owner}/${repo}/crons`),
    createCron: (owner: string, repo: string, data: {
      name: string
      schedule: string
      branch: string
      yamlConfig?: string
    }) =>
      request<CronSchedule>(`/repos/${owner}/${repo}/crons`, { method: 'POST', body: data }),
    getCron: (owner: string, repo: string, id: number) =>
      request<CronSchedule>(`/repos/${owner}/${repo}/crons/${id}`),
    updateCron: (owner: string, repo: string, id: number, data: {
      schedule: string
      branch: string
      yamlConfig?: string
      enabled: boolean
    }) =>
      request<CronSchedule>(`/repos/${owner}/${repo}/crons/${id}`, { method: 'PATCH', body: data }),
    deleteCron: (owner: string, repo: string, id: number) =>
      request<void>(`/repos/${owner}/${repo}/crons/${id}`, { method: 'DELETE' }),
  }
}
