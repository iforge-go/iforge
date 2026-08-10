'use client'

import { Box, Button, Container, Flex, HStack, Text } from '@chakra-ui/react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'
import { api } from '@/lib/api'
import { Logo } from '@/components/Logo'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import { useSiteSettings } from '@/contexts/SiteSettingsContext'
import { NavigationLinks } from './header/NavigationLinks'
import { HeaderSearch } from './header/HeaderSearch'
import { LanguageSwitcher } from './header/LanguageSwitcher'
import { NotificationPopover } from './header/NotificationPopover'
import { NewButton } from './header/NewButton'
import { UserMenu } from './header/UserMenu'

export function Header({ siteName = 'iForge' }: { siteName?: string }) {
  const router = useRouter()
  const { user } = useCurrentUser()
  const [unreadCount, setUnreadCount] = useState(0)
  const { t } = useI18n()
  const { settings } = useSiteSettings()

  // 初始加载（登录后调一次 API 拿当前未读数，避免 WS 连接前的空窗期）
  useEffect(() => {
    if (!user) {
      setUnreadCount(0)
      return
    }
    api.getUnreadNotificationCount().then((res) => setUnreadCount(res.count || 0)).catch(() => {})
  }, [user])

  return (
    <Box bg="white" borderBottom="1px solid" borderColor="myGray.200" position="sticky" top={0} zIndex={100}>
      <Container maxW="container.xl">
        <Flex h="56px" align="center" justify="space-between">
          {/* Left: Logo + Navigation */}
          <HStack spacing={6}>
            <Link href="/">
              <HStack spacing={2}>
                <Logo size={32} />
                <Text fontSize="xl" fontWeight="bold" color="primary.600">
                  {siteName}
                </Text>
              </HStack>
            </Link>
            <NavigationLinks />
          </HStack>

          {/* Center: Search */}
          <HeaderSearch />

          {/* Right: User Actions */}
          <HStack spacing={3}>
            <LanguageSwitcher />
            {user ? (
              <>
                <NotificationPopover unreadCount={unreadCount} onUnreadCountChange={setUnreadCount} />
                <NewButton />
                <UserMenu />
              </>
            ) : (
              <HStack spacing={2}>
                <Button size="sm" variant="whiteBase" onClick={() => router.push('/login')}>
                  {t('header.login')}
                </Button>
                {settings?.allowRegistration && (
                  <Button size="sm" variant="primary" onClick={() => router.push('/register')}>
                    {t('header.register')}
                  </Button>
                )}
              </HStack>
            )}
          </HStack>
        </Flex>
      </Container>
    </Box>
  )
}
