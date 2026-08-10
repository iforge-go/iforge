'use client'

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react'
import { api } from '@/lib/api'

interface SiteSettings {
  siteName: string
  description: string
  timezone: string
  allowRegistration: boolean
}

interface SiteSettingsContextType {
  settings: SiteSettings | null
  loading: boolean
  refresh: () => void
}

const SiteSettingsContext = createContext<SiteSettingsContextType>({
  settings: null,
  loading: true,
  refresh: () => {},
})

export function SiteSettingsProvider({ children }: { children: React.ReactNode }) {
  const [settings, setSettings] = useState<SiteSettings | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchSettings = useCallback(() => {
    api.getPublicGeneralSettings()
      .then((data) => {
        setSettings({
          siteName: data?.siteName || 'iForge',
          description: data?.description || '',
          timezone: data?.timezone || '',
          allowRegistration: data?.allowRegistration === true,
        })
      })
      .catch(() => {
        // API 调用失败时，默认不允许注册（安全起见）
        setSettings({
          siteName: 'iForge',
          description: '',
          timezone: '',
          allowRegistration: false,
        })
      })
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    fetchSettings()
  }, [fetchSettings])

  return (
    <SiteSettingsContext.Provider value={{ settings, loading, refresh: fetchSettings }}>
      {children}
    </SiteSettingsContext.Provider>
  )
}

export function useSiteSettings() {
  return useContext(SiteSettingsContext)
}
