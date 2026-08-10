'use client'

import { Box, Text, HStack, Avatar, Code, Link } from '@chakra-ui/react'
import NextLink from 'next/link'
import { formatRelativeTime } from '@/lib/time'
import { useI18n } from '@/contexts/I18nContext'

interface Commit {
  id: string
  message: string
  author: string
  timestamp: string
}

interface CommitInfoBarProps {
  owner: string
  repo: string
  ref: string
  dirPath?: string
  commits: Commit[]
}

export function CommitInfoBar({ owner, repo, ref, dirPath, commits }: CommitInfoBarProps) {
  const { t } = useI18n()
  if (commits.length === 0) return null
  const c = commits[0]
  const href = `/${owner}/${repo}/commits/${ref}${dirPath ? '/' + dirPath : ''}`
  return (
    <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200" bg="white">
      <HStack spacing={3}>
        <Avatar size="xs" name={c.author} />
        <Link as={NextLink} href={href} _hover={{ textDecoration: 'none' }}>
          <Text fontSize="sm" fontWeight="medium" color="myGray.900" noOfLines={1} _hover={{ color: 'blue.500' }}>
            {c.message}
          </Text>
        </Link>
        <Text fontSize="xs" color="myGray.500" flexShrink={0}>
          {c.author} {t('common.committedAt')} {formatRelativeTime(c.timestamp, t)}
        </Text>
        <Link as={NextLink} href={href}>
          <Code fontSize="xs" colorScheme="gray" px={2} py={0.5} borderRadius="md" _hover={{ color: 'blue.500' }}>
            {c.id.substring(0, 7)}
          </Code>
        </Link>
      </HStack>
    </Box>
  )
}
