'use client'

import {
  Box,
  Text,
  Button,
  VStack,
  HStack,
  Input,
  Textarea,
  Icon,
  Select,
  Code,
  Spinner,
} from '@chakra-ui/react'
import { useGithubToast } from '@/app/providers'
import { useParams, useRouter, useSearchParams } from 'next/navigation'
import { Suspense, useState, useEffect, useRef } from 'react'
import { api } from '@/lib/api'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useI18n } from '@/contexts/I18nContext'
import { FiGitBranch, FiGitPullRequest, FiFile, FiPlus, FiMinus, FiCheckCircle, FiAlertTriangle } from 'react-icons/fi'
import type { Repository } from '@/lib/types'

interface CompareResult {
  commits: Array<{ id: string; message: string; author: string; timestamp: string }>
  fileChanges: Array<{ filename: string; additions: number; deletions: number; status: string }>
  baseCommit: string
  headCommit: string
  mergeable?: boolean
}

function NewMergeRequestContent() {
  const params = useParams()
  const router = useRouter()
  const searchParams = useSearchParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const { branches, refreshData } = useRepo()
  const toast = useGithubToast()
  const { t } = useI18n()

  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [head, setHead] = useState('')
  const [base, setBase] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [compareResult, setCompareResult] = useState<CompareResult | null>(null)
  const [loadingCompare, setLoadingCompare] = useState(false)
  const titleAutoFilled = useRef(false)

  // Cross-repo MR support
  const [headRepoOwner, setHeadRepoOwner] = useState(owner)
  const [headRepoName, setHeadRepoName] = useState(repoName)
  const [forks, setForks] = useState<Repository[]>([])
  const [headBranches, setHeadBranches] = useState(branches)
  const [loadingForks, setLoadingForks] = useState(false)

  // Load forks on mount
  useEffect(() => {
    const loadForks = async () => {
      setLoadingForks(true)
      try {
        const forkList = await api.listForks(owner, repoName)
        setForks(forkList || [])
      } catch (error) {
        console.error('Failed to load forks:', error)
      } finally {
        setLoadingForks(false)
      }
    }
    loadForks()
  }, [owner, repoName])

  // Load branches when head repo changes
  useEffect(() => {
    const loadBranches = async () => {
      if (headRepoOwner === owner && headRepoName === repoName) {
        setHeadBranches(branches)
      } else {
        try {
          const branchList = await api.listBranches(headRepoOwner, headRepoName)
          setHeadBranches(branchList || [])
        } catch (error) {
          console.error('Failed to load branches:', error)
          setHeadBranches([])
        }
      }
    }
    loadBranches()
  }, [headRepoOwner, headRepoName, owner, repoName, branches])

  const fetchCompare = async (headBranch: string, baseBranch: string) => {
    if (!headBranch || !baseBranch) {
      setCompareResult(null)
      return
    }
    if (headBranch === baseBranch && headRepoOwner === owner && headRepoName === repoName) {
      setCompareResult({
        commits: [],
        fileChanges: [],
        baseCommit: '',
        headCommit: '',
        mergeable: false,
      })
      return
    }
    setLoadingCompare(true)
    try {
      const isCrossRepo = headRepoOwner !== owner || headRepoName !== repoName
      const result = await api.compare(
        owner,
        repoName,
        baseBranch,
        headBranch,
        isCrossRepo ? headRepoOwner : undefined,
        isCrossRepo ? headRepoName : undefined
      )
      setCompareResult(result)
      // 首次加载时自动填充标题和描述
      if (!titleAutoFilled.current) {
        // 标题：使用第一个（最新）提交的消息
        if (result.commits && result.commits.length > 0) {
          const firstCommitMsg = result.commits[0].message.split('\n')[0]
          setTitle(firstCommitMsg)
        } else {
          setTitle(`Merge branch '${headBranch}' into ${baseBranch}`)
        }
        // 描述：列出所有提交
        if (result.commits && result.commits.length > 0) {
          const commitList = result.commits
            .map((c) => `- ${c.message.split('\n')[0]} (${c.author}, ${c.id.substring(0, 7)})`)
            .join('\n')
          setBody(`### Commits\n\n${commitList}`)
        }
        titleAutoFilled.current = true
      }
    } catch {
      setCompareResult(null)
    } finally {
      setLoadingCompare(false)
    }
  }

  useEffect(() => {
    const headParam = searchParams.get('head')
    const baseParam = searchParams.get('base')
    const headUserParam = searchParams.get('headUser')
    const headRepoParam = searchParams.get('headRepo')
    if (headParam) setHead(headParam)
    if (baseParam) setBase(baseParam)
    if (headUserParam && headRepoParam) {
      setHeadRepoOwner(headUserParam)
      setHeadRepoName(headRepoParam)
    }
    if (branches && branches.length > 0) {
      const defaultBranch = branches.find((b) => b.isDefault)?.name || branches[0]?.name || ''
      if (!baseParam) setBase(defaultBranch)
      if (!headParam) {
        const nonDefault = branches.find((b) => b.name !== defaultBranch)
        if (nonDefault) setHead(nonDefault.name)
      }
    }
  }, [searchParams, branches])

  useEffect(() => {
    if (head && base) {
      fetchCompare(head, base)
    }
  }, [head, base, headRepoOwner, headRepoName])

  const handleSubmit = async () => {
    if (!title.trim() || !head || !base) return
    try {
      setSubmitting(true)
      const isCrossRepo = headRepoOwner !== owner || headRepoName !== repoName
      const mr = await api.createMergeRequest(
        owner,
        repoName,
        title,
        body,
        head,
        base,
        isCrossRepo ? headRepoOwner : undefined,
        isCrossRepo ? headRepoName : undefined
      )
      await refreshData()
      toast({
        title: t('repo.mrCreated'),
        status: 'success',
        duration: 2000,
      })
      router.push(`/${owner}/${repoName}/merge-requests/${mr.issueId}`)
    } catch (error: any) {
      toast({
        title: t('repo.createFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSubmitting(false)
    }
  }

  const totalAdditions = compareResult?.fileChanges?.reduce((sum, f) => sum + f.additions, 0) || 0
  const totalDeletions = compareResult?.fileChanges?.reduce((sum, f) => sum + f.deletions, 0) || 0

  return (
    <VStack spacing={4} align="stretch">
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
        <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200">
          <HStack spacing={2}>
            <Icon as={FiGitPullRequest} color="myGray.500" />
            <Text fontSize="sm" fontWeight="medium">{t('repo.newMergeRequest')}</Text>
          </HStack>
        </Box>
        <Box p={4}>
          <VStack spacing={4} align="stretch">
            <HStack spacing={3} align="flex-end">
              <Box flex={1}>
                <Text fontSize="sm" fontWeight="medium" mb={1}>{t('repo.sourceBranch')}</Text>
                <Select
                  value={head}
                  onChange={(e) => setHead(e.target.value)}
                  size="sm"
                >
                  {headBranches?.map((b) => (
                    <option key={b.name} value={b.name}>{b.name}</option>
                  ))}
                </Select>
              </Box>
              <Text fontSize="sm" color="myGray.500" pb={2}>→</Text>
              <Box flex={1}>
                <Text fontSize="sm" fontWeight="medium" mb={1}>{t('repo.targetBranch')}</Text>
                <Select
                  value={base}
                  onChange={(e) => setBase(e.target.value)}
                  size="sm"
                >
                  {branches?.map((b) => (
                    <option key={b.name} value={b.name}>{b.name}</option>
                  ))}
                </Select>
              </Box>
            </HStack>

            {/* Repository selector for cross-repo MR */}
            <Box>
              <Text fontSize="sm" fontWeight="medium" mb={1}>{t('repo.sourceRepository')}</Text>
              <Select
                value={`${headRepoOwner}/${headRepoName}`}
                onChange={(e) => {
                  const [repoOwner, repo] = e.target.value.split('/')
                  setHeadRepoOwner(repoOwner)
                  setHeadRepoName(repo)
                }}
                size="sm"
                isDisabled={loadingForks}
              >
                <option value={`${owner}/${repoName}`}>{owner}/{repoName} (this repository)</option>
                {forks.map((fork) => (
                  <option key={fork.userName + '/' + fork.repositoryName} value={`${fork.userName}/${fork.repositoryName}`}>
                    {fork.userName}/{fork.repositoryName}
                  </option>
                ))}
              </Select>
            </Box>

            {compareResult && !loadingCompare && (
              <HStack spacing={2} px={3} py={2} borderRadius="md" bg={compareResult.mergeable ? 'green.50' : 'yellow.50'}>
                <Icon
                  as={compareResult.mergeable ? FiCheckCircle : FiAlertTriangle}
                  color={compareResult.mergeable ? 'green.600' : 'yellow.600'}
                  w={4}
                  h={4}
                />
                <Text fontSize="sm" color={compareResult.mergeable ? 'green.700' : 'yellow.700'}>
                  {compareResult.mergeable
                    ? t('repo.ableToMerge')
                    : head === base
                      ? t('repo.sameBranch')
                      : t('repo.hasConflicts')}
                </Text>
              </HStack>
            )}

            <Box>
              <Text fontSize="sm" fontWeight="medium" mb={1}>{t('repo.title')}</Text>
              <Input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder={t('repo.mrTitlePlaceholder')}
                size="sm"
              />
            </Box>

            <Box>
              <Text fontSize="sm" fontWeight="medium" mb={1}>{t('repo.descriptionOptional')}</Text>
              <Textarea
                value={body}
                onChange={(e) => setBody(e.target.value)}
                placeholder={t('repo.addDescription')}
                size="sm"
                rows={4}
              />
            </Box>

            <HStack justify="flex-end">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => router.back()}
              >
                {t('repo.cancel')}
              </Button>
              <Button
                variant="primary"
                size="sm"
                onClick={handleSubmit}
                isLoading={submitting}
                isDisabled={!title.trim() || !head || !base}
              >
                {t('repo.createMergeRequest')}
              </Button>
            </HStack>
          </VStack>
        </Box>
      </Box>

      {loadingCompare && (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
          <HStack spacing={3} justify="center">
            <Spinner size="sm" />
            <Text fontSize="sm" color="myGray.500">{t('repo.loadingCommits')}</Text>
          </HStack>
        </Box>
      )}

      {!loadingCompare && compareResult && compareResult.commits?.length > 0 && (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
          <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200">
            <HStack spacing={2}>
              <Icon as={FiGitBranch} color="myGray.500" w={4} h={4} />
              <Text fontSize="sm" fontWeight="medium">
                {t('repo.commitCount', { count: compareResult.commits?.length || 0 })}
              </Text>
              <Text fontSize="xs" color="myGray.500">
                ({t('repo.additionsCount', { count: totalAdditions })}, {t('repo.deletionsCount', { count: totalDeletions })})
              </Text>
            </HStack>
          </Box>

          <Box>
            {compareResult.commits?.map((commit) => (
              <Box
                key={commit.id}
                px={4}
                py={2}
                borderBottom="1px"
                borderColor="myGray.100"
                _last={{ borderBottom: 'none' }}
              >
                <HStack spacing={2}>
                  <Code fontSize="xs" colorScheme="gray" px={1.5} py={0.5}>
                    {commit.id.substring(0, 7)}
                  </Code>
                  <Text fontSize="sm" color="myGray.700" noOfLines={1} flex={1}>
                    {commit.message.split('\n')[0]}
                  </Text>
                  <Text fontSize="xs" color="myGray.500">
                    {commit.author}
                  </Text>
                </HStack>
              </Box>
            ))}
          </Box>

          {compareResult.fileChanges?.length > 0 && (
            <Box borderTop="1px" borderColor="myGray.200">
              <Box px={4} py={2} bg="myGray.50">
                <Text fontSize="xs" fontWeight="medium" color="myGray.600">
                  {t('repo.fileChangesCount', { count: compareResult.fileChanges.length })}
                </Text>
              </Box>
              {compareResult.fileChanges?.map((file) => (
                <Box
                  key={file.filename}
                  px={4}
                  py={1.5}
                  borderBottom="1px"
                  borderColor="myGray.100"
                  _last={{ borderBottom: 'none' }}
                >
                  <HStack spacing={2}>
                    <Icon as={FiFile} w={3} h={3} color="myGray.500" />
                    <Text fontSize="xs" color="myGray.700" flex={1} noOfLines={1}>
                      {file.filename}
                    </Text>
                    <HStack spacing={1}>
                      {file.additions > 0 && (
                        <HStack spacing={0.5}>
                          <Icon as={FiPlus} w={3} h={3} color="green.600" />
                          <Text fontSize="xs" color="green.600" fontWeight="medium">
                            {file.additions}
                          </Text>
                        </HStack>
                      )}
                      {file.deletions > 0 && (
                        <HStack spacing={0.5}>
                          <Icon as={FiMinus} w={3} h={3} color="red.600" />
                          <Text fontSize="xs" color="red.600" fontWeight="medium">
                            {file.deletions}
                          </Text>
                        </HStack>
                      )}
                    </HStack>
                  </HStack>
                </Box>
              ))}
            </Box>
          )}
        </Box>
      )}

      {!loadingCompare && compareResult && compareResult.commits?.length === 0 && head !== base && (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
          <Text fontSize="sm" color="myGray.500" textAlign="center">
            {t('repo.noDiffBetweenBranches')}
          </Text>
        </Box>
      )}
    </VStack>
  )
}

export default function NewMergeRequestPage() {
  return (
    <Suspense fallback={null}>
      <NewMergeRequestContent />
    </Suspense>
  )
}
