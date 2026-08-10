import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'
import { API_BASE } from '@/lib/api'

// 判断当前请求的语言偏好,供 server component 的 generateMetadata 读取。
// 优先级:locale cookie > Accept-Language header > 默认 'zh'。
// I18nContext 在客户端设置 locale cookie(用户切换语言或首次访问时)。
function getLocale(request: NextRequest): 'zh' | 'en' {
  const cookieLocale = request.cookies.get('locale')?.value
  if (cookieLocale === 'zh' || cookieLocale === 'en') return cookieLocale

  // 没有 cookie 时(首次访问),用 Accept-Language header 判断
  const acceptLang = request.headers.get('accept-language') || ''
  return acceptLang.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

export async function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl

  if (
    pathname.startsWith('/_next') ||
    pathname.startsWith('/api') ||
    pathname.match(/\.(ico|png|jpg|jpeg|svg|css|js|woff|woff2)$/)
  ) {
    return NextResponse.next()
  }

  try {
    const response = await fetch(`${API_BASE}/setup/status`, {
      cache: 'no-store',
    })
    const data = await response.json()

    if (!data.initialized && pathname !== '/setup') {
      return NextResponse.redirect(new URL('/setup', request.url))
    }

    if (data.initialized && pathname === '/setup') {
      return NextResponse.redirect(new URL('/', request.url))
    }
  } catch (err) {
    console.error('[proxy] setup/status fetch failed:', err)
  }

  // 把当前 pathname 和 locale 注入到 request header,供 server component 的 generateMetadata 读取。
  // [owner]/[repo]/layout.tsx 据此设置动态 title(支持 i18n),无需为每个子目录创建 layout.tsx。
  const requestHeaders = new Headers(request.headers)
  requestHeaders.set('x-pathname', pathname)
  requestHeaders.set('x-locale', getLocale(request))

  return NextResponse.next({
    request: {
      headers: requestHeaders,
    },
  })
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
}
