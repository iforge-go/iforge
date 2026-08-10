'use client'

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react'
import { api } from '@/lib/api'
import { User } from '@/lib/types'

interface UserContextType {
  user: User | null
  setUser: (user: User | null) => void
  refreshUser: () => void
  // 是否仍在进行首次认证检查。初始为 true（SSR 和客户端首次渲染一致，避免 hydration mismatch），
  // 在 useEffect 调用 refreshUser 后变为 false。
  // 页面应先检查 authLoading 再检查 !user，避免认证未完成时闪现"请先登录"。
  authLoading: boolean
}

const UserContext = createContext<UserContextType | null>(null)

// 把 token 同步到 cookie,让 server component(如 generateMetadata)能读取。
// localStorage 只在客户端可访问,server component 读不到,必须通过 cookie 传递。
// 与 locale cookie 同样的模式(I18nContext.syncLocaleCookie)。
// 安全性:token 本就在 localStorage(同样受 XSS 影响),写入 non-httponly cookie 不降低安全性。
// samesite=lax 防止跨站自动携带;max-age 30 天与 localStorage 实际生命周期一致。
function syncTokenCookie(token: string | null) {
  if (typeof document === 'undefined') return
  if (token) {
    document.cookie = `auth_token=${encodeURIComponent(token)}; path=/; max-age=2592000; samesite=lax`
  } else {
    document.cookie = 'auth_token=; path=/; max-age=0; samesite=lax'
  }
}

export function UserProvider({ children }: { children: React.ReactNode }) {
  // user 和 authLoading 的初始值在 SSR 和客户端首次渲染时保持一致，避免 hydration mismatch。
  // user 初始为 null（SSR 无 localStorage），authLoading 初始为 true（显示 loading 态而非"请先登录"）。
  // 真正的用户数据在 useEffect（仅客户端执行）中通过 refreshUser 填充。
  const [user, setUser] = useState<User | null>(null)
  const [authLoading, setAuthLoading] = useState<boolean>(true)

  const refreshUser = useCallback(() => {
    const cached = localStorage.getItem('cached_user')
    if (cached) {
      try { setUser(JSON.parse(cached)) } catch {}
      // 有缓存用户：可立即显示内容，API 调用仅用于后台刷新，无需阻塞
      setAuthLoading(false)
    }
    const token = localStorage.getItem('token')
    // 同步 token 到 cookie,让 server component(generateMetadata)能读取,
    // 用于 fetch 需认证的 API(如 PMS 项目名)。
    syncTokenCookie(token)
    if (token) {
      if (!cached) {
        // 无缓存但有 token：必须等待 API 返回才能确定用户身份
        setAuthLoading(true)
      }
      api.getCurrentUser().then((u) => {
        setUser(u)
        localStorage.setItem('cached_user', JSON.stringify(u))
      }).catch(() => {
        setUser(null)
      }).finally(() => {
        setAuthLoading(false)
      })
    } else {
      setUser(null)
      setAuthLoading(false)
    }
  }, [])

  useEffect(() => {
    refreshUser()
    const handleStorage = (e: StorageEvent) => {
      if (e.key === 'token' || e.key === 'cached_user') {
        refreshUser()
      }
    }
    window.addEventListener('storage', handleStorage)
    const handleAuthChange = () => refreshUser()
    window.addEventListener('auth-change', handleAuthChange)
    return () => {
      window.removeEventListener('storage', handleStorage)
      window.removeEventListener('auth-change', handleAuthChange)
    }
  }, [refreshUser])

  return (
    <UserContext.Provider value={{ user, setUser, refreshUser, authLoading }}>
      {children}
    </UserContext.Provider>
  )
}

export function useCurrentUser() {
  const context = useContext(UserContext)
  if (!context) {
    throw new Error('useCurrentUser must be used within UserProvider')
  }
  return context
}
