'use client'

import { Box, VStack, HStack, Text, Icon } from '@chakra-ui/react'
import {
  FiHome,
  FiUsers,
  FiSettings,
  FiDatabase,
  FiPackage,
  FiChevronDown,
  FiChevronRight,
  FiMail,
  FiShield,
  FiGlobe,
  FiUpload,
  FiFolder,
  FiKey,
  FiCpu,
  FiFileText,
  FiZap,
} from 'react-icons/fi'
import { AiOutlineBank } from 'react-icons/ai'
import { useState } from 'react'
import { useI18n } from '@/contexts/I18nContext'

export type AdminPage =
  | 'overview'
  | 'users'
  | 'organizations'
  | 'repos'
  | 'runners'
  | 'audit'
  | 'settings-general'
  | 'settings-smtp'
  | 'settings-ldap'
  | 'settings-webhook'
  | 'settings-upload'
  | 'settings-repository'
  | 'settings-oidc'
  | 'settings-ai'
  | 'database'
  | 'plugins'

interface SidebarItem {
  key: AdminPage
  label: string
  icon: any
  children?: { key: AdminPage; label: string; icon: any }[]
}

interface AdminSidebarProps {
  activePage: AdminPage
  onNavigate: (page: AdminPage) => void
}

export default function AdminSidebar({ activePage, onNavigate }: AdminSidebarProps) {
  const { t } = useI18n()
  const [settingsExpanded, setSettingsExpanded] = useState(true)

  const items: SidebarItem[] = [
    { key: 'overview', label: t('admin.overview'), icon: FiHome },
    { key: 'users', label: t('admin.users'), icon: FiUsers },
    { key: 'organizations', label: t('admin.organizations'), icon: AiOutlineBank },
    { key: 'repos', label: t('admin.repos'), icon: FiFolder },
    { key: 'runners', label: t('admin.runners'), icon: FiZap },
    {
      key: 'settings-general',
      label: t('admin.settings'),
      icon: FiSettings,
      children: [
        { key: 'settings-general', label: t('admin.settingsGeneral'), icon: FiSettings },
        { key: 'settings-smtp', label: t('admin.settingsSMTP'), icon: FiMail },
        { key: 'settings-ldap', label: t('admin.settingsLDAP'), icon: FiShield },
        { key: 'settings-webhook', label: t('admin.settingsWebhook'), icon: FiGlobe },
        { key: 'settings-upload', label: t('admin.settingsUpload'), icon: FiUpload },
        { key: 'settings-repository', label: t('admin.settingsRepository'), icon: FiFolder },
        { key: 'settings-oidc', label: t('admin.settingsOIDC'), icon: FiKey },
        { key: 'settings-ai', label: t('admin.settingsAI'), icon: FiCpu },
      ],
    },
    { key: 'database', label: t('admin.database'), icon: FiDatabase },
    { key: 'plugins', label: t('admin.plugins'), icon: FiPackage },
    { key: 'audit', label: t('admin.auditLogs'), icon: FiFileText },
  ]

  const isActive = (key: AdminPage) => activePage === key

  return (
    <Box
      w="220px"
      bg="white"
      borderWidth="1px"
      borderColor="myGray.200"
      borderRadius="lg"
      overflow="hidden"
      flexShrink={0}
    >
      <VStack align="stretch" spacing={0} py={2}>
        {items.map((item) => {
          if (item.children) {
            const isSettingsGroup = item.key === 'settings-general'
            const hasActiveChild = item.children.some((child) => isActive(child.key))

            return (
              <Box key={item.key}>
                <HStack
                  px={4}
                  py={2.5}
                  cursor="pointer"
                  bg={hasActiveChild ? 'primary.50' : 'transparent'}
                  color={hasActiveChild ? 'primary.600' : 'myGray.700'}
                  _hover={{ bg: 'myGray.50' }}
                  onClick={() => setSettingsExpanded(!settingsExpanded)}
                  spacing={3}
                >
                  <Icon as={item.icon} w={4} h={4} />
                  <Text fontSize="sm" fontWeight="medium" flex={1}>
                    {item.label}
                  </Text>
                  <Icon as={settingsExpanded ? FiChevronDown : FiChevronRight} w={3} h={3} />
                </HStack>
                {settingsExpanded && isSettingsGroup && (
                  <VStack align="stretch" spacing={0} pl={4}>
                    {item.children.map((child) => (
                      <HStack
                        key={child.key}
                        px={4}
                        py={2}
                        cursor="pointer"
                        bg={isActive(child.key) ? 'primary.50' : 'transparent'}
                        color={isActive(child.key) ? 'primary.600' : 'myGray.600'}
                        _hover={{ bg: 'myGray.50' }}
                        onClick={() => onNavigate(child.key)}
                        spacing={3}
                      >
                        <Icon as={child.icon} w={3.5} h={3.5} />
                        <Text fontSize="sm">{child.label}</Text>
                      </HStack>
                    ))}
                  </VStack>
                )}
              </Box>
            )
          }

          return (
            <HStack
              key={item.key}
              px={4}
              py={2.5}
              cursor="pointer"
              bg={isActive(item.key) ? 'primary.50' : 'transparent'}
              color={isActive(item.key) ? 'primary.600' : 'myGray.700'}
              _hover={{ bg: 'myGray.50' }}
              onClick={() => onNavigate(item.key)}
              spacing={3}
            >
              <Icon as={item.icon} w={4} h={4} />
              <Text fontSize="sm" fontWeight="medium">
                {item.label}
              </Text>
            </HStack>
          )
        })}
      </VStack>
    </Box>
  )
}
