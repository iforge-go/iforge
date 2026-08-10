'use client'

import { Box } from '@chakra-ui/react'
import { usePathname } from 'next/navigation'
import { Header } from './Header'
import { Footer } from './Footer'
import { useSiteSettings } from '@/contexts/SiteSettingsContext'

export function ClientLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const { settings } = useSiteSettings()

  if (pathname?.startsWith('/setup')) {
    return <>{children}</>
  }

  return (
    <Box display="flex" flexDirection="column" minH="100vh">
      <Header siteName={settings?.siteName || 'iForge'} />
      <Box flex="1">
        {children}
      </Box>
      <Footer />
    </Box>
  )
}
