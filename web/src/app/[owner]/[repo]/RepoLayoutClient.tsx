'use client'

import {
  Box,
  Container,
  Text,
  Button,
  VStack,
  HStack,
  Badge,
  Icon,
  Link,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
} from '@chakra-ui/react'
import NextLink from 'next/link'
import { useEffect, useState, useCallback } from 'react'
import { useParams, usePathname } from 'next/navigation'
import { api } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { RepoContext } from './RepoContext'
import type { Repository, Branch, Issue, MergeRequest } from '@/lib/types'
import {
  FiBook,
  FiStar,
  FiGitBranch,
  FiEye,
  FiMessageSquare,
  FiGitPullRequest,
  FiSettings,
  FiCode,
  FiUsers,
  FiActivity,
  FiBarChart2,
  FiChevronDown,
  FiTrendingUp,
  FiGitCommit,
} from 'react-icons/fi'

export default function RepoLayout({ children }: { children: React.ReactNode }) {
  const params = useParams()
  const pathname = usePathname()
  const toast = useGithubToast()
  const { t } = useI18n()
  const owner = (params.owner as string) || ''
  const repoName = (params.repo as string) || ''

  const [repo, setRepo] = useState<Repository | null>(null)
  const [branches, setBranches] = useState<Branch[]>([])
  const [issues, setIssues] = useState<Issue[]>([])
  const [mergeRequests, setMergeRequests] = useState<MergeRequest[]>([])
  const [loading, setLoading] = useState(true)
  const [starred, setStarred] = useState(false)
  const [watching, setWatching] = useState(false)
  const [starCount, setStarCount] = useState(0)
  const [forkCount, setForkCount] = useState(0)
  const [userRole, setUserRole] = useState<'owner' | 'member' | 'viewer' | '' | null>(null)
  const [canCreateIssue, setCanCreateIssue] = useState(false)
  const [forking, setForking] = useState(false)

  const loadData = useCallback(async () => {
    try {
      const [repoData, branchesData, issuesResp, mrsResp, starredData, watchingData, stargazersData, forkCountData, roleData] = await Promise.all([
        api.getRepo(owner, repoName).catch(() => null),
        api.listBranches(owner, repoName).catch(() => []),
        api.listIssues(owner, repoName).catch(() => ({ issues: [], participants: [] })),
        api.listMergeRequests(owner, repoName).catch(() => ({ mergeRequests: [], participants: [] })),
        api.isStarred(owner, repoName).catch(() => ({ starred: false })),
        api.isWatching(owner, repoName).catch(() => ({ watching: false })),
        api.getStargazers(owner, repoName).catch(() => ({ stargazers: [] })),
        api.getForkCount(owner, repoName).catch(() => ({ count: 0 })),
        api.getUserRole(owner, repoName).catch(() => ({ role: null, canCreateIssue: false })),
      ])

      const issuesData = issuesResp?.issues || []
      const mergeRequestsData = mrsResp?.mergeRequests || []

      setRepo(repoData)
      setBranches(branchesData || [])
      setIssues(issuesData)
      setMergeRequests(mergeRequestsData)
      setStarred(starredData?.starred || false)
      setWatching(watchingData?.watching || false)
      setStarCount(stargazersData?.stargazers?.length || 0)
      setForkCount(forkCountData?.count || 0)
      setUserRole(roleData?.role || null)
      setCanCreateIssue(roleData?.canCreateIssue || false)
    } catch (error) {
      console.error('Failed to load repository:', error)
    } finally {
      setLoading(false)
    }
  }, [owner, repoName])

  useEffect(() => {
    loadData()
  }, [owner, repoName, loadData])

  const handleStar = async () => {
    try {
      if (starred) {
        await api.unstarRepo(owner, repoName)
        setStarred(false)
        setStarCount(Math.max(0, starCount - 1))
      } else {
        await api.starRepo(owner, repoName)
        setStarred(true)
        setStarCount(starCount + 1)
      }
    } catch (error) {
      console.error('Failed to toggle star:', error)
    }
  }

  const handleWatch = async () => {
    try {
      if (watching) {
        await api.unwatchRepo(owner, repoName)
        setWatching(false)
      } else {
        await api.watchRepo(owner, repoName, true)
        setWatching(true)
      }
    } catch (error) {
      console.error('Failed to toggle watch:', error)
    }
  }

  const handleFork = async () => {
    setForking(true)
    try {
      const forkedRepo = await api.forkRepo(owner, repoName)
      toast({
        title: t('repo.forkSuccess'),
        description: t('repo.forkSuccessDesc', { path: `${forkedRepo.userName}/${forkedRepo.repositoryName}` }),
        status: 'success',
        duration: 3000,
      })
      // Force full page reload to ensure fresh data
      window.location.href = `/${forkedRepo.userName}/${forkedRepo.repositoryName}`
    } catch (error: any) {
      toast({
        title: t('repo.forkFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setForking(false)
    }
  }

  if (loading) {
    return (
      <Box minH="100vh" bg="myGray.50">
        <Container maxW="container.xl" py={8}>
          <Text>{t('common.loading')}</Text>
        </Container>
      </Box>
    )
  }

  if (!repo) {
    return (
      <Box minH="100vh" bg="myGray.50">
        <Container maxW="container.xl" py={8}>
          <Text>{t('repo.repoNotFound')}</Text>
        </Container>
      </Box>
    )
  }

  // 判断当前激活的 tab
  // milestones 归入 issues，releases/search 归入 code，activity 归入 insights
  const getActiveTab = () => {
    if (pathname === `/${owner}/${repoName}`) return 'code'
    if (pathname.startsWith(`/${owner}/${repoName}/issues`)) return 'issues'
    if (pathname.startsWith(`/${owner}/${repoName}/milestones`)) return 'issues'
    if (pathname.startsWith(`/${owner}/${repoName}/merge-requests`)) return 'merge-requests'
    if (pathname.startsWith(`/${owner}/${repoName}/wiki`)) return 'wiki'
    if (pathname.startsWith(`/${owner}/${repoName}/releases`)) return 'code'
    if (pathname.startsWith(`/${owner}/${repoName}/pipelines`)) return 'pipelines'
    if (pathname.startsWith(`/${owner}/${repoName}/jobs`)) return 'pipelines'
    if (pathname.startsWith(`/${owner}/${repoName}/search`)) return 'code'
    if (pathname.startsWith(`/${owner}/${repoName}/activity`)) return 'insights'
    if (pathname.startsWith(`/${owner}/${repoName}/contributors`)) return 'insights'
    if (pathname.startsWith(`/${owner}/${repoName}/code-quality`)) return 'insights'
    if (pathname.startsWith(`/${owner}/${repoName}/settings`)) return 'settings'
    return 'code'
  }

  const activeTab = getActiveTab()

  const navItems = [
    { key: 'code', label: t('repo.code'), href: `/${owner}/${repoName}`, icon: FiCode },
    { key: 'issues', label: t('repo.issues'), href: `/${owner}/${repoName}/issues`, icon: FiMessageSquare, count: issues.length },
    { key: 'merge-requests', label: t('repo.mergeRequests'), href: `/${owner}/${repoName}/merge-requests`, icon: FiGitPullRequest, count: mergeRequests.filter(mr => mr.state === 'open').length },
    { key: 'pipelines', label: t('repo.pipelines'), href: `/${owner}/${repoName}/pipelines`, icon: FiGitCommit },
  ]

  const insightItems = [
    { key: 'activity', label: t('repo.activity'), href: `/${owner}/${repoName}/activity`, icon: FiTrendingUp },
    { key: 'contributors', label: t('repo.contributors'), href: `/${owner}/${repoName}/contributors`, icon: FiUsers },
    { key: 'code-quality', label: t('repo.codeQuality'), href: `/${owner}/${repoName}/code-quality`, icon: FiActivity },
  ]

  return (
    <RepoContext.Provider
      value={{
        repo,
        branches,
        issues,
        mergeRequests,
        starred,
        watching,
        starCount,
        forkCount,
        isArchived: repo.isArchived || false,
        userRole,
        canCreateIssue,
        refreshData: loadData,
        handleStar,
        handleWatch,
        handleFork,
      }}
    >
      <Box minH="100vh" bg="myGray.50">
        <Container maxW="container.xl" py={6}>
          <VStack spacing={6} align="stretch">
            {/* 归档提示条 */}
            {repo.isArchived && (
              <Box bg="yellow.50" borderWidth="1px" borderColor="yellow.200" borderRadius="md" px={4} py={3}>
                <HStack spacing={2}>
                  <Badge colorScheme="yellow">{t('repo.archivedBadge')}</Badge>
                  <Text fontSize="sm" color="yellow.700">
                    {t('repo.archivedBanner')}
                  </Text>
                </HStack>
              </Box>
            )}

            {/* 仓库头部 */}
            <HStack justify="space-between" align="center" flexWrap="wrap" gap={4}>
              <HStack spacing={3}>
                <Icon as={FiBook} w={5} h={5} color="myGray.500" />
                <HStack spacing={0}>
                  <Link as={NextLink} href={`/${owner}`} color="blue.600" _hover={{ textDecoration: 'underline' }}>
                    {repo.userName}
                  </Link>
                  <Text color="myGray.900" fontWeight="medium">/</Text>
                  <Link as={NextLink} href={`/${owner}/${repoName}`} color="blue.600" _hover={{ textDecoration: 'underline' }}>
                    {repo.repositoryName}
                  </Link>
                </HStack>
                <Badge colorScheme={repo.isPrivate ? 'yellow' : 'green'} variant="subtle">
                  {repo.isPrivate ? t('repo.private') : t('repo.public')}
                </Badge>
                {repo.isArchived && (
                  <Badge colorScheme="red" variant="subtle">
                    {t('repo.archivedBadge')}
                  </Badge>
                )}
                {repo.isTemplate && (
                  <Badge colorScheme="purple" variant="subtle">
                    {t('repo.templateBadge')}
                  </Badge>
                )}
                {repo.originUserName && repo.originRepositoryName && (
                  <Text fontSize="sm" color="myGray.500">
                    forked from{' '}
                    <Link as={NextLink} href={`/${repo.originUserName}/${repo.originRepositoryName}`} color="blue.600" _hover={{ textDecoration: 'underline' }}>
                      {repo.originUserName}/{repo.originRepositoryName}
                    </Link>
                  </Text>
                )}
              </HStack>
              <HStack spacing={2}>
                <Button leftIcon={<Icon as={FiStar} />} variant="whiteBase" size="sm" onClick={handleStar}>
                  {starred ? t('repo.unstar') : t('repo.star')} {starCount > 0 && starCount}
                </Button>
                <Button leftIcon={<Icon as={FiEye} />} variant="whiteBase" size="sm" onClick={handleWatch}>
                  {watching ? t('repo.unwatch') : t('repo.watch')}
                </Button>
                <Button leftIcon={<Icon as={FiGitBranch} />} variant="whiteBase" size="sm" onClick={handleFork} isLoading={forking} loadingText={t('repo.forking')}>
                  {t('repo.fork')} {forkCount > 0 && forkCount}
                </Button>
              </HStack>
            </HStack>

            {/* 仓库描述 */}
            {repo.description && (
              <Text color="myGray.600" fontSize="sm">
                {repo.description}
              </Text>
            )}

            {/* 导航 Tabs - 空仓库时不显示 */}
            {branches.length > 0 && (
              <HStack spacing={1} borderBottom="2px solid" borderColor="myGray.200">
                {navItems.map((item) => {
                  const isActive = activeTab === item.key
                  return (
                    <Link key={item.key} href={item.href}>
                      <Button
                        variant="ghost"
                        size="sm"
                        leftIcon={<Icon as={item.icon} />}
                        fontWeight={isActive ? 'semibold' : 'medium'}
                        color={isActive ? 'primary.600' : 'myGray.600'}
                        borderBottom={isActive ? '2px solid' : 'none'}
                        borderColor="primary.600"
                        mb="-2px"
                        _hover={{
                          bg: isActive ? 'transparent' : 'myGray.100',
                          color: isActive ? 'primary.600' : 'myGray.900',
                        }}
                      >
                        {item.label}
                        {item.count !== undefined && item.count > 0 && (
                          <Badge ml={2} colorScheme="gray" fontSize="xs">
                            {item.count}
                          </Badge>
                        )}
                      </Button>
                    </Link>
                  )
                })}
                {/* Insights 下拉菜单 */}
                <Menu>
                  <MenuButton
                    as={Button}
                    variant="ghost"
                    size="sm"
                    rightIcon={<Icon as={FiChevronDown} w={3} h={3} />}
                    leftIcon={<Icon as={FiBarChart2} />}
                    fontWeight={activeTab === 'insights' ? 'semibold' : 'medium'}
                    color={activeTab === 'insights' ? 'primary.600' : 'myGray.600'}
                    borderBottom={activeTab === 'insights' ? '2px solid' : 'none'}
                    borderColor="primary.600"
                    mb="-2px"
                    _hover={{
                      bg: activeTab === 'insights' ? 'transparent' : 'myGray.100',
                      color: activeTab === 'insights' ? 'primary.600' : 'myGray.900',
                    }}
                  >
                    {t('repo.insights')}
                  </MenuButton>
                  <MenuList>
                    {insightItems.map((item) => (
                      <MenuItem key={item.key} as={NextLink} href={item.href} icon={<Icon as={item.icon} />}>
                        {item.label}
                      </MenuItem>
                    ))}
                  </MenuList>
                </Menu>
                {/* Settings tab (owner only) */}
                {userRole === 'owner' && (
                  <Link href={`/${owner}/${repoName}/settings`}>
                    <Button
                      variant="ghost"
                      size="sm"
                      leftIcon={<Icon as={FiSettings} />}
                      fontWeight={activeTab === 'settings' ? 'semibold' : 'medium'}
                      color={activeTab === 'settings' ? 'primary.600' : 'myGray.600'}
                      borderBottom={activeTab === 'settings' ? '2px solid' : 'none'}
                      borderColor="primary.600"
                      mb="-2px"
                      _hover={{
                        bg: activeTab === 'settings' ? 'transparent' : 'myGray.100',
                        color: activeTab === 'settings' ? 'primary.600' : 'myGray.900',
                      }}
                    >
                      {t('repo.settings')}
                    </Button>
                  </Link>
                )}
              </HStack>
            )}

            {/* 子页面内容 */}
            {children}
          </VStack>
        </Container>
      </Box>
    </RepoContext.Provider>
  )
}
