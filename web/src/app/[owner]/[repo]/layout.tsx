import type { Metadata } from 'next'
import { cookies, headers } from 'next/headers'
import RepoLayoutClient from './RepoLayoutClient'
import zh from '@/locales/zh.json'
import en from '@/locales/en.json'
import { API_BASE } from '@/lib/api'
import { extractFirstH1 } from '@/lib/markdown'

// 仓库子页面的 title i18n key 映射
// key 是 URL 中的路径段,value 是 locales 文件中的 i18n key
// 新增子页面时,只需在此处添加一行,并在 locales 文件中添加对应翻译
const PAGE_TITLE_KEYS: Record<string, string> = {
  activity: 'repo.activity',
  blob: 'repo.code',
  branches: 'repo.branches',
  'code-quality': 'repo.codeQuality',
  commits: 'repo.commits',
  contributors: 'repo.contributors',
  issues: 'repo.issues',
  'merge-requests': 'repo.mergeRequests',
  milestones: 'repo.milestones',
  pipelines: 'repo.pipelines',
  releases: 'repo.releases',
  search: 'search.title',
  settings: 'repo.settings',
  tags: 'repo.tags',
  tree: 'repo.code',
  wiki: 'repo.wiki',
}

const translations: Record<'zh' | 'en', typeof zh> = { zh, en }

// 按 dotted key 路径(如 "repo.issues")从翻译对象中取值
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
// 用于 fetch 需认证的 API(如私有仓库的 issue),公开仓库即使无 token 也能正常工作。
async function getAuthHeaders(): Promise<Record<string, string>> {
  const cookieStore = await cookies()
  const encodedToken = cookieStore.get('auth_token')?.value
  if (!encodedToken) return {}
  // syncTokenCookie 写入时用 encodeURIComponent 编码,这里解码还原原 token
  return { Authorization: decodeURIComponent(encodedToken) }
}

// 获取仓库首页的 title: 优先从 README 的 h1 提取，其次 "owner/repo: description"，最后 "owner/repo"
// /repos/{owner}/{repo} 是公开 API,无需认证,可用 revalidate 缓存
async function fetchRepoHomeTitle(owner: string, repo: string): Promise<string> {
  let title = `${owner}/${repo}`
  
  // 尝试从 README 提取 h1 标题
  try {
    const readmeRes = await fetch(`${API_BASE}/repos/${owner}/${repo}/contents/README.md`, {
      next: { revalidate: 30 },
    })
    if (readmeRes.ok) {
      const readmeData = await readmeRes.json()
      if (readmeData?.content) {
        const content = Buffer.from(readmeData.content, 'base64').toString('utf-8')
        const h1 = extractFirstH1(content)
        if (h1) return h1
      }
    }
  } catch {
    // fallback to next method
  }
  
  // 尝试从仓库 description 获取
  try {
    const res = await fetch(`${API_BASE}/repos/${owner}/${repo}`, {
      next: { revalidate: 30 },
    })
    if (res.ok) {
      const data = await res.json()
      if (data?.description) {
        title = `${owner}/${repo}: ${data.description}`
      }
    }
  } catch {
    // fallback to "owner/repo"
  }
  return title
}

// 通用详情页 fetch:GET {API_BASE}/repos/{owner}/{repo}/{path}/{detailId},
// 从响应中提取 title 字段。fetch 失败返回 null。
// 使用 cache: 'no-store' 避免不同用户的 fetch 结果被共享缓存(私有仓库信息不应跨用户泄露)
async function fetchDetailTitle(
  owner: string,
  repo: string,
  path: string,
  detailId: string,
  reqHeaders: Record<string, string>,
  extractTitle: (data: any) => string | null,
  encodeDetailId = false,
): Promise<string | null> {
  try {
    const encodedId = encodeDetailId ? encodeURIComponent(detailId) : detailId
    const res = await fetch(`${API_BASE}/repos/${owner}/${repo}/${path}/${encodedId}`, {
      headers: reqHeaders,
      cache: 'no-store',
    })
    if (res.ok) {
      const data = await res.json()
      return extractTitle(data)
    }
  } catch {
    // fallback to null
  }
  return null
}

