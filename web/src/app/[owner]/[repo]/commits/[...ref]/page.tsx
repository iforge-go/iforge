'use client'

import { Spinner, VStack, Text } from '@chakra-ui/react'
import { useParams } from 'next/navigation'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { parseBranchAndPath, shouldWaitForBranches } from '@/lib/branchPath'
import { CommitsList } from '../CommitsList'

export default function CommitsByBranchPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const pathSegments = params.ref as string[]
  const { branches } = useRepo()

  const defaultBranch = branches.find(b => b.isDefault)?.name || branches[0]?.name || 'main'
  // 分支名可能含 "/"（如 feature/task-15），用已知分支列表反向匹配前缀来正确分离分支名
  const { ref: branch } = parseBranchAndPath(pathSegments, branches, defaultBranch)

  // 多段路径可能含 "/" 分支名，需等 branches 加载后才能正确解析
  if (shouldWaitForBranches(pathSegments, branches)) {
    return (
      <VStack spacing={2} py={8}>
        <Spinner size="xl" />
        <Text color="myGray.500">Loading...</Text>
      </VStack>
    )
  }

  return <CommitsList owner={owner} repoName={repoName} branch={branch} />
}
