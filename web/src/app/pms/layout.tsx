import type { Metadata } from 'next'
import { headers } from 'next/headers'
import zh from '@/locales/zh.json'
import en from '@/locales/en.json'

const translations: Record<'zh' | 'en', typeof zh> = { zh, en }

// /pms 主页的 title:显示 "项目管理"(中文)或 "PMS"(英文)
export async function generateMetadata(): Promise<Metadata> {
  const headersList = await headers()
  const locale = (headersList.get('x-locale') as 'zh' | 'en') || 'zh'
  const t = translations[locale]
  return {
    title: t.nav.pms,
  }
}

export default function Layout({ children }: { children: React.ReactNode }) {
  return children
}
