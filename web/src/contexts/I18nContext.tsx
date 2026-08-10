'use client'

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react'
import zh from '@/locales/zh.json'
import en from '@/locales/en.json'

export type Locale = 'zh' | 'en'

type Translations = typeof zh

const translations: Record<Locale, Translations> = { zh, en }

interface I18nContextType {
  t: (key: string, params?: Record<string, string | number>) => string
  locale: Locale
  setLocale: (locale: Locale) => void
}

const I18nContext = createContext<I18nContextType | null>(null)

function getNestedValue(obj: Record<string, any>, key: string): string | undefined {
  const keys = key.split('.')
  let current: any = obj
  for (const k of keys) {
    if (current == null || typeof current !== 'object') return undefined
    current = current[k]
  }
  return typeof current === 'string' ? current : undefined
}

// 把 locale 同步到 cookie,让 server component(如 generateMetadata)能读取当前语言。
// localStorage 只在客户端可访问,server component 读不到,必须通过 cookie 传递。
function syncLocaleCookie(locale: Locale) {
  document.cookie = `locale=${locale}; path=/; max-age=31536000; samesite=lax`
}

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>('zh')

  useEffect(() => {
    const saved = localStorage.getItem('locale') as Locale
    let initialLocale: Locale
    if (saved === 'zh' || saved === 'en') {
      initialLocale = saved
    } else {
      const browser = navigator.language.split('-')[0]
      initialLocale = browser === 'zh' ? 'zh' : 'en'
    }
    setLocaleState(initialLocale)
    syncLocaleCookie(initialLocale)
    // 同步更新 HTML lang 属性
    document.documentElement.lang = initialLocale
  }, [])

  const setLocale = useCallback((newLocale: Locale) => {
    setLocaleState(newLocale)
    localStorage.setItem('locale', newLocale)
    syncLocaleCookie(newLocale)
    // 更新 HTML lang 属性，让客户端立即感知语言变化
    document.documentElement.lang = newLocale
    // 不再调用 router.refresh()，避免全页刷新
    // Server component 会在下次导航时自动读取新 cookie
  }, [])

  const t = useCallback(
    (key: string, params?: Record<string, string | number>) => {
      let text = getNestedValue(translations[locale], key)
      if (text === undefined) {
        text = getNestedValue(translations['zh'], key)
      }
      if (text === undefined) return key
      if (params) {
        Object.entries(params).forEach(([k, v]) => {
          text = text!.replace(new RegExp(`\\{\\{${k}\\}\\}`, 'g'), String(v))
          text = text!.replace(new RegExp(`\\{${k}\\}`, 'g'), String(v))
        })
      }
      return text
    },
    [locale]
  )

  return <I18nContext.Provider value={{ t, locale, setLocale }}>{children}</I18nContext.Provider>
}

export function useI18n() {
  const context = useContext(I18nContext)
  if (!context) {
    throw new Error('useI18n must be used within I18nProvider')
  }
  return context
}
