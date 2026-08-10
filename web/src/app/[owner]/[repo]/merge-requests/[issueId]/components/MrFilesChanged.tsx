'use client'

import {
  Box,
  Text,
  HStack,
  VStack,
  Badge,
  Avatar,
  IconButton,
} from '@chakra-ui/react'
import { useState } from 'react'
import { useI18n } from '@/contexts/I18nContext'
import { parsePatch } from '@/lib/diff'
import { SplitDiffView } from '@/components/SplitDiffView'
import { ReviewComment } from '@/lib/api'
import { FiChevronDown, FiChevronRight } from 'react-icons/fi'

interface FileChange {
  filename: string
  additions: number
  deletions: number
  status: string
  patch?: string
}

interface MrFilesChangedProps {
  fileChanges: FileChange[]
  reviewComments: ReviewComment[]
  dateLocale: string
}

function FileChangeItem({
  file,
  fileComments,
  dateLocale,
}: {
  file: FileChange
  fileComments: ReviewComment[]
  dateLocale: string
}) {
  const { t } = useI18n()
  const [expanded, setExpanded] = useState(false)
  const hasComments = fileComments.length > 0
  const fileDiff = expanded && file.patch
    ? parsePatch(file.filename, file.status, file.patch)
    : null

  return (
    <Box>
      {/* File header — always rendered, lightweight */}
      <Box
        borderWidth="1px"
        borderRadius="md"
        borderColor="myGray.200"
        overflow="hidden"
        bg="white"
        cursor="pointer"
        onClick={() => setExpanded(!expanded)}
        _hover={{ bg: 'myGray.50' }}
      >
        <Box px={4} py={2} borderBottom={expanded ? '1px solid' : 'none'} borderColor="myGray.200" bg="myGray.50">
          <HStack spacing={3}>
            <IconButton
              aria-label={expanded ? 'collapse' : 'expand'}
              icon={<Box as={expanded ? FiChevronDown : FiChevronRight} w={3.5} h={3.5} />}
              size="xs"
              variant="ghost"
              onClick={(e) => { e.stopPropagation(); setExpanded(!expanded) }}
            />
            <Text fontSize="sm" fontFamily="mono" color="myGray.900" flex={1}>
              {file.filename}
            </Text>
            <HStack spacing={1}>
              {file.additions > 0 && (
                <Text fontSize="xs" color="green.600" fontWeight="medium">+{file.additions}</Text>
              )}
              {file.deletions > 0 && (
                <Text fontSize="xs" color="red.600" fontWeight="medium">-{file.deletions}</Text>
              )}
            </HStack>
            {hasComments && (
              <Badge colorScheme="blue" fontSize="xs">
                {fileComments.length}
              </Badge>
            )}
          </HStack>
        </Box>
        {/* Diff body — only rendered when expanded, avoids massive DOM when collapsed */}
        {expanded && fileDiff && (
          <Box px={2} py={2}>
            <SplitDiffView
              filename={fileDiff.filename}
              status={fileDiff.status}
              additions={fileDiff.additions}
              deletions={fileDiff.deletions}
              hunks={fileDiff.hunks}
            />
          </Box>
        )}
        {expanded && !file.patch && (
          <Box px={4} py={3} textAlign="center">
            <Text fontSize="sm" color="myGray.500">{t('repo.cannotDisplayDiff')}</Text>
          </Box>
        )}
      </Box>
      {/* Inline comments */}
      {hasComments && (
        <Box mt={2} p={3} bg="myGray.50" borderWidth="1px" borderColor="myGray.200" borderRadius="md">
          <VStack spacing={2} align="stretch">
            <Text fontSize="xs" fontWeight="medium" color="myGray.600">
              {t('repo.inlineComments')}
            </Text>
            {fileComments.map((c) => (
              <Box key={c.commentId} p={2} bg="white" borderWidth="1px" borderColor="myGray.200" borderRadius="md">
                <HStack spacing={2} mb={1}>
                  <Avatar size="2xs" name={c.commenter} />
                  <Text fontSize="xs" fontWeight="medium" color="myGray.900">
                    {c.commenter}
                  </Text>
                  <Badge fontSize="xs" colorScheme="gray">
                    {t('repo.line', { line: c.line })}
                  </Badge>
                  <Text fontSize="xs" color="myGray.500">
                    {new Date(c.registeredDate).toLocaleDateString(dateLocale)}
                  </Text>
                </HStack>
                <Text fontSize="xs" whiteSpace="pre-wrap" color="myGray.700">
                  {c.content}
                </Text>
              </Box>
            ))}
          </VStack>
        </Box>
      )}
    </Box>
  )
}

export function MrFilesChanged({ fileChanges, reviewComments, dateLocale }: MrFilesChangedProps) {
  const { t } = useI18n()

  return (
    <>
      {fileChanges.length > 0 ? (
        <VStack spacing={4} align="stretch" p={4}>
          {fileChanges.map((file) => {
            const fileComments = reviewComments.filter((c) => c.filePath === file.filename)
            return (
              <FileChangeItem
                key={file.filename}
                file={file}
                fileComments={fileComments}
                dateLocale={dateLocale}
              />
            )
          })}
        </VStack>
      ) : (
        <Box px={4} py={8} textAlign="center">
          <Text fontSize="sm" color="myGray.500">{t('repo.noFilesChanged')}</Text>
        </Box>
      )}
    </>
  )
}
