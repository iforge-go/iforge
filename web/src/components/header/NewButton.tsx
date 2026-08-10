'use client'

import { Button, Icon, Menu, MenuButton, MenuItem, MenuList } from '@chakra-ui/react'
import { useRouter, usePathname } from 'next/navigation'
import { FiPlus, FiChevronDown, FiBook, FiFolder } from 'react-icons/fi'
import { AiOutlineBank } from 'react-icons/ai'
import { useI18n } from '@/contexts/I18nContext'

export function NewButton() {
  const router = useRouter()
  const pathname = usePathname()
  const { t } = useI18n()

  const isScrumMode = pathname.startsWith('/pms') || pathname.startsWith('/projects')

  return (
    <Menu>
      <MenuButton as={Button} size="sm" variant="primary" leftIcon={<FiPlus />} rightIcon={<FiChevronDown />}>
        {t('header.new')}
      </MenuButton>
      <MenuList>
        {isScrumMode ? (
          <>
            <MenuItem onClick={() => router.push('/projects/new')}>
              <Icon as={FiFolder} mr={2} />
              {t('header.newProject')}
            </MenuItem>
          </>
        ) : (
          <>
            <MenuItem onClick={() => router.push('/new')}>
              <Icon as={FiBook} mr={2} />
              {t('header.newRepo')}
            </MenuItem>
            <MenuItem onClick={() => router.push('/organizations?modal=open')}>
              <Icon as={AiOutlineBank} mr={2} />
              {t('header.newOrganization')}
            </MenuItem>
          </>
        )}
      </MenuList>
    </Menu>
  )
}
