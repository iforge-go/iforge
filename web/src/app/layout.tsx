import type { Metadata } from 'next'
import { Providers } from './providers'
import { ClientLayout } from '@/components/ClientLayout'
import { ColorModeScriptInjector } from '@/components/ColorModeScriptInjector'
import { API_BASE } from '@/lib/api'

const DEFAULT_TITLE = 'iForge'
const DEFAULT_DESC = 'A modern Git management platform'

// 从后端获取站点设置，用于 SSR 和客户端导航时由 Next.js metadata 系统管理 title。
async function getSiteSettings(): Promise<{ siteName: string; description: string }> {
  try {
    const res = await fetch(`${API_BASE}/settings/general`, {
      next: { revalidate: 10 },
    })
    if (!res.ok) throw new Error('failed to fetch site settings')
    const data = await res.json()
    return {
      siteName: data?.siteName || DEFAULT_TITLE,
      description: data?.description || DEFAULT_DESC,
    }
  } catch {
    return { siteName: DEFAULT_TITLE, description: DEFAULT_DESC }
  }
}

export async function generateMetadata(): Promise<Metadata> {
  const { siteName, description } = await getSiteSettings()
  return {
    title: siteName,
    description,
  }
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="zh-CN" suppressHydrationWarning style={{ overflowY: 'scroll' }}>
      <body suppressHydrationWarning style={{ backgroundColor: '#f7f8fa' }}>
        <ColorModeScriptInjector />
        <Providers>
          <ClientLayout>
            {children}
          </ClientLayout>
        </Providers>
      </body>
    </html>
  )
}
