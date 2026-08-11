'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  VStack,
  HStack,
  Icon,
  Tabs,
  TabList,
  Tab,
  TabPanels,
  TabPanel,
} from '@chakra-ui/react'
import { useRouter, useSearchParams } from 'next/navigation'
import { useMemo, Suspense } from 'react'
import { FiUser, FiKey, FiSliders, FiShield, FiLock, FiAward } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import ProfileTab from './components/ProfileTab'
import AccountTab from './components/AccountTab'
import PreferencesTab from './components/PreferencesTab'
import SshKeysTab from './components/SshKeysTab'
import AccessTokensTab from './components/AccessTokensTab'
import GpgKeysTab from './components/GpgKeysTab'

// Tab 索引映射
const TAB_MAP = ['profile', 'account', 'preferences', 'sshkeys', 'tokens', 'gpgkeys']

// 内部组件，使用 useSearchParams
function SettingsContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()

  // 从 URL 读取当前 tab
  const currentTab = searchParams.get('tab') || 'profile'
  const tabIndex = useMemo(() => {
    const index = TAB_MAP.indexOf(currentTab)
    return index >= 0 ? index : 0
  }, [currentTab])

  // 处理 tab 切换
  const handleTabChange = (index: number) => {
    const newTab = TAB_MAP[index]
    if (newTab) {
      router.push(`/settings?tab=${newTab}`)
    }
  }

  if (authLoading) return null

  if (!user) {
    return (
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={8}>
          <Text>{t('common.pleaseLogin')}</Text>
        </Container>
      </Box>
    )
  }

  return (
    <Box bg="myGray.50">
      <Container maxW="container.xl" py={8}>
        <VStack spacing={6} align="stretch">
          <Heading size="lg" color="myGray.900">
            {t('settings.title')}
          </Heading>

          <Tabs
            variant="line"
            colorScheme="primary"
            index={tabIndex}
            onChange={handleTabChange}
            isLazy
          >
            <TabList borderBottomColor="myGray.200">
              <Tab>
                <HStack spacing={2}>
                  <Icon as={FiUser} />
                  <Text>{t('settings.profile')}</Text>
                </HStack>
              </Tab>
              <Tab>
                <HStack spacing={2}>
                  <Icon as={FiKey} />
                  <Text>{t('settings.account')}</Text>
                </HStack>
              </Tab>
              <Tab>
                <HStack spacing={2}>
                  <Icon as={FiSliders} />
                  <Text>{t('settings.preferences')}</Text>
                </HStack>
              </Tab>
              <Tab>
                <HStack spacing={2}>
                  <Icon as={FiShield} />
                  <Text>{t('settings.sshKeys')}</Text>
                </HStack>
              </Tab>
              <Tab>
                <HStack spacing={2}>
                  <Icon as={FiLock} />
                  <Text>{t('settings.accessTokens')}</Text>
                </HStack>
              </Tab>
              <Tab>
                <HStack spacing={2}>
                  <Icon as={FiAward} />
                  <Text>{t('settings.gpgKeys')}</Text>
                </HStack>
              </Tab>
            </TabList>

            <TabPanels>
              {/* 个人资料 */}
              <TabPanel px={0} pt={6}>
                <ProfileTab user={user} />
              </TabPanel>

              {/* 账户（密码 + 邮箱） */}
              <TabPanel px={0} pt={6}>
                <AccountTab user={user} isActive={currentTab === 'account'} />
              </TabPanel>

              {/* 偏好设置 */}
              <TabPanel px={0} pt={6}>
                <PreferencesTab isActive={currentTab === 'preferences'} />
              </TabPanel>

              {/* SSH 密钥 */}
              <TabPanel px={0} pt={6}>
                <SshKeysTab isActive={currentTab === 'sshkeys'} />
              </TabPanel>

              {/* 访问令牌 */}
              <TabPanel px={0} pt={6}>
                <AccessTokensTab isActive={currentTab === 'tokens'} />
              </TabPanel>

              {/* GPG 密钥 */}
              <TabPanel px={0} pt={6}>
                <GpgKeysTab isActive={currentTab === 'gpgkeys'} />
              </TabPanel>
            </TabPanels>
          </Tabs>
        </VStack>
      </Container>
    </Box>
  )
}

// 主页面组件，用 Suspense 包裹
export default function SettingsPage() {
  const { t } = useI18n()
  return (
    <Suspense fallback={
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={8}>
          <Text>{t('common.loading')}</Text>
        </Container>
      </Box>
    }>
      <SettingsContent />
    </Suspense>
  )
}