// 仓库 layout(server component)
//
// 集中管理所有仓库页面的 title(支持 i18n):
//   - 仓库首页: "owner/repo: description"
//   - 列表页:   "页面名 · owner/repo"(页面名根据 locale 翻译)
//   - 详情页:   "实体标题 · owner/repo"(fetch 具体实体标题,
//              fetch 失败 fallback 到 i18n 翻译的页面名)
//
// 通过 proxy.ts 注入的 x-pathname 和 x-locale header 判断当前路径和语言,
// 用 PAGE_TITLE_KEYS 映射表查找 i18n key,从 locales 文件获取翻译。
//
// 实际 UI 逻辑由 RepoLayoutClient(client component)渲染,
// 这样既支持 server-side metadata,又保留 client-side 交互(star/fork/watch 等)。
//
// 注意:RepoContext 和 useRepo 定义在 ./RepoContext.tsx(client component),
// 子页面通过 `import { useRepo } from '@/app/[owner]/[repo]/RepoContext'` 引用。
// 不能在这里 re-export,否则会让本 server layout 被 Next.js 标记为 client component,
// 导致 generateMetadata 报错。
export async function generateMetadata({
  params,
}: {
  params: Promise<{ owner: string; repo: string }>
}): Promise<Metadata> {
  const { owner, repo } = await params
  const headersList = await headers()
  const pathname = headersList.get('x-pathname') || ''
  const locale = (headersList.get('x-locale') as 'zh' | 'en') || 'zh'
  const t = translations[locale]

  // pathname 形如:
  //   /owner/repo                  (首页)
  //   /owner/repo/issues           (列表页)
  //   /owner/repo/issues/13        (详情页)
  // segments[0]=owner, segments[1]=repo, segments[2]=子页面段, segments[3]=详情实体 id
  const segments = pathname.split('/').filter(Boolean)
  const pageSegment = segments[2]
  const detailSegment = segments[3]

  // 详情页:fetch 实体标题
  if (detailSegment && pageSegment) {
    const reqHeaders = await getAuthHeaders()
    let detailTitle: string | null = null
    let fallbackKey: string | null = null

    switch (pageSegment) {
      case 'issues':
        detailTitle = await fetchDetailTitle(owner, repo, 'issues', detailSegment, reqHeaders, (d) => d?.title || null)
        fallbackKey = 'repo.issues'
        break
      case 'merge-requests':
        detailTitle = await fetchDetailTitle(owner, repo, 'merge-requests', detailSegment, reqHeaders, (d) => d?.title || null)
        fallbackKey = 'repo.mergeRequests'
        break
      case 'milestones':
        detailTitle = await fetchDetailTitle(owner, repo, 'milestones', detailSegment, reqHeaders, (d) => d?.title || null)
        fallbackKey = 'repo.milestones'
        break
      case 'wiki':
        // wiki pageName 可能含特殊字符(如 中文/空格),需编码
        detailTitle = await fetchDetailTitle(
          owner, repo, 'wiki', detailSegment, reqHeaders,
          (d) => d?.title || null,
          true, // encodeDetailId
        )
        fallbackKey = 'repo.wiki'
        break
      case 'commit':
        // commit 详情页路由是 commit/[hash](单数),不是 commits
        // commit message 可能多行,取第一行作为 title
        detailTitle = await fetchDetailTitle(owner, repo, 'commits', detailSegment, reqHeaders, (d) => {
          const msg = d?.message
          if (!msg) return null
          return msg.split('\n')[0].trim() || null
        })
        fallbackKey = 'repo.commits'
        break
      case 'pipelines':
        // pipeline message 通常是触发 commit 的 message
        detailTitle = await fetchDetailTitle(owner, repo, 'pipelines', detailSegment, reqHeaders, (d) => d?.message || null)
        fallbackKey = 'repo.pipelines'
        break
      // 注:releases 没有详情页路由(只有列表页),无需处理
    }

    if (fallbackKey) {
      // 优先用实体标题,fetch 失败则 fallback 到 i18n 翻译的页面名
      const titlePart = detailTitle || getNestedValue(t, fallbackKey) || pageSegment
      return { title: `${titlePart} · ${owner}/${repo}` }
    }
  }

  // 列表页:用 i18n 翻译的页面名
  if (pageSegment && PAGE_TITLE_KEYS[pageSegment]) {
    const pageTitle = getNestedValue(t, PAGE_TITLE_KEYS[pageSegment]) || pageSegment
    return {
      title: `${pageTitle} · ${owner}/${repo}`,
    }
  }

  // 仓库首页或未匹配的页面:用 "owner/repo: description" 格式
  return {
    title: await fetchRepoHomeTitle(owner, repo),
  }
}

export default function RepoLayout({ children }: { children: React.ReactNode }) {
  return <RepoLayoutClient>{children}</RepoLayoutClient>
}
