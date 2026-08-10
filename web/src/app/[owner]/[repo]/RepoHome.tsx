'use client'

import {
  Box,
  Text,
  Button,
  VStack,
  HStack,
  Icon,
  Input,
  InputGroup,
  InputLeftElement,
  InputRightElement,
  IconButton,
  Code,
  useClipboard,
  Heading,
  Tabs,
  TabList,
  Tab,
  TabPanels,
  TabPanel,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverBody,
  PopoverArrow,
  PopoverCloseButton,
  Spinner,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useEffect, useState, useRef } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { api, CommitInfo, FileEntry, Tag, SERVER_BASE } from '@/lib/api'
import {
  FiGitBranch,
  FiCopy,
  FiCheck,
  FiDownload,
  FiSearch,
  FiFile,
  FiChevronRight,
  FiGitPullRequest,
  FiArchive,
  FiRss,
  FiUpload,
  FiPackage,
} from 'react-icons/fi'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { RepoFileTree } from '@/app/[owner]/[repo]/components/RepoFileTree'
import { RepoReadme } from '@/app/[owner]/[repo]/components/RepoReadme'

export default function RepoCodePage() {
  const params = useParams()
  const router = useRouter()
  const owner = params.owner as string
  const repoName = params.repo as string
  const { repo, branches, refreshData, userRole, canCreateIssue } = useRepo()
  const toast = useGithubToast()
  const { t } = useI18n()

  // 权限判断
  const canCreateBranch = userRole === 'owner' || userRole === 'member'
  const canEditFile = userRole === 'owner' || userRole === 'member'

  const [commits, setCommits] = useState<CommitInfo[]>([])
  const [tags, setTags] = useState<Tag[]>([])
  const [files, setFiles] = useState<FileEntry[]>([])
  const [readmeContent, setReadmeContent] = useState<string>('')
  const [readmeRaw, setReadmeRaw] = useState<string>('')
  const [selectedBranch, setSelectedBranch] = useState('')
  const [loading, setLoading] = useState(true)

  const [branchSearch, setBranchSearch] = useState('')
  const [creatingBranch, setCreatingBranch] = useState(false)
  const [aheadBranches, setAheadBranches] = useState<{ name: string; ahead: number; headUser?: string; headRepo?: string; baseOwner?: string; baseRepo?: string; baseBranch?: string }[]>([])
  const [forkStatus, setForkStatus] = useState<{ isFork: boolean; behind?: number; ahead?: number; parentOwner?: string; parentRepo?: string } | null>(null)
  const [syncing, setSyncing] = useState(false)
  const [cloneUrlTemplates, setCloneUrlTemplates] = useState<{ httpsUrlTemplate: string; sshUrlTemplate: string } | null>(null)
  const [sshPort, setSshPort] = useState(22)
  const [allFiles, setAllFiles] = useState<FileEntry[]>([])
  const [fileSearchLoading, setFileSearchLoading] = useState(false)
  const [fileSearch, setFileSearch] = useState('')
  const [showFileResults, setShowFileResults] = useState(false)
  const [cloneProtocolIndex, setCloneProtocolIndex] = useState(0)
  const [copiedUrl, setCopiedUrl] = useState<'https' | 'ssh' | null>(null)
  const fileSearchRef = useRef<HTMLInputElement>(null)

  // "/" 快捷键：聚焦"跳转到文件"搜索框
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === '/' && !['INPUT', 'TEXTAREA'].includes((e.target as HTMLElement)?.tagName)) {
        e.preventDefault()
        fileSearchRef.current?.focus()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [])

  // 获取 clone URL 模板和 SSH 端口
  useEffect(() => {
    api.getCloneUrlTemplates().then((templates) => {
      setCloneUrlTemplates(templates)
    }).catch(() => {})
    api.getSSHSettings().then((settings) => {
      setSshPort(settings?.port || 22)
    }).catch(() => {})
  }, [])

  useEffect(() => {
    const loadData = async () => {
      if (!repo || !branches) {
        return
      }

      if (branches.length === 0) {
        setLoading(false)
        return
      }

      const defaultBranch = branches.find((b) => b.isDefault)?.name || branches[0]?.name || ''
      setSelectedBranch(defaultBranch)

      try {
        const [commitsData, tagsData, filesData] = await Promise.all([
          api.listCommits(owner, repoName, defaultBranch, 1, 10).catch(() => []),
          api.listTags(owner, repoName).catch(() => []),
          api.listFiles(owner, repoName, defaultBranch).catch(() => []),
        ])

        setCommits(commitsData || [])
        setTags(tagsData || [])
        setFiles(filesData || [])

        const readmeFiles = (filesData || []).filter(f =>
          f.type === 'file' && /^readme(\.(md|markdown|txt))?$/i.test(f.name)
        )
        if (readmeFiles.length > 0) {
          try {
            const readmeFile = readmeFiles[0]
            const fileContent = await api.getFileContent(owner, repoName, readmeFile.path, defaultBranch)
            setReadmeContent(fileContent.content)
            setReadmeRaw(fileContent.content)
          } catch (error) {
            console.error('Failed to load README:', error)
          }
        }
      } catch (error) {
        console.error('Failed to load code page data:', error)
      } finally {
        setLoading(false)
      }
    }

    loadData()
  }, [owner, repoName, repo, branches, branches?.length])

  // Load all file paths for "Go to file" feature
  useEffect(() => {
    if (!selectedBranch) return
    setFileSearchLoading(true)
    api.listAllFiles(owner, repoName, selectedBranch)
      .then((data) => setAllFiles(data || []))
      .catch(() => {})
      .finally(() => setFileSearchLoading(false))
  }, [owner, repoName, selectedBranch])

  useEffect(() => {
    const checkAhead = async () => {
      if (!branches || !repo) {
        return
      }
      const defaultBranch = branches.find((b) => b.isDefault)?.name || branches[0]?.name || ''

      // Fetch all MRs (all states) to determine which branches should be skipped.
      // - OPEN MR → skip (someone is actively working on it)
      // - MERGED MR → skip only if the branch HEAD hasn't changed since the MR was created
      //   (i.e., no new commits after the merge). If the HEAD differs, show the prompt.
      // - CLOSED MR (not merged) → don't skip (user can retry with a new MR)
      const mrsResp = await api.listMergeRequests(owner, repoName).catch(() => ({ mergeRequests: [], participants: [] }))
      const mrs = mrsResp.mergeRequests || []
      // Map: requestBranch → most recent MR for that branch
      const mrByBranch = new Map<string, typeof mrs[number]>()
      for (const mr of mrs) {
        const existing = mrByBranch.get(mr.requestBranch)
        if (!existing || (mr.createdAt > existing.createdAt)) {
          mrByBranch.set(mr.requestBranch, mr)
        }
      }

      // Helper: decide whether a branch should be skipped based on its MR state.
      // currentHeadCommitId is the latest commit ID on the branch (optional).
      const shouldSkipBranch = (branchName: string, currentHeadCommitId?: string): boolean => {
        const mr = mrByBranch.get(branchName)
        if (!mr) return false
        if (mr.state === 'open') return true
        if (mr.state === 'merged') {
          // If we know the head commit at MR creation time and the current head,
          // only skip when they match (no new commits since the MR).
          if (mr.commitIdFrom && currentHeadCommitId) {
            return mr.commitIdFrom === currentHeadCommitId
          }
          // commitIdFrom missing (old MR created before tracking) —
          // don't skip, let the compare result decide whether to prompt.
          return false
        }
        // closed (not merged) → allow retry
        return false
      }

      const results: { name: string; ahead: number; headUser?: string; headRepo?: string; baseOwner?: string; baseRepo?: string; baseBranch?: string; lastCommitTime?: string }[] = []

      // Check branches in current repository
      if (branches.length > 1) {
        for (const branch of branches) {
          if (branch.name === defaultBranch) continue
          try {
            const result = await api.getAheadBehind(owner, repoName, branch.name, defaultBranch)
            if (result && result.ahead > 0) {
              let lastCommitTime: string | undefined
              let headCommitId: string | undefined
              try {
                const branchCommits = await api.listCommits(owner, repoName, branch.name, 1, 1)
                lastCommitTime = branchCommits?.[0]?.timestamp
                headCommitId = branchCommits?.[0]?.id
              } catch {}
              if (shouldSkipBranch(branch.name, headCommitId)) continue
              results.push({ name: branch.name, ahead: result.ahead, lastCommitTime })
            }
          } catch (err) {
            console.error(`checkAhead: error checking ${branch.name}:`, err)
          }
        }
      }

      // Case 1: current repository IS a fork → compare its branches against
      // the upstream (parent) default branch to detect ahead branches that
      // can be merged back upstream.
      if (forkStatus?.isFork && forkStatus.parentOwner && forkStatus.parentRepo) {
        const parentOwner = forkStatus.parentOwner
        const parentRepoName = forkStatus.parentRepo
        // Fetch parent's default branch
        let parentDefaultBranch = defaultBranch
        try {
          const parentRepoInfo = await api.getRepo(parentOwner, parentRepoName)
          parentDefaultBranch = parentRepoInfo?.defaultBranch || defaultBranch
        } catch {}
        // Fetch MRs from parent repo to skip branches that already have an open MR
        const parentMrsResp = await api.listMergeRequests(parentOwner, parentRepoName).catch(() => ({ mergeRequests: [], participants: [] }))
        const parentMrs = parentMrsResp.mergeRequests || []
        const parentMrByBranch = new Map<string, typeof parentMrs[number]>()
        for (const mr of parentMrs) {
          // Only consider MRs whose source matches this fork
          if (mr.requestUserName !== owner || mr.requestRepositoryName !== repoName) continue
          const existing = parentMrByBranch.get(mr.requestBranch)
          if (!existing || (mr.createdAt > existing.createdAt)) {
            parentMrByBranch.set(mr.requestBranch, mr)
          }
        }
        const shouldSkipBranchParent = (branchName: string, currentHeadCommitId?: string): boolean => {
          const mr = parentMrByBranch.get(branchName)
          if (!mr) return false
          if (mr.state === 'open') return true
          if (mr.state === 'merged') {
            if (mr.commitIdFrom && currentHeadCommitId) {
              return mr.commitIdFrom === currentHeadCommitId
            }
            return false
          }
          return false
        }

        for (const branch of branches) {
          // Skip branches that haven't been modified in the fork
          if (branch.name !== defaultBranch) {
            try {
              const forkAhead = await api.getAheadBehind(owner, repoName, branch.name, defaultBranch)
              if (!forkAhead || forkAhead.ahead === 0) {
                continue
              }
            } catch (err) {
              continue
            }
          }

          try {
            // Compare fork branch with parent's default branch
            const result = await api.compare(parentOwner, parentRepoName, parentDefaultBranch, branch.name, owner, repoName)
            if (result && result.commits && result.commits.length > 0) {
              // Skip if there are no actual file changes (content already merged)
              if (!result.fileChanges || result.fileChanges.length === 0) {
                continue
              }
              const currentHead = result.commits[0]?.id
              if (shouldSkipBranchParent(branch.name, currentHead)) continue
              results.push({ name: branch.name, ahead: result.commits.length, headUser: owner, headRepo: repoName, baseOwner: parentOwner, baseRepo: parentRepoName, baseBranch: parentDefaultBranch, lastCommitTime: result.commits[0]?.timestamp })
            }
          } catch (err) {
            console.error(`checkAhead: error comparing fork branch ${branch.name} against parent:`, err)
          }
        }
      }

      // Sort by latest commit time (descending) and only show the most recent one
      results.sort((a, b) => {
        const timeA = a.lastCommitTime ? new Date(a.lastCommitTime).getTime() : 0
        const timeB = b.lastCommitTime ? new Date(b.lastCommitTime).getTime() : 0
        return timeB - timeA
      })
      setAheadBranches(results.slice(0, 1))
    }
    checkAhead()
  }, [owner, repoName, branches, repo, forkStatus])

  useEffect(() => {
    if (!repo) return
    api.getForkStatus(owner, repoName).then((status) => {
      setForkStatus(status)
    }).catch(() => {})
  }, [owner, repoName, repo])

  const handleSyncFork = async () => {
    try {
      setSyncing(true)
      await api.syncFork(owner, repoName)
      toast({
        title: t('repo.syncForkSuccess'),
        status: 'success',
        duration: 2000,
      })
      // Refresh fork status
      const status = await api.getForkStatus(owner, repoName)
      setForkStatus(status)
      // Refresh page data
      window.location.reload()
    } catch (err: any) {
      toast({
        title: t('repo.syncForkFailed'),
        description: err.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSyncing(false)
    }
  }

  const handleBranchChange = (branchName: string) => {
    setSelectedBranch(branchName)
    window.location.href = `/${owner}/${repoName}/tree/${branchName}`
  }

  const handleCreateBranch = async (branchName: string) => {
    try {
      setCreatingBranch(true)
      const defaultBranch = branches.find((b) => b.isDefault)?.name || branches[0]?.name || ''
      await api.createBranch(owner, repoName, branchName, defaultBranch)
      toast({
        title: t('repo.branchCreated'),
        status: 'success',
        duration: 2000,
      })
      setBranchSearch('')
      await refreshData()
      handleBranchChange(branchName)
    } catch (error: any) {
      toast({
        title: t('repo.createBranchFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setCreatingBranch(false)
    }
  }

  const filteredBranches = branches.filter(b =>
    b.name.toLowerCase().includes(branchSearch.toLowerCase())
  )

  // "跳转到文件" 搜索：模糊匹配（大小写不敏感），最多显示 20 条
  const filteredFiles = fileSearch.trim()
    ? allFiles
        .filter(f => f.path.toLowerCase().includes(fileSearch.toLowerCase()))
        .slice(0, 20)
    : []

  if (loading) {
    return <Text>{t('common.loading')}</Text>
  }

  if (!repo) {
    return <Text>{t('repo.repoNotFound')}</Text>
  }

  const browserHost = typeof window !== 'undefined' ? window.location.hostname : 'localhost'
  const browserPort = typeof window !== 'undefined' ? (window.location.port || (window.location.protocol === 'https:' ? '443' : '80')) : '80'
  // 用模板生成 clone URL，如果模板为空则用默认值
  // 模板只配置前缀（协议+主机+端口），/{owner}/{repo}.git 由前端自动拼接
  // HTTPS: 端口 443/80 时省略，{port} 包含冒号
  const httpsPort = browserPort === '443' || browserPort === '80' ? '' : `:${browserPort}`
  const repoPath = `/${repo.userName}/${repo.repositoryName}.git`
  let cloneUrl = cloneUrlTemplates?.httpsUrlTemplate
    ? cloneUrlTemplates.httpsUrlTemplate
        .replace('{host}', browserHost)
        .replace('{port}', httpsPort) + repoPath
    : `${SERVER_BASE}${repoPath}`
  // 如果模板中没有 {port} 变量，自动检测并移除 443/80 端口
  if (cloneUrlTemplates?.httpsUrlTemplate && !cloneUrlTemplates.httpsUrlTemplate.includes('{port}')) {
    cloneUrl = cloneUrl.replace(/:(443|80)(\/|$)/, '$2')
  }
  // SSH: 根据模板解析出的端口决定格式
  // 模板中有 {port} 变量时替换为系统端口，否则直接使用模板中的端口
  // 端口 22: SCP-like 格式 git@host:owner/repo.git
  // 其他端口: ssh://git@host:port/owner/repo.git
  const sshTemplate = cloneUrlTemplates?.sshUrlTemplate || 'ssh://git@{host}:{port}'
  let sshUrl = sshTemplate.replace('{host}', browserHost)
  let extractedPort = sshPort
  if (sshTemplate.includes('{port}')) {
    sshUrl = sshUrl.replace('{port}', String(sshPort))
  } else {
    // 模板中没有 {port} 变量，从 URL 中解析端口
    const portMatch = sshUrl.match(/:(\d+)(?:\/|$)/)
    if (portMatch) {
      extractedPort = parseInt(portMatch[1])
    }
  }
  // 端口 22 时转换为 SCP-like 格式
  const sshCloneUrl = extractedPort === 22
    ? `${sshUrl.replace(/^ssh:\/\//, '').replace(/:22(?:\/|$)/, ':')}${repo.userName}/${repo.repositoryName}.git`
    : `${sshUrl}/${repo.userName}/${repo.repositoryName}.git`
  const isEmpty = !branches || branches.length === 0

  if (isEmpty) {
    return <EmptyRepoGuide owner={repo.userName} repoName={repo.repositoryName} cloneUrl={cloneUrl} sshCloneUrl={sshCloneUrl} sshEnabled={true} userRole={userRole} />
  }

  return (
    <VStack spacing={4} align="stretch">
      <HStack justify="space-between" flexWrap="wrap" gap={3}>
        <HStack spacing={3}>
          <Popover placement="bottom-start">
            <PopoverTrigger>
              <Button
                leftIcon={<Icon as={FiGitBranch} />}
                rightIcon={<Icon as={FiChevronRight} transform="rotate(90deg)" />}
                variant="outline"
                size="sm"
                bg="white"
                borderWidth="1px"
                borderColor="myGray.200"
                _hover={{ bg: 'myGray.50', borderColor: 'myGray.300' }}
              >
                {selectedBranch || repo.defaultBranch}
              </Button>
            </PopoverTrigger>
            <PopoverContent w="400px" maxH="600px" overflow="hidden" bg="white" position="relative">
              <PopoverArrow />
              <IconButton
                position="absolute"
                right={3}
                top={3}
                size="sm"
                onClick={() => document.body.click()}
                aria-label="Close"
                icon={<Icon as={FiChevronRight} transform="rotate(45deg)" w={4} h={4} />}
                bg="transparent"
                _hover={{ bg: 'myGray.100' }}
                zIndex={10}
              />
              <PopoverBody p={0}>
                <Box display="flex" flexDirection="column">
                  <Box p={3} borderBottom="1px" borderColor="myGray.200">
                    <InputGroup size="sm">
                      <InputLeftElement>
                        <Icon as={FiSearch} color="myGray.400" w={3} h={3} />
                      </InputLeftElement>
                      <Input
                        placeholder={t('repo.searchBranches')}
                        value={branchSearch}
                        onChange={(e) => setBranchSearch(e.target.value)}
                      />
                    </InputGroup>
                  </Box>

                  <Box display="flex" flexDirection="column" gap={0}>
                    {branchSearch && !branches.some(b => b.name === branchSearch) && canCreateBranch && (
                      <Box px={3} py={1} w="100%">
                        <Box
                          key="create-new"
                          display="flex"
                          alignItems="center"
                          gap={2}
                          px={3}
                          py={1}
                          borderRadius="md"
                          cursor={creatingBranch ? 'not-allowed' : 'pointer'}
                          _hover={creatingBranch ? {} : { bg: 'myGray.100' }}
                          onClick={() => !creatingBranch && handleCreateBranch(branchSearch)}
                          opacity={creatingBranch ? 0.6 : 1}
                        >
                          {creatingBranch ? (
                            <Spinner size="xs" />
                          ) : (
                            <Icon as={FiGitBranch} w={4} h={4} />
                          )}
                          <Text fontSize="sm">{t('repo.createBranch')}</Text>
                          <Code fontSize="xs" colorScheme="gray" px={1} ml={1}>
                            {branchSearch}
                          </Code>
                          <Text fontSize="sm" color="myGray.500" ml={1}>{t('repo.from')}</Text>
                          <Code fontSize="xs" colorScheme="gray" px={1} ml={1}>
                            {branches.find((b) => b.isDefault)?.name || branches[0]?.name || ''}
                          </Code>
                        </Box>
                      </Box>
                    )}

                    {filteredBranches.length === 0 && !branchSearch ? (
                      <Box p={4} textAlign="center">
                        <Text fontSize="sm" color="myGray.500">{t('repo.noBranchesFound')}</Text>
                      </Box>
                    ) : filteredBranches.length > 0 ? (
                      filteredBranches.map((branch) => (
                        <Box key={branch.name} px={3} py={1} w="100%">
                          <Box
                            display="flex"
                            alignItems="center"
                            gap={2}
                            px={3}
                            py={1}
                            borderRadius="md"
                            cursor="pointer"
                            _hover={{ bg: 'myGray.100' }}
                            onClick={() => handleBranchChange(branch.name)}
                            bg={branch.name === selectedBranch ? 'myGray.200' : 'transparent'}
                          >
                            <Icon as={FiGitBranch} w={4} h={4} />
                            <Text fontSize="sm" fontWeight={branch.name === selectedBranch ? 'medium' : 'normal'}>
                              {branch.name}
                            </Text>
                            {branch.isDefault && (
                              <Text fontSize="xs" color="myGray.500" ml="auto">{t('repo.default')}</Text>
                            )}
                          </Box>
                        </Box>
                      ))
                    ) : null}
                  </Box>

                  <Box p={3} borderTop="1px" borderColor="myGray.200">
                    <Link href={`/${owner}/${repoName}/branches`}>
                      <Button variant="ghost" size="sm" color="primary.600" w="100%">
                        {t('repo.viewAllBranches', { count: branches.length })}
                      </Button>
                    </Link>
                  </Box>
                </Box>
              </PopoverBody>
            </PopoverContent>
          </Popover>
          <Link href={`/${owner}/${repoName}/commits/${selectedBranch || repo.defaultBranch}`}>
            <Text fontSize="sm" color="myGray.500" _hover={{ color: 'primary.500', textDecoration: 'underline' }}>
              <Text as="span" fontWeight="medium" color="myGray.900">{commits.length}</Text> {t('repo.commits')}
            </Text>
          </Link>
          <Text fontSize="sm" color="myGray.300">•</Text>
          <Link href={`/${owner}/${repoName}/branches`}>
            <Text fontSize="sm" color="myGray.500" _hover={{ color: 'primary.500', textDecoration: 'underline' }}>
              <Text as="span" fontWeight="medium" color="myGray.900">{branches.length}</Text> {t('repo.branches')}
            </Text>
          </Link>
          <Text fontSize="sm" color="myGray.300">•</Text>
          <Link href={`/${owner}/${repoName}/tags`}>
            <Text fontSize="sm" color="myGray.500" _hover={{ color: 'primary.500', textDecoration: 'underline' }}>
              <Text as="span" fontWeight="medium" color="myGray.900">{tags.length}</Text> {t('repo.tags')}
            </Text>
          </Link>
          <Text fontSize="sm" color="myGray.300">•</Text>
          <Link href={`/${owner}/${repoName}/releases`}>
            <Text fontSize="sm" color="myGray.500" _hover={{ color: 'primary.500', textDecoration: 'underline' }}>
              <Icon as={FiPackage} w={3} h={3} display="inline" mr={1} verticalAlign="text-bottom" />
              {t('repo.releases')}
            </Text>
          </Link>
        </HStack>
        <HStack spacing={2} flexWrap="wrap">
          {/* 代码搜索按钮：跳转到仓库代码搜索页 */}
          <Link href={`/${owner}/${repoName}/search`}>
            <Button
              leftIcon={<Icon as={FiSearch} w={3} h={3} />}
              variant="outline"
              size="sm"
              bg="white"
              borderWidth="1px"
              borderColor="myGray.200"
              _hover={{ bg: 'myGray.50', borderColor: 'myGray.300' }}
            >
              {t('repo.codeSearch')}
            </Button>
          </Link>
          {/* "跳转到文件" 搜索框：与 Clone/Download 同行，避免独占一行被误认为换行 */}
          <Box position="relative">
            <InputGroup size="sm" w="220px">
              <InputLeftElement>
                <Icon as={FiSearch} color="myGray.400" w={3} h={3} />
              </InputLeftElement>
              <Input
                ref={fileSearchRef}
                placeholder={t('repo.goToFile')}
                value={fileSearch}
                onChange={(e) => {
                  setFileSearch(e.target.value)
                  setShowFileResults(true)
                }}
                onFocus={() => setShowFileResults(true)}
                onBlur={() => setTimeout(() => setShowFileResults(false), 150)}
                onKeyDown={(e) => {
                  if (e.key === 'Escape') {
                    setFileSearch('')
                    setShowFileResults(false)
                    ;(e.target as HTMLInputElement).blur()
                  }
                }}
              />
            </InputGroup>
            {showFileResults && fileSearch.trim().length > 0 && (
              <Box
                position="absolute"
                top="100%"
                left={0}
                mt={1}
                w="380px"
                maxH="400px"
                overflowY="auto"
                bg="white"
                border="1px solid"
                borderColor="myGray.200"
                borderRadius="md"
                boxShadow="md"
                zIndex={10}
              >
                {fileSearchLoading ? (
                  <HStack p={3} spacing={2}>
                    <Spinner size="sm" />
                    <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
                  </HStack>
                ) : filteredFiles.length === 0 ? (
                  <Text fontSize="sm" color="myGray.500" p={3}>{t('repo.noFilesFound')}</Text>
                ) : (
                  <VStack spacing={0} align="stretch">
                    {filteredFiles.map((file) => {
                      const fileName = file.path.split('/').pop() || file.path
                      const dirPath = file.path.includes('/')
                        ? file.path.substring(0, file.path.lastIndexOf('/'))
                        : ''
                      return (
                        <a
                          key={file.path}
                          href={`/${owner}/${repoName}/blob/${selectedBranch}/${file.path}`}
                          onMouseDown={(e) => e.preventDefault()}
                          onClick={(e) => {
                            e.preventDefault()
                            router.push(`/${owner}/${repoName}/blob/${selectedBranch}/${file.path}`)
                            setFileSearch('')
                            setShowFileResults(false)
                          }}
                        >
                          <HStack
                            spacing={2}
                            px={3}
                            py={2}
                            _hover={{ bg: 'myGray.50' }}
                            cursor="pointer"
                          >
                            <Icon as={FiFile} color="myGray.400" w={4} h={4} flexShrink={0} />
                            <VStack spacing={0} align="start" flex={1} minW={0}>
                              <Text fontSize="sm" color="myGray.900" fontWeight="medium" noOfLines={1}>
                                {fileName}
                              </Text>
                              {dirPath && (
                                <Text fontSize="xs" color="myGray.500" noOfLines={1}>
                                  {dirPath}
                                </Text>
                              )}
                            </VStack>
                          </HStack>
                        </a>
                      )
                    })}
                  </VStack>
                )}
              </Box>
            )}
          </Box>
          <Popover placement="bottom-end">
            <PopoverTrigger>
              <Button leftIcon={<Icon as={FiCopy} />} variant="whiteBase" size="sm">
                {t('repo.clone')}
              </Button>
            </PopoverTrigger>
            <PopoverContent w="400px">
              <PopoverArrow />
              <PopoverCloseButton />
              <PopoverBody p={4}>
                <VStack spacing={3} align="stretch">
                  <Text fontSize="sm" fontWeight="medium">{t('repo.cloneThisRepo')}</Text>
                  <Tabs size="sm" variant="soft-rounded" colorScheme="primary" index={cloneProtocolIndex} onChange={setCloneProtocolIndex}>
                    <TabList>
                      <Tab>HTTPS</Tab>
                      <Tab>SSH</Tab>
                    </TabList>
                    <TabPanels>
                      <TabPanel p={3}>
                        <HStack spacing={2}>
                          <InputGroup size="sm" flex={1}>
                            <Input value={cloneUrl} readOnly pr="32px" />
                            <InputRightElement w="32px">
                              <IconButton
                                aria-label="Copy"
                                icon={copiedUrl === 'https' ? <Icon as={FiCheck} /> : <Icon as={FiCopy} />}
                                size="xs"
                                variant="ghost"
                                color={copiedUrl === 'https' ? 'green.500' : undefined}
                                onClick={() => {
                                  navigator.clipboard.writeText(cloneUrl)
                                  setCopiedUrl('https')
                                  setTimeout(() => setCopiedUrl(null), 2000)
                                }}
                              />
                            </InputRightElement>
                          </InputGroup>
                          {copiedUrl === 'https' && (
                            <Text fontSize="xs" color="green.500" fontWeight="medium" whiteSpace="nowrap">
                              {t('repo.copied')}
                            </Text>
                          )}
                        </HStack>
                      </TabPanel>
                      <TabPanel p={3}>
                        <HStack spacing={2}>
                          <InputGroup size="sm" flex={1}>
                            <Input value={sshCloneUrl} readOnly pr="32px" />
                            <InputRightElement w="32px">
                              <IconButton
                                aria-label="Copy"
                                icon={copiedUrl === 'ssh' ? <Icon as={FiCheck} /> : <Icon as={FiCopy} />}
                                size="xs"
                                variant="ghost"
                                color={copiedUrl === 'ssh' ? 'green.500' : undefined}
                                onClick={() => {
                                  navigator.clipboard.writeText(sshCloneUrl)
                                  setCopiedUrl('ssh')
                                  setTimeout(() => setCopiedUrl(null), 2000)
                                }}
                              />
                            </InputRightElement>
                          </InputGroup>
                          {copiedUrl === 'ssh' && (
                            <Text fontSize="xs" color="green.500" fontWeight="medium" whiteSpace="nowrap">
                              {t('repo.copied')}
                            </Text>
                          )}
                        </HStack>
                      </TabPanel>
                    </TabPanels>
                  </Tabs>
                </VStack>
              </PopoverBody>
            </PopoverContent>
          </Popover>
          <Popover placement="bottom-end">
            <PopoverTrigger>
              <Button leftIcon={<Icon as={FiDownload} />} variant="whiteBase" size="sm">
                {t('repo.download')}
              </Button>
            </PopoverTrigger>
            <PopoverContent w="280px">
              <PopoverArrow />
              <PopoverCloseButton />
              <PopoverBody p={4}>
                <VStack spacing={3} align="stretch">
                  <Text fontSize="sm" fontWeight="medium" color="myGray.900">
                    {t('repo.downloadSource')}
                  </Text>
                  <VStack spacing={2} align="stretch">
                    <Button
                      as="a"
                      href={api.getArchiveUrl(owner, repoName, selectedBranch || repo.defaultBranch, 'zip')}
                      leftIcon={<Icon as={FiArchive} />}
                      size="sm"
                      variant="outline"
                      borderColor="myGray.200"
                      justifyContent="flex-start"
                    >
                      {t('repo.downloadZip')}
                    </Button>
                    <Button
                      as="a"
                      href={api.getArchiveUrl(owner, repoName, selectedBranch || repo.defaultBranch, 'tar.gz')}
                      leftIcon={<Icon as={FiArchive} />}
                      size="sm"
                      variant="outline"
                      borderColor="myGray.200"
                      justifyContent="flex-start"
                    >
                      {t('repo.downloadTarGz')}
                    </Button>
                  </VStack>
                  <Box borderTop="1px" borderColor="myGray.200" my={1} />
                  <Text fontSize="sm" fontWeight="medium" color="myGray.900">
                    {t('repo.atomFeed')}
                  </Text>
                  <Button
                    as="a"
                    href={api.getRepoAtomUrl(owner, repoName)}
                    target="_blank"
                    leftIcon={<Icon as={FiRss} />}
                    size="sm"
                    variant="outline"
                    borderColor="myGray.200"
                    justifyContent="flex-start"
                  >
                    {t('repo.repoAtomFeed')}
                  </Button>
                </VStack>
              </PopoverBody>
            </PopoverContent>
          </Popover>
        </HStack>
      </HStack>

      {canCreateIssue && aheadBranches.length > 0 && (
        <Box
          px={4}
          py={3}
          bg="yellow.50"
          borderWidth="1px"
          borderRadius="lg"
          borderColor="yellow.200"
        >
          {aheadBranches.map((branchInfo, idx) => {
            const headLabel = branchInfo.headUser
              ? `${branchInfo.headUser}/${branchInfo.headRepo}:${branchInfo.name}`
              : branchInfo.name
            const baseLabel = branchInfo.baseOwner
              ? `${branchInfo.baseOwner}/${branchInfo.baseRepo}:${branchInfo.baseBranch}`
              : (selectedBranch || repo.defaultBranch)
            const mrBaseOwner = branchInfo.baseOwner || owner
            const mrBaseRepo = branchInfo.baseRepo || repoName
            const mrBaseBranch = branchInfo.baseBranch || selectedBranch || repo.defaultBranch
            return (
            <HStack key={`${branchInfo.headUser || owner}/${branchInfo.headRepo || repoName}:${branchInfo.name}-${idx}`} justify="space-between" mb={2} _last={{ mb: 0 }}>
              <HStack spacing={2}>
                <Icon as={FiGitBranch} color="yellow.700" w={4} h={4} />
                <Text fontSize="sm" color="yellow.800">
                  <Text as="span" fontWeight="medium">
                    {headLabel}
                  </Text>
                  {' '}{t('repo.ahead')}{' '}
                  <Text as="span" fontWeight="medium">
                    {baseLabel}
                  </Text>
                  {' '}{t('repo.commitsCount', { count: branchInfo.ahead })}
                </Text>
              </HStack>
              <Link href={`/${mrBaseOwner}/${mrBaseRepo}/merge-requests/new?head=${branchInfo.name}&base=${mrBaseBranch}${branchInfo.headUser ? `&headUser=${branchInfo.headUser}&headRepo=${branchInfo.headRepo}` : ''}`}>
                <Button
                  leftIcon={<Icon as={FiGitPullRequest} />}
                  size="sm"
                  colorScheme="green"
                  fontWeight="medium"
                >
                  {t('repo.compareAndCreateMR')}
                </Button>
              </Link>
            </HStack>
            )
          })}
        </Box>
      )}

      {forkStatus?.isFork && forkStatus.behind !== undefined && forkStatus.behind > 0 && (
        <Box
          px={4}
          py={3}
          bg="blue.50"
          borderWidth="1px"
          borderRadius="lg"
          borderColor="blue.200"
        >
          <HStack justify="space-between">
            <HStack spacing={2}>
              <Icon as={FiGitPullRequest} color="blue.700" w={4} h={4} />
              <Text fontSize="sm" color="blue.800">
                {t('repo.forkBehind', {
                  count: forkStatus.behind,
                  parent: `${forkStatus.parentOwner}/${forkStatus.parentRepo}`,
                })}
              </Text>
            </HStack>
            <Button
              size="sm"
              colorScheme="blue"
              variant="outline"
              onClick={handleSyncFork}
              isLoading={syncing}
            >
              {t('repo.syncFork')}
            </Button>
          </HStack>
        </Box>
      )}

      <RepoFileTree
        owner={owner}
        repoName={repoName}
        selectedBranch={selectedBranch}
        defaultBranch={repo.defaultBranch}
        commits={commits}
        files={files}
      />

      <RepoReadme
        owner={owner}
        repoName={repoName}
        selectedBranch={selectedBranch || repo.defaultBranch}
        readmeContent={readmeContent}
        readmeRaw={readmeRaw}
        canEditFile={canEditFile}
      />
    </VStack>
  )
}

function EmptyRepoGuide({ owner, repoName, cloneUrl, sshCloneUrl, sshEnabled, userRole }: { owner: string; repoName: string; cloneUrl: string; sshCloneUrl: string; sshEnabled: boolean; userRole: string | null }) {
  const { hasCopied: hasCopiedNew, onCopy: onCopyNew } = useClipboard('')
  const { hasCopied: hasCopiedPush, onCopy: onCopyPush } = useClipboard('')
  const { hasCopied: hasCopiedClone, onCopy: onCopyClone } = useClipboard('')
  const [protocol, setProtocol] = useState<'https' | 'ssh'>('https')
  const currentUrl = protocol === 'ssh' ? sshCloneUrl : cloneUrl
  const { t } = useI18n()

  const canWrite = userRole === 'owner' || userRole === 'member'

  useEffect(() => {
    if (sshEnabled) {
      setProtocol('ssh')
    }
  }, [sshEnabled])

  const newRepoCommands = `echo "# ${repoName}" >> README.md
git init
git add README.md
git commit -m "first commit"
git branch -M main
git remote add origin ${currentUrl}
git push -u origin main`

  const pushRepoCommands = `git remote add origin ${currentUrl}
git branch -M main
git push -u origin main`

  return (
    <VStack spacing={6} align="stretch">
      {/* Web-based actions - "Create new file" / "Upload files" */}
      {canWrite && (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
          <VStack spacing={4} align="stretch">
            <HStack spacing={3}>
              <Icon as={FiFile} color="primary.500" w={5} h={5} />
              <Heading size="md">{t('repo.startWithWeb')}</Heading>
            </HStack>
            <Text color="myGray.600" fontSize="sm">
              {t('repo.startWithWebDesc')}
            </Text>
            <HStack spacing={3} flexWrap="wrap">
              <Link href={`/${owner}/${repoName}/new/main`}>
                <Button variant="primary" size="sm" leftIcon={<Icon as={FiFile} />}>
                  {t('repo.createFile')}
                </Button>
              </Link>
              <Link href={`/${owner}/${repoName}/upload/main`}>
                <Button variant="whiteBase" size="sm" leftIcon={<Icon as={FiUpload} />}>
                  {t('repo.uploadFiles')}
                </Button>
              </Link>
            </HStack>
          </VStack>
        </Box>
      )}

      {/* Quick setup section */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
        <VStack spacing={4} align="stretch">
          <HStack spacing={3}>
            <Icon as={FiGitBranch} color="primary.500" w={5} h={5} />
            <Heading size="md">{t('repo.quickSetup')}</Heading>
          </HStack>
          <Text color="myGray.600" fontSize="sm">
            {t('repo.quickSetupDesc')}
          </Text>

          {/* Protocol selector and clone URL */}
          <HStack spacing={2}>
            <Text fontSize="sm" color="myGray.600">{t('repo.protocolLabel')}</Text>
            <Button
              size="xs"
              variant={protocol === 'https' ? 'primary' : 'whiteBase'}
              onClick={() => setProtocol('https')}
            >
              HTTPS
            </Button>
            {sshEnabled && (
              <Button
                size="xs"
                variant={protocol === 'ssh' ? 'primary' : 'whiteBase'}
                onClick={() => setProtocol('ssh')}
              >
                SSH
              </Button>
            )}
            <InputGroup size="sm" flex={1}>
              <Input value={currentUrl} readOnly pr="32px" fontSize="sm" fontFamily="mono" />
              <InputRightElement w="32px">
                <IconButton
                  aria-label="Copy"
                  icon={<Icon as={hasCopiedClone ? FiCheck : FiCopy} />}
                  size="xs"
                  variant="ghost"
                  onClick={() => {
                    onCopyClone()
                    navigator.clipboard.writeText(currentUrl)
                  }}
                />
              </InputRightElement>
            </InputGroup>
          </HStack>

          {/* Command line instructions */}
          <Tabs variant="line" colorScheme="primary">
            <TabList>
              <Tab>{t('repo.createNewRepoTab')}</Tab>
              <Tab>{t('repo.pushExistingRepoTab')}</Tab>
              <Tab>{t('repo.importCodeTab')}</Tab>
            </TabList>

            <TabPanels>
              <TabPanel>
                <VStack spacing={4} align="stretch">
                  <Text fontSize="sm" color="myGray.600">
                    {t('repo.createNewRepoInstructions')}
                  </Text>
                  <Box position="relative">
                    <Code
                      display="block"
                      whiteSpace="pre"
                      p={4}
                      borderRadius="md"
                      bg="myGray.50"
                      fontSize="sm"
                      overflowX="auto"
                    >
                      {newRepoCommands}
                    </Code>
                    <Button
                      position="absolute"
                      top={2}
                      right={2}
                      size="xs"
                      leftIcon={<Icon as={hasCopiedNew ? FiCheck : FiCopy} />}
                      onClick={() => {
                        onCopyNew()
                        navigator.clipboard.writeText(newRepoCommands)
                      }}
                    >
                      {hasCopiedNew ? t('repo.copied') : t('repo.copyUrl')}
                    </Button>
                  </Box>
                </VStack>
              </TabPanel>

              <TabPanel>
                <VStack spacing={4} align="stretch">
                  <Text fontSize="sm" color="myGray.600">
                    {t('repo.pushExistingRepoInstructions')}
                  </Text>
                  <Box position="relative">
                    <Code
                      display="block"
                      whiteSpace="pre"
                      p={4}
                      borderRadius="md"
                      bg="myGray.50"
                      fontSize="sm"
                      overflowX="auto"
                    >
                      {pushRepoCommands}
                    </Code>
                    <Button
                      position="absolute"
                      top={2}
                      right={2}
                      size="xs"
                      leftIcon={<Icon as={hasCopiedPush ? FiCheck : FiCopy} />}
                      onClick={() => {
                        onCopyPush()
                        navigator.clipboard.writeText(pushRepoCommands)
                      }}
                    >
                      {hasCopiedPush ? t('repo.copied') : t('repo.copyUrl')}
                    </Button>
                  </Box>
                </VStack>
              </TabPanel>

              <TabPanel>
                <VStack spacing={4} align="stretch">
                  <Text fontSize="sm" color="myGray.600">
                    {t('repo.importCodeInstructions')}
                  </Text>
                  <Button variant="primary" size="sm">
                    {t('repo.importRepo')}
                  </Button>
                </VStack>
              </TabPanel>
            </TabPanels>
          </Tabs>
        </VStack>
      </Box>
    </VStack>
  )
}
