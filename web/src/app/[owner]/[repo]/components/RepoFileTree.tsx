'use client'

import {
  Box,
  VStack,
} from '@chakra-ui/react'
import { CommitInfo, FileEntry } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { FileListTable } from '@/components/FileListTable'
import { CommitInfoBar } from '@/components/CommitInfoBar'

interface RepoFileTreeProps {
  owner: string
  repoName: string
  selectedBranch: string
  defaultBranch: string
  commits: CommitInfo[]
  files: FileEntry[]
}

export function RepoFileTree({
  owner,
  repoName,
  selectedBranch,
  defaultBranch,
  commits,
  files,
}: RepoFileTreeProps) {
  const { t } = useI18n()

  const ref = selectedBranch || defaultBranch

  return (
    <VStack spacing={4} align="stretch">
      {/* File list */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden">
        <CommitInfoBar
          owner={owner}
          repo={repoName}
          ref={ref}
          commits={commits}
        />

        <FileListTable
          files={files}
          owner={owner}
          repo={repoName}
          ref={ref}
          labels={{
            name: t('repo.fileName'),
            lastCommitMessage: t('repo.lastCommitMessage'),
            lastCommitDate: t('repo.lastCommitDate'),
            empty: t('repo.noFiles'),
          }}
        />
      </Box>
    </VStack>
  )
}
