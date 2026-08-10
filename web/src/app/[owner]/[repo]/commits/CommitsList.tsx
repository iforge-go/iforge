'use client'

import {
  Box,
  Heading,
  Text,
  VStack,
  HStack,
  Icon,
  Avatar,
  Badge,
  Code,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useEffect, useState } from 'react'
import { api, CommitInfo } from '@/lib/api'
import { FiGitCommit } from 'react-icons/fi'
import { formatRelativeTime } from '@/lib/time'
import { useI18n } from '@/contexts/I18nContext'

interface CommitsListProps {
  owner: string
  repoName: string
  branch: string
}

export function CommitsList({ owner, repoName, branch }: CommitsListProps) {
  const { t } = useI18n()
  const [commits, setCommits] = useState<CommitInfo[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.listCommits(owner, repoName, branch)
      .then((data) => setCommits(data || []))
      .finally(() => setLoading(false))
  }, [owner, repoName, branch])

  if (loading) {
    return <Text>{t('common.loading')}</Text>
  }

  return (
    <VStack spacing={4} align="stretch">
      <HStack justify="space-between">
        <HStack spacing={4}>
          <Heading size="lg" color="myGray.900">
            {t('repo.commits')}
          </Heading>
          <Badge colorScheme="blue" fontSize="md">
            {branch}
          </Badge>
        </HStack>
      </HStack>

      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" borderColor="myGray.200" bg="white">
        <Table variant="simple" size="sm" borderColor="myGray.200" sx={{ 'tbody tr': { bg: 'white' } }}>
          <Thead bg="myGray.100">
            <Tr>
              <Th width="100px" borderColor="myGray.200">{t('repo.commitHash')}</Th>
              <Th borderColor="myGray.200">{t('repo.commitMessage')}</Th>
              <Th width="150px" borderColor="myGray.200">{t('repo.author')}</Th>
              <Th width="200px" borderColor="myGray.200">{t('repo.commitDate')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {commits.length === 0 ? (
              <Tr>
                <Td colSpan={4} textAlign="center" py={8}>
                  <VStack spacing={4}>
                    <Icon as={FiGitCommit} w={12} h={12} color="myGray.400" />
                    <Text color="myGray.600">{t('repo.noCommits')}</Text>
                  </VStack>
                </Td>
              </Tr>
            ) : (
              commits.map((commit) => (
                <Tr key={commit.id} _hover={{ bg: 'myGray.50' }}>
                  <Td>
                    <Link href={`/${owner}/${repoName}/commit/${commit.id}?branch=${branch}`}>
                      <Code colorScheme="gray" fontSize="xs" _hover={{ colorScheme: 'blue' }}>
                        {commit.id.substring(0, 7)}
                      </Code>
                    </Link>
                  </Td>
                  <Td>
                    <Link href={`/${owner}/${repoName}/commit/${commit.id}?branch=${branch}`}>
                      <Text fontWeight="medium" color="myGray.900" _hover={{ color: 'primary.500' }}>
                        {commit.message.split('\n')[0]}
                      </Text>
                      {commit.message.includes('\n\n') && (
                        <Text fontSize="xs" color="myGray.500" mt={1} noOfLines={2}>
                          {commit.message.split('\n\n').slice(1).join('\n\n')}
                        </Text>
                      )}
                    </Link>
                  </Td>
                  <Td>
                    <HStack spacing={2}>
                      <Avatar size="xs" name={commit.author} />
                      <Text fontSize="sm">{commit.author}</Text>
                    </HStack>
                  </Td>
                  <Td>
                    <Text fontSize="sm" color="myGray.600">
                      {formatRelativeTime(commit.timestamp, t)}
                    </Text>
                  </Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>
      </Box>
    </VStack>
  )
}
