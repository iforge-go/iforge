'use client'

import { Avatar, HStack, Icon, Menu, MenuButton, MenuItem, MenuList, Text } from '@chakra-ui/react'
import { useRouter, usePathname } from 'next/navigation'
import { FiUser, FiSettings, FiEdit2, FiLogOut, FiBook, FiStar, FiFolder, FiCheckSquare } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'

export function UserMenu() {
  const router = useRouter()
  const pathname = usePathname()
  const { user, setUser } = useCurrentUser()
  const { t } = useI18n()

  const isScrumMode = pathname.startsWith('/pms') || pathname.startsWith('/projects')

  const handleLogout = () => {
    localStorage.removeItem('token')
    localStorage.removeItem('cached_user')
    setUser(null)
    window.dispatchEvent(new Event('auth-change'))
    router.push('/login')
  }

  if (!user) return null

  return (
    <Menu>
      <MenuButton>
        <HStack spacing={2}>
          <Avatar size="sm" name={user.fullName || user.userName} src={user.image || undefined} />
          <Text fontSize="sm" color="myGray.700" display={{ base: 'none', md: 'block' }}>
            {user.userName}
          </Text>
        </HStack>
      </MenuButton>
      <MenuList>
        <MenuItem onClick={() => router.push(`/${user.userName}`)}>
          <Icon as={FiUser} mr={2} />
          {t('header.yourProfile')}
        </MenuItem>
        {isScrumMode ? (
          <>
            <MenuItem onClick={() => router.push('/pms')}>
              <Icon as={FiFolder} mr={2} />
              {t('header.myProjects')}
            </MenuItem>
            <MenuItem onClick={() => router.push('/pms?tab=tasks')}>
              <Icon as={FiCheckSquare} mr={2} />
              {t('header.myTasks')}
            </MenuItem>
          </>
        ) : (
          <>
            <MenuItem onClick={() => router.push(`/${user.userName}?tab=repositories`)}>
              <Icon as={FiBook} mr={2} />
              {t('header.yourRepos')}
            </MenuItem>
            <MenuItem onClick={() => router.push(`/${user.userName}?tab=stars`)}>
              <Icon as={FiStar} mr={2} />
              {t('header.yourStars')}
            </MenuItem>
          </>
        )}
        <MenuItem onClick={() => router.push('/settings')}>
          <Icon as={FiEdit2} mr={2} />
          {t('header.settings')}
        </MenuItem>
        {user.isAdmin && (
          <MenuItem onClick={() => router.push('/admin')}>
            <Icon as={FiSettings} mr={2} />
            {t('header.administration')}
          </MenuItem>
        )}
        <MenuItem onClick={handleLogout}>
          <Icon as={FiLogOut} mr={2} />
          {t('header.logout')}
        </MenuItem>
      </MenuList>
    </Menu>
  )
}
