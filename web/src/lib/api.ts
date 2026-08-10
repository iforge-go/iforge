import { createAuthApi } from '../api/auth'
import { createRepoApi } from '../api/repo'
import { createIssueApi } from '../api/issue'
import { createUserApi } from '../api/user'
import { createAdminApi } from '../api/admin'
import { createSettingsApi } from '../api/settings'
import { createMiscApi } from '../api/misc'
import { createScrumApi } from '../api/scrum'
import { createCICDApi } from '../api/cicd'
import { ApiError } from './errorMessages'

// 适配反向代理和直连两种模式：
// - 开发模式（端口 3001）：直接访问后端 8081 端口
// - 生产模式（Nginx 反向代理）：使用相对路径
// - SSR 端：使用 SERVER_API_URL（Docker 内部网络）或环境变量
export function getServerBase(): string {
  // 浏览器端
  if (typeof window !== 'undefined') {
    // 如果配置了环境变量，使用环境变量
    if (process.env.NEXT_PUBLIC_API_BASE) {
      return process.env.NEXT_PUBLIC_API_BASE.replace(/\/api\/v1$/, '')
    }
    
    // 检查当前端口，判断是开发模式还是生产模式
    const port = window.location.port
    if (port === '3001') {
      // 开发模式：直接访问后端 8081 端口
      return `http://${window.location.hostname}:8081`
    }
    if (port === '3000') {
      // Docker 直连模式：前端 3000 → 后端 8080
      return `http://${window.location.hostname}:8080`
    }
    
    // 生产模式（Nginx 反向代理，同源）：使用相对路径
    return ''
  }
  // SSR 端：Docker 内 web 容器用 SERVER_API_URL 访问 api 容器（如 http://api:8081）
  if (process.env.SERVER_API_URL) {
    return process.env.SERVER_API_URL.replace(/\/api\/v1$/, '')
  }
  if (process.env.NEXT_PUBLIC_API_BASE) {
    return process.env.NEXT_PUBLIC_API_BASE.replace(/\/api\/v1$/, '')
  }
  return 'http://localhost:8081'
}
export const SERVER_BASE = getServerBase()
export const API_BASE = `${SERVER_BASE}/api/v1`

// Re-export all types for backward compatibility
export * from './types'

interface RequestOptions {
  method?: string
  body?: unknown
  token?: string | null
}

class ApiClient {
  baseUrl: string

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl
  }

  getToken(): string | null {
    if (typeof window === 'undefined') return null
    return localStorage.getItem('token')
  }

  async request<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
    const { method = 'GET', body, token } = options
    const authToken = token ?? this.getToken()

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }

    if (authToken) {
      headers['Authorization'] = authToken
    }

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
      credentials: 'include',
    })

    if (!response.ok) {
      const errorBody = await response.json().catch(() => ({ error: response.statusText }))
      const err = new Error(errorBody.error || errorBody.message || `API error: ${response.status}`) as ApiError
      err.status = response.status
      // 透传结构化错误字段供前端 catch 识别:
      // - messageKey: 面向用户的本地化错误 i18n key (getLocalizedErrorMessage 用)
      // - reason: 程序逻辑判断标识符 (如 task_blocked, ai_not_configured)
      // - blockedTasks: task_blocked 时附带的前置依赖任务列表
      if (errorBody.messageKey) { err.messageKey = errorBody.messageKey }
      if (errorBody.reason) { err.reason = errorBody.reason }
      if (errorBody.blockedTasks) { err.blockedTasks = errorBody.blockedTasks }
      if (response.status === 401) {
        localStorage.removeItem('token')
        localStorage.removeItem('cached_user')
        // 同步清除 auth_token cookie,避免 server component 用过期 token 继续请求后端
        if (typeof document !== 'undefined') {
          document.cookie = 'auth_token=; path=/; max-age=0; samesite=lax'
        }
        if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
          window.location.href = '/login'
        }
      }
      throw err
    }

    if (response.status === 204) {
      return {} as T
    }

    return response.json()
  }
}

const core = new ApiClient(API_BASE)

export const api = {
  baseUrl: core.baseUrl,
  getToken: core.getToken.bind(core),
  request: core.request.bind(core),
  ...createAuthApi(core.request.bind(core), core.baseUrl),
  ...createRepoApi(core.request.bind(core), core.baseUrl),
  ...createIssueApi(core.request.bind(core), core.baseUrl),
  ...createUserApi(core.request.bind(core), core.baseUrl),
  ...createAdminApi(core.request.bind(core), core.baseUrl),
  ...createSettingsApi(core.request.bind(core), core.baseUrl),
  ...createMiscApi(core.request.bind(core), core.baseUrl),
  ...createScrumApi(core.request.bind(core), core.baseUrl),
  ...createCICDApi(core.request.bind(core), core.baseUrl),
}
