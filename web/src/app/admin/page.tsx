'use client'

import { Box, Container, Heading, HStack, Text } from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import AdminSidebar from './components/AdminSidebar'
import type { AdminPage } from './components/AdminSidebar'
import AdminOverview from './components/AdminOverview'
import AdminUsers from './components/AdminUsers'
import AdminOrganizations from './components/AdminOrganizations'
import AdminRepos from './components/AdminRepos'
import AdminRunners from './components/AdminRunners'
import AdminAudit from './components/AdminAudit'
import AdminDatabase from './components/AdminDatabase'
import AdminPlugins from './components/AdminPlugins'
import GeneralSettings from './components/settings/GeneralSettings'
import SMTPSettings from './components/settings/SMTPSettings'
import LDAPSettings from './components/settings/LDAPSettings'
import WebhookSettings from './components/settings/WebhookSettings'
import UploadSettings from './components/settings/UploadSettings'
import RepositorySettings from './components/settings/RepositorySettings'
import OIDCSettings from './components/settings/OIDCSettings'
import AISettings from './components/settings/AISettings'
import { api, SystemInfo } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'

export default function AdminPage() {
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()
  const [activePage, setActivePage] = useState<AdminPage>('overview')
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null)
  const [uptimeSeconds, setUptimeSeconds] = useState(0)

  useEffect(() => {
    if (!user?.isAdmin) return
    const loadSystemInfo = async () => {
      try {
        const data = await api.getSystemInfo()
        setSystemInfo(data)
        if (data?.uptime?.seconds) {
          setUptimeSeconds(data.uptime.seconds)
        }
      } catch (error) {
        console.error('Failed to load system info:', error)
      }
    }
    loadSystemInfo()
  }, [user?.isAdmin])

  useEffect(() => {
    if (uptimeSeconds <= 0) return
    const timer = setInterval(() => {
      setUptimeSeconds(s => s + 1)
    }, 1000)
    return () => clearInterval(timer)
  }, [uptimeSeconds])

  // Guard: show nothing while auth is loading (prevents flash of admin UI)
  if (authLoading) {
    return null
  }

  // Guard: not logged in or non-admin
  if (!user || !user.isAdmin) {
    return (
      <Box bg="myGray.50" minH="100vh">
        <Container maxW="container.xl" py={8}>
          <Text color="myGray.600" fontSize="lg">{t('common.forbidden')}</Text>
        </Container>
      </Box>
    )
  }

  const renderContent = () => {
    switch (activePage) {
      case 'overview':
        return <AdminOverview systemInfo={systemInfo} uptimeSeconds={uptimeSeconds} />
      case 'users':
        return <AdminUsers />
      case 'organizations':
        return <AdminOrganizations />
      case 'repos':
        return <AdminRepos />
      case 'runners':
        return <AdminRunners />
      case 'audit':
        return <AdminAudit />
      case 'settings-general':
        return <GeneralSettings />
      case 'settings-smtp':
        return <SMTPSettings />
      case 'settings-ldap':
        return <LDAPSettings />
      case 'settings-webhook':
        return <WebhookSettings />
      case 'settings-upload':
        return <UploadSettings />
      case 'settings-repository':
        return <RepositorySettings />
      case 'settings-oidc':
        return <OIDCSettings />
      case 'settings-ai':
        return <AISettings />
      case 'database':
        return <AdminDatabase />
      case 'plugins':
        return <AdminPlugins />
      default:
        return <AdminOverview systemInfo={systemInfo} uptimeSeconds={uptimeSeconds} />
    }
  }

  return (
    <Box bg="myGray.50" minH="100vh">
      <Container maxW="container.xl" py={8}>
        <HStack spacing={6} align="flex-start">
          <AdminSidebar activePage={activePage} onNavigate={setActivePage} />
          <Box flex={1}>
            <Heading size="lg" color="myGray.900" mb={6}>
              {t('admin.title')}
            </Heading>
            {renderContent()}
          </Box>
        </HStack>
      </Container>
    </Box>
  )
}
