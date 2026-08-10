'use client'

import { HStack, Text } from '@chakra-ui/react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useI18n } from '@/contexts/I18nContext'

export function NavigationLinks() {
  const pathname = usePathname()
  const { t } = useI18n()

  const isActive = (path: string) => pathname === path || pathname.startsWith(path + '/')

  return (
    <HStack spacing={4} display={{ base: 'none', md: 'flex' }}>
      <Link href="/pms">
        <Text
          fontSize="sm"
          color={isActive('/pms') || isActive('/projects') ? 'primary.600' : 'myGray.600'}
          fontWeight={isActive('/pms') || isActive('/projects') ? 'medium' : 'normal'}
          _hover={{ color: 'primary.600' }}
          cursor="pointer"
        >
          {t('nav.pms')}
        </Text>
      </Link>
      <Link href="/vcs">
        <Text
          fontSize="sm"
          color={isActive('/vcs') ? 'primary.600' : 'myGray.600'}
          fontWeight={isActive('/vcs') ? 'medium' : 'normal'}
          _hover={{ color: 'primary.600' }}
          cursor="pointer"
        >
          {t('nav.vcs')}
        </Text>
      </Link>
    </HStack>
  )
}
