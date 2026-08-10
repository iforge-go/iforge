import type { Webhook, WebhookDelivery, TableInfo, ColumnInfo, QueryResult } from '../lib/types'

type RequestFn = <T>(endpoint: string, options?: { method?: string; body?: any; token?: string | null }) => Promise<T>

export function createMiscApi(request: RequestFn, baseUrl: string) {
  return {
    // Database Viewer
    async listDbTables() {
      return request<TableInfo[]>('/admin/dbviewer/tables')
    },

    async getDbTableSchema(tableName: string) {
      return request<ColumnInfo[]>(`/admin/dbviewer/tables/${tableName}/schema`)
    },

    async queryDbTable(tableName: string, limit: number, offset: number) {
      return request<QueryResult>(`/admin/dbviewer/tables/${tableName}/query`, {
        method: 'POST',
        body: { limit, offset },
      })
    },

    async executeDbQuery(query: string) {
      return request<QueryResult>('/admin/dbviewer/query', {
        method: 'POST',
        body: { query },
      })
    },

    // Webhooks
    async listWebhooks(owner: string, repo: string) {
      return request<Webhook[]>(`/repos/${owner}/${repo}/webhooks`)
    },

    async createWebhook(owner: string, repo: string, data: {
      url: string
      contentType: string
      token?: string
      events: string[]
      active: boolean
      insecureSSL: boolean
    }) {
      return request<Webhook>(`/repos/${owner}/${repo}/webhooks`, {
        method: 'POST',
        body: data,
      })
    },

    async deleteWebhook(owner: string, repo: string, id: number) {
      return request<void>(`/repos/${owner}/${repo}/webhooks/${id}`, {
        method: 'DELETE',
      })
    },

    async testWebhook(owner: string, repo: string, id: number) {
      return request<{ message: string }>(`/repos/${owner}/${repo}/webhooks/${id}/test`, {
        method: 'POST',
      })
    },

    async listWebhookDeliveries(owner: string, repo: string, webhookId: number) {
      return request<WebhookDelivery[]>(`/repos/${owner}/${repo}/webhooks/${webhookId}/deliveries`)
    },
  }
}
