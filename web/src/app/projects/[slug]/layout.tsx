import type { Metadata } from 'next'
import { cookies, headers } from 'next/headers'
import zh from '@/locales/zh.json'
import en from '@/locales/en.json'
import ProjectLayoutClient from './ProjectLayoutClient'
import { API_BASE } from '@/lib/api'

// 项目子页面的 title i18n key 映射
// key 是 URL 中的路径段,value 是 locales 文件中的 i18n key
// 新增子页面时,只需在此处添加一行,并在 locales 文件中添加对应翻译
const PAGE_TITLE_KEYS: Record<string, string> = {
  backlog: 'pms.backlog',
  board: 'pms.board',
  members: 'pms.members',
  sprint: 'pms.sprints',
  story: 'pms.userStories',
  task: 'pms.tasks',
  tasks: 'pms.tasks',
}

const translations: Record<'zh' | 'en', typeof zh> = { zh, en }

// 按 dotted key 路径(如 "pms.issues")从翻译对象中取值
function getNestedValue(obj: Record<string, any>, key: string): string | undefined {
  const keys = key.split('.')
  let current: any = obj
  for (const k of keys) {
    if (current == null || typeof current !== 'object') return undefined
    current = current[k]
  }
  return typeof current === 'string' ? current : undefined
}

// 从 auth_token cookie 读取 token(UserContext.syncTokenCookie 写入),
// 构造 Authorization header 给后端 API。
// server component 无法访问 localStorage,必须通过 cookie 传递。
async function getAuthHeaders(): Promise<Record<string, string>> {
  const cookieStore = await cookies()
  const encodedToken = cookieStore.get('auth_token')?.value
  if (!encodedToken) return {}
  // syncTokenCookie 写入时用 encodeURIComponent 编码,这里解码还原原 token
  return { Authorization: decodeURIComponent(encodedToken) }
}

// 获取项目名称(fallback 到 slug)
async function fetchProjectName(slug: string, reqHeaders: Record<string, string>): Promise<string> {
  try {
    const res = await fetch(`${API_BASE}/projects/${slug}`, {
      headers: reqHeaders,
      next: { revalidate: 30 },
    })
    if (res.ok) {
      const data = await res.json()
      return data?.name || slug
    }
  } catch {
    // fallback to slug
  }
  return slug
}

// 获取 sprint 标题(fetch 失败返回 null,调用方 fallback 到 i18n 翻译)
async function fetchSprintTitle(sprintSlug: string, reqHeaders: Record<string, string>): Promise<string | null> {
  try {
    const res = await fetch(`${API_BASE}/sprints/${sprintSlug}`, {
      headers: reqHeaders,
      next: { revalidate: 30 },
    })
    if (res.ok) {
      const data = await res.json()
      return data?.title || null
    }
  } catch {
    // fallback to null
  }
  return null
}

// 获取 user story 标题
async function fetchUserStoryTitle(storySlug: string, reqHeaders: Record<string, string>): Promise<string | null> {
  try {
    const res = await fetch(`${API_BASE}/user-stories/${storySlug}`, {
      headers: reqHeaders,
      next: { revalidate: 30 },
    })
    if (res.ok) {
      const data = await res.json()
      return data?.title || null
    }
  } catch {
    // fallback to null
  }
  return null
}

// 获取 task 标题
// 后端 getTask 返回 { task, assignees, subtaskCount, subtaskCompletedCount }
async function fetchTaskTitle(projectSlug: string, taskId: string, reqHeaders: Record<string, string>): Promise<string | null> {
  try {
    const res = await fetch(`${API_BASE}/projects/${projectSlug}/tasks/${taskId}`, {
      headers: reqHeaders,
      next: { revalidate: 30 },
    })
    if (res.ok) {
      const data = await res.json()
      return data?.task?.title || data?.title || null
    }
  } catch {
    // fallback to null
  }
  return null
}

// 项目 layout(server component)
//
// 集中管理所有项目页面的 title(支持 i18n):
//   - 项目概览: "项目名"
//   - 列表页:   "页面名 · 项目名"(页面名根据 locale 翻译)
//   - 详情页:   "实体标题 · 项目名"(sprint/task/story 详情页 fetch 具体实体标题,
//              fetch 失败 fallback 到 i18n 翻译的页面名)
//
// 通过 proxy.ts 注入的 x-pathname 和 x-locale header 判断当前路径和语言,
// 用 PAGE_TITLE_KEYS 映射表查找 i18n key,从 locales 文件获取翻译。
//
// 实际 UI 逻辑由 ProjectLayoutClient(client component)渲染,
// 这样既支持 server-side metadata,又保留 client-side 交互(数据获取、权限判断等)。
//
// 注意:ProjectContext 和 useProject 定义在 ./ProjectContext.tsx(client component),
// 子页面通过 `import { useProject } from '@/app/projects/[slug]/ProjectContext'` 引用。
// 不能在这里 re-export,否则会让本 server layout 被 Next.js 标记为 client component,
// 导致 generateMetadata 报错。
export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>
}): Promise<Metadata> {
  const { slug } = await params
  const headersList = await headers()
  const pathname = headersList.get('x-pathname') || ''
  const locale = (headersList.get('x-locale') as 'zh' | 'en') || 'zh'
  const t = translations[locale]

  const reqHeaders = await getAuthHeaders()

  // pathname 形如:
  //   /projects/slug               (概览)
  //   /projects/slug/backlog       (列表页)
  //   /projects/slug/sprint/xxx    (详情页)
  // segments[0]=projects, segments[1]=slug, segments[2]=子页面段, segments[3]=详情实体 slug/id
  const segments = pathname.split('/').filter(Boolean)
  const pageSegment = segments[2]
  const detailSegment = segments[3]

  // 详情页:fetch 实体标题,与项目名并行请求提高性能
  if (detailSegment && pageSegment) {
    let detailTitlePromise: Promise<string | null> = Promise.resolve(null)
    let fallbackPageTitleKey: string | null = null

    if (pageSegment === 'sprint') {
      detailTitlePromise = fetchSprintTitle(detailSegment, reqHeaders)
      fallbackPageTitleKey = 'pms.sprints'
    } else if (pageSegment === 'task') {
      detailTitlePromise = fetchTaskTitle(slug, detailSegment, reqHeaders)
      fallbackPageTitleKey = 'pms.tasks'
    } else if (pageSegment === 'story') {
      detailTitlePromise = fetchUserStoryTitle(detailSegment, reqHeaders)
      fallbackPageTitleKey = 'pms.userStories'
    }

    if (fallbackPageTitleKey) {
      const [detailTitle, projectName] = await Promise.all([
        detailTitlePromise,
        fetchProjectName(slug, reqHeaders),
      ])
      // 优先用实体标题,fetch 失败则 fallback 到 i18n 翻译的页面名
      const titlePart = detailTitle || getNestedValue(t, fallbackPageTitleKey) || pageSegment
      return { title: `${titlePart} · ${projectName}` }
    }
  }

  // 列表页:用 i18n 翻译的页面名
  if (pageSegment && PAGE_TITLE_KEYS[pageSegment]) {
    const projectName = await fetchProjectName(slug, reqHeaders)
    const pageTitle = getNestedValue(t, PAGE_TITLE_KEYS[pageSegment]) || pageSegment
    return {
      title: `${pageTitle} · ${projectName}`,
    }
  }

  // 项目概览页:用项目名作为 title
  return {
    title: await fetchProjectName(slug, reqHeaders),
  }
}

export default function ProjectLayout({ children }: { children: React.ReactNode }) {
  return <ProjectLayoutClient>{children}</ProjectLayoutClient>
}
