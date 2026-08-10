'use client'

import { Box, Input } from '@chakra-ui/react'
import { useRouter } from 'next/navigation'
import { useState } from 'react'
import { usePathname } from 'next/navigation'
import { useI18n } from '@/contexts/I18nContext'

export function HeaderSearch() {
  const router = useRouter()
  const pathname = usePathname()
  const { t } = useI18n()
  const [searchInput, setSearchInput] = useState('')

  const isScrumMode = pathname.startsWith('/pms') || pathname.startsWith('/projects')

  const handleSearchSubmit = (e: React.KeyboardEvent<HTMLInputElement>) => {
    const q = searchInput.trim()
    if (!q) return
    const target = isScrumMode ? `/pms?q=${encodeURIComponent(q)}` : `/repos?q=${encodeURIComponent(q)}`
    router.push(target)
  }

  return (
    <Box position="relative" maxW="400px" display={{ base: 'none', md: 'block' }}>
      <Input
        placeholder={t(isScrumMode ? 'header.searchProjects' : 'header.searchRepos')}
        border="1px solid"
        borderColor="myGray.200"
        _focus={{ borderColor: 'primary.500', boxShadow: '0px 0px 0px 2.4px rgba(51, 112, 255, 0.15)' }}
        size="sm"
        pl={9}
        value={searchInput}
        onChange={(e) => setSearchInput(e.target.value)}
        onKeyDown={handleSearchSubmit}
        aria-label={t(isScrumMode ? 'header.searchProjects' : 'header.searchRepos')}
      />
      <Box
        as="svg"
        w={4}
        h={4}
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth={2}
        position="absolute"
        left={3}
        top="50%"
        transform="translateY(-50%)"
        color="myGray.400"
        zIndex={2}
        pointerEvents="none"
      >
        <circle cx={11} cy={11} r={8} />
        <path d="m21 21-4.35-4.35" />
      </Box>
    </Box>
  )
}
