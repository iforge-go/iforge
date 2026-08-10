'use client'

import {
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  HStack,
  VStack,
  Text,
  Icon,
} from '@chakra-ui/react'
import Link from 'next/link'
import { FiFolder, FiFile } from 'react-icons/fi'
import { formatRelativeTime } from '@/lib/time'
import { useI18n } from '@/contexts/I18nContext'

export interface FileEntry {
  name: string
  path: string
  type: 'file' | 'dir'
  lastCommit?: {
    id?: string
    message?: string
    timestamp?: string
  }
}

interface FileListTableProps {
  files: FileEntry[]
  owner: string
  repo: string
  ref: string
  labels: {
    name: string
    lastCommitMessage: string
    lastCommitDate: string
    empty: string
  }
}

export function FileListTable({ files, owner, repo, ref, labels }: FileListTableProps) {
  const { t } = useI18n()
  const sortedFiles = [...files].sort((a, b) => {
    if (a.type === b.type) return a.name.localeCompare(b.name)
    return a.type === 'dir' ? -1 : 1
  })

  return (
    <Table variant="simple" size="sm" borderColor="myGray.200" sx={{ tableLayout: 'fixed', 'td': { borderColor: 'myGray.100' }, 'tbody tr': { bg: 'white' } }}>
      <Thead bg="myGray.100">
        <Tr>
          <Th color="myGray.600" w="40%" whiteSpace="nowrap">{labels.name}</Th>
          <Th color="myGray.600" w="45%" whiteSpace="nowrap">{labels.lastCommitMessage}</Th>
          <Th color="myGray.600" w="15%" isNumeric whiteSpace="nowrap">{labels.lastCommitDate}</Th>
        </Tr>
      </Thead>
      <Tbody>
        {sortedFiles.length === 0 ? (
          <Tr>
            <Td colSpan={3}>
              <VStack spacing={4} py={8}>
                <Icon as={FiFile} w={10} h={10} color="myGray.300" />
                <Text color="myGray.500" fontSize="sm">
                  {labels.empty}
                </Text>
              </VStack>
            </Td>
          </Tr>
        ) : (
          sortedFiles.map((file) => {
            const fileLink = file.type === 'dir'
              ? `/${owner}/${repo}/tree/${ref}/${file.path}`
              : `/${owner}/${repo}/blob/${ref}/${file.path}`

            return (
              <Tr key={file.path} _hover={{ bg: 'myGray.50' }}>
                <Td>
                  <a href={fileLink}>
                    <HStack spacing={2} cursor="pointer">
                      <Icon
                        as={file.type === 'dir' ? FiFolder : FiFile}
                        color={file.type === 'dir' ? 'primary.500' : 'myGray.500'}
                        w={4}
                        h={4}
                      />
                      <Text fontSize="sm" fontWeight="medium" color="primary.600" _hover={{ textDecoration: 'underline' }}>
                        {file.name}
                      </Text>
                    </HStack>
                  </a>
                </Td>
                <Td>
                  {file.lastCommit?.id && file.lastCommit?.message ? (
                    <Link href={`/${owner}/${repo}/commit/${file.lastCommit.id}?branch=${ref}`}>
                      <Text fontSize="sm" color="myGray.600" noOfLines={1} _hover={{ color: 'primary.500' }}>
                        {file.lastCommit.message}
                      </Text>
                    </Link>
                  ) : (
                    <Text fontSize="sm" color="myGray.600" noOfLines={1}>
                      {file.lastCommit?.message || '-'}
                    </Text>
                  )}
                </Td>
                <Td isNumeric>
                  <Text fontSize="xs" color="myGray.500">
                    {file.lastCommit?.timestamp ? formatRelativeTime(file.lastCommit.timestamp, t) : '-'}
                  </Text>
                </Td>
              </Tr>
            )
          })
        )}
      </Tbody>
    </Table>
  )
}
