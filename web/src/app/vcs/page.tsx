'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  Button,
  VStack,
  HStack,
  Badge,
  Icon,
  Flex,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'
import { api, Repository, Activity, Organization } from '@/lib/api'
import {
  FiBook,
  FiGitCommit,
  FiGitPullRequest,
  FiCircle,
  FiCheckCircle,
  FiTag,
  FiTrendingUp,
  FiStar,
  FiGitBranch,
  FiMessageSquare,
} from 'react-icons/fi'
import { AiOutlineBank } from 'react-icons/ai'
import { ContributionGraph } from '@/components/ContributionGraph'
import { formatRelativeTime } from '@/lib/time'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'

function getActivityIcon(type: string) {
  switch (type) {
    case 'commit':
    case 'push':
      return { icon: FiGitCommit, color: 'blue.500' }
    case 'issue_open':
    case 'open_issue':
      return { icon: FiCircle, color: 'green.500' }
    case 'issue_close':
      return { icon: FiCheckCircle, color: 'purple.500' }
    case 'issue_comment':
      return { icon: FiMessageSquare, color: 'blue.500' }
    case 'open_merge_request':
      return { icon: FiGitPullRequest, color: 'green.500' }
    case 'merge_request_close':
      return { icon: FiGitPullRequest, color: 'purple.500' }
    case 'merge_request_merge':
      return { icon: FiCheckCircle, color: 'purple.500' }
    case 'merge_request_comment':
      return { icon: FiMessageSquare, color: 'blue.500' }
    case 'release':
      return { icon: FiTag, color: 'yellow.600' }
    case 'star':
      return { icon: FiStar, color: 'yellow.500' }
    case 'fork_repository':
      return { icon: FiBook, color: 'primary.500' }
    case 'create_repository':
      return { icon: FiBook, color: 'green.500' }
    case 'delete_repository':
      return { icon: FiBook, color: 'red.500' }
    case 'fork':
      return { icon: FiGitBranch, color: 'primary.500' }
    default:
      return { icon: FiTrendingUp, color: 'myGray.500' }
  }
}

function renderActivityMessage(activity: Activity, t: (key: string, params?: Record<string, string | number>) => string) {
  const { message, activityType, activityUserName } = activity

  const keyMap: Record<string, string> = {
    push: 'home.activityPush',
    create_repository: 'home.activityCreateRepo',
    delete_repository: 'home.activityDeleteRepo',
    fork_repository: 'home.activityForkRepo',
    open_issue: 'home.activityOpenIssue',
    issue_close: 'home.activityCloseIssue',
    issue_reopen: 'home.activityReopenIssue',
    issue_comment: 'home.activityIssueComment',
    open_merge_request: 'home.activityOpenMR',
    merge_request_close: 'home.activityCloseMR',
    merge_request_reopen: 'home.activityReopenMR',
    merge_request_merge: 'home.activityMergeMR',
    merge_request_comment: 'home.activityMRComment',
  }

  const key = keyMap[activityType]
  if (key) {
    const translated = t(key, { user: activityUserName, repo: message, title: message })
    return <Text as="span" color="myGray.700">{translated}</Text>
  }

  return <Text as="span" color="myGray.700">{message}</Text>
}

interface IssueActivityInfo {
  issueNumber: number
  issueTitle: string
}

function IssueActivityDetail({ activity }: { activity: Activity }) {
  const { t } = useI18n()
  const info: IssueActivityInfo | null = activity.additionalInfo ? JSON.parse(activity.additionalInfo) : null

  const actionMap: Record<string, string> = {
    open_issue: 'home.activityOpenedIssue',
    issue_close: 'home.activityClosedIssue',
    issue_reopen: 'home.activityReopenedIssue',
    issue_comment: 'home.activityCommentedIssue',
    open_merge_request: 'home.activityOpenedMR',
    merge_request_close: 'home.activityClosedMR',
    merge_request_reopen: 'home.activityReopenedMR',
    merge_request_merge: 'home.activityMergedMR',
    merge_request_comment: 'home.activityCommentedMR',
  }

  const actionKey = actionMap[activity.activityType]
  if (!actionKey || !info) {
    return <Text as="span" color="myGray.700">{activity.message}</Text>
  }

  const isMR = activity.activityType === 'open_merge_request' || activity.activityType.startsWith('merge_request_')
  const detailPath = isMR ? 'merge-requests' : 'issues'
  const issueRef = `${activity.userName}/${activity.repositoryName}#${info.issueNumber}`
  const issueUrl = `/${activity.userName}/${activity.repositoryName}/${detailPath}/${info.issueNumber}`

  const verbMap: Record<string, string> = {
    open_issue: 'home.verbOpened',
    issue_close: 'home.verbClosed',
    issue_reopen: 'home.verbReopened',
    issue_comment: 'home.verbCommentedOn',
    open_merge_request: 'home.verbOpenedMR',
    merge_request_close: 'home.verbClosedMR',
    merge_request_reopen: 'home.verbReopenedMR',
    merge_request_merge: 'home.verbMergedMR',
    merge_request_comment: 'home.verbCommentedOnMR',
  }

  const verbKey = verbMap[activity.activityType]

  return (
    <VStack align="start" spacing={1} w="100%">
      <Text fontSize="sm" color="myGray.700" lineHeight="1.5">
        {activity.activityUserName}
        {' '}
        {verbKey && t(verbKey)}
        {' '}
        <Link href={issueUrl}>
          <Text as="span" color="primary.600" _hover={{ textDecoration: 'underline' }}>
            {issueRef}
          </Text>
        </Link>
      </Text>
      <Text fontSize="sm" color="myGray.900" fontWeight="medium">
        {info.issueTitle}
      </Text>
    </VStack>
  )
}

interface PushCommitInfo {
  id: string
  message: string
  author: string
  email: string
  time: string
}

interface PushRefUpdate {
  refName: string
  branchName: string
  oldSha: string
  newSha: string
  commits: PushCommitInfo[]
  isNewBranch: boolean
  isDeleted: boolean
}

interface PushDetail {
  refUpdates: PushRefUpdate[]
}

function PushActivityDetail({ activity }: { activity: Activity }) {
  const { t } = useI18n()
  const detail: PushDetail | null = activity.additionalInfo ? JSON.parse(activity.additionalInfo) : null

  if (!detail?.refUpdates?.length) {
    return <Text as="span" color="myGray.700">{t('home.activityPush', { user: activity.activityUserName, repo: activity.message })}</Text>
  }

  const totalCommits = detail.refUpdates.reduce((sum, ref) => sum + (ref.commits?.length || 0), 0)
  const showMore = totalCommits > 10

  return (
    <>
      <Text as="span" color="myGray.700">
        {t('home.activityPush', { user: activity.activityUserName, repo: activity.message })}
      </Text>
      {detail.refUpdates.map((ref) => {
        const commits = ref.commits || []
        const visibleCommits = showMore ? commits.slice(0, 10) : commits
        const hiddenCount = commits.length - visibleCommits.length

        return (
          <Box key={ref.refName} mt={2} ml={1} pl={3} borderLeft="2px solid" borderColor="myGray.200" w="100%">
            <Text fontSize="xs" color="myGray.500" fontWeight="medium" mb={1}>
              {ref.isNewBranch
                ? t('home.pushNewBranch', { branch: ref.branchName })
                : ref.isDeleted
                ? t('home.pushDeleteBranch', { branch: ref.branchName })
                : t('home.pushToBranch', { branch: ref.branchName })}
            </Text>
            {visibleCommits.map((commit) => (
              <HStack key={commit.id} spacing={2} py={0.5}>
                <Icon as={FiGitCommit} w={3} h={3} color="myGray.400" flexShrink={0} />
                <Link href={`/${activity.userName}/${activity.repositoryName}/commit/${commit.id}?branch=${ref.branchName}`}>
                  <Text as="span" fontSize="xs" color="primary.600" fontFamily="mono" _hover={{ textDecoration: 'underline' }}>
                    {commit.id.substring(0, 7)}
                  </Text>
                </Link>
                <Text fontSize="xs" color="myGray.700" noOfLines={1} flex={1}>
                  {commit.message.split('\n')[0]}
                </Text>
                <Text fontSize="xs" color="myGray.400" flexShrink={0}>
                  {commit.author}
                </Text>
              </HStack>
            ))}
            {hiddenCount > 0 && (
              <Link href={`/${activity.userName}/${activity.repositoryName}/commits?branch=${ref.branchName}`}>
                <Text fontSize="xs" color="primary.600" py={0.5} _hover={{ textDecoration: 'underline' }}>
                  {t('home.pushMoreCommits', { count: hiddenCount })}
                </Text>
              </Link>
            )}
          </Box>
        )
      })}
    </>
  )
}

export default function VcsPage() {
  const router = useRouter()
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()
  const [repos, setRepos] = useState<Repository[]>([])
  const [activities, setActivities] = useState<Activity[]>([])
  const [organizations, setOrganizations] = useState<Organization[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const token = localStorage.getItem('token')
    if (!token) {
      setLoading(false)
      return
    }
    Promise.all([
      api.getMyRepos().catch(() => []),
      api.listActivities(20, 0).catch(() => ({ activities: [], participants: [] })),
      api.listMyOrganizations().catch(() => []),
    ]).then(([reposData, activitiesData, organizationsData]) => {
      setRepos(reposData || [])
      setActivities(activitiesData.activities || [])
      setOrganizations(organizationsData || [])
    }).finally(() => setLoading(false))
  }, [])

  // Redirect unauthenticated users to login
  useEffect(() => {
    if (authLoading) return
    if (!user) {
      router.replace('/login')
      return
    }
  }, [authLoading, user, router])

  if (loading || !user) {
    return (
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={8}>
          <Text color="myGray.600">{t('common.loading')}</Text>
        </Container>
      </Box>
    )
  }

  return (
    <Box bg="myGray.50">
      <Container maxW="container.xl" py={6}>
        <Flex gap={6} align="flex-start">
          {/* 左侧主内容 */}
          <Box flex={1} minW={0}>
            {/* 贡献热力图 */}
            <Box bg="white" borderWidth="1px" borderColor="myGray.200" borderRadius="lg" overflow="hidden" mb={6} p={6}>
              <ContributionGraph username={user.userName} />
            </Box>

            {/* 动态列表 */}
            <Box borderWidth="1px" borderColor="myGray.200" borderRadius="lg" overflow="hidden" bg="white">
              <Box px={5} py={3} borderBottom="1px" borderColor="myGray.200">
                <HStack spacing={3}>
                  <Icon as={FiTrendingUp} w={5} h={5} color="myGray.600" />
                  <Heading size="sm" color="myGray.900">{t('home.activity')}</Heading>
                </HStack>
              </Box>

              {activities.length === 0 ? (
                <Box py={16}>
                  <VStack spacing={4}>
                    <Box p={4} bg="myGray.50" borderRadius="full">
                      <Icon as={FiTrendingUp} w={8} h={8} color="myGray.400" />
                    </Box>
                    <VStack spacing={2}>
                      <Text color="myGray.700" fontSize="sm" fontWeight="medium">
                        {t('home.noActivity')}
                      </Text>
                      <Text color="myGray.500" fontSize="xs" textAlign="center" maxW="300px">
                        {t('home.noActivityDesc')}
                      </Text>
                    </VStack>
                  </VStack>
                </Box>
              ) : (
                <VStack spacing={0} align="stretch">
                  {activities.map((activity) => {
                    const { icon, color } = getActivityIcon(activity.activityType)

                    return (
                      <HStack
                        key={activity.activityId}
                        align="start"
                        spacing={4}
                        py={4}
                        px={5}
                        borderBottom="1px solid"
                        borderBottomColor="myGray.100"
                        _hover={{ bg: 'myGray.50' }}
                        transition="all 0.15s"
                        _last={{ borderBottom: 'none' }}
                      >
                        <Box p={2} bg={`${color.split('.')[0]}.50`} borderRadius="full" flexShrink={0}>
                          <Icon as={icon} w={4} h={4} color={color} />
                        </Box>
                        <VStack align="start" spacing={1.5} flex={1}>
                          {activity.activityType === 'push' && activity.additionalInfo ? (
                            <PushActivityDetail activity={activity} />
                          ) : (['open_issue', 'issue_close', 'issue_reopen', 'issue_comment', 'open_merge_request', 'merge_request_close', 'merge_request_reopen', 'merge_request_merge', 'merge_request_comment'].includes(activity.activityType) && activity.additionalInfo) ? (
                            <IssueActivityDetail activity={activity} />
                          ) : (
                            <Text fontSize="sm" flexWrap="wrap" lineHeight="1.5">
                              {renderActivityMessage(activity, t)}
                            </Text>
                          )}
                          <HStack spacing={2} fontSize="xs">
                            <Link href={`/${activity.userName}/${activity.repositoryName}`}>
                              <Text color="primary.600" fontWeight="medium" _hover={{ textDecoration: 'underline' }}>
                                {activity.userName}/{activity.repositoryName}
                              </Text>
                            </Link>
                            <Text color="myGray.400">•</Text>
                            <Text color="myGray.500">{formatRelativeTime(activity.activityDate, t)}</Text>
                          </HStack>
                        </VStack>
                      </HStack>
                    )
                  })}
                </VStack>
              )}
            </Box>
          </Box>

          {/* 右侧边栏 */}
          <Box w="300px" flexShrink={0}>
            {/* 我的仓库 */}
            <Box borderWidth="1px" borderColor="myGray.200" borderRadius="lg" overflow="hidden" bg="white">
              <Box px={5} py={3} borderBottom="1px" borderColor="myGray.200">
                <HStack spacing={3}>
                  <Icon as={FiBook} w={5} h={5} color="myGray.600" />
                  <Heading size="sm" color="myGray.900">{t('home.myRepos')}</Heading>
                </HStack>
              </Box>

              {repos.length === 0 ? (
                <Box py={8} px={5}>
                  <VStack spacing={3}>
                    <Icon as={FiBook} w={8} h={8} color="myGray.300" />
                    <Text color="myGray.500" fontSize="sm" textAlign="center">{t('home.noRepos')}</Text>
                    <Link href="/new">
                      <Button size="xs" variant="primary">{t('home.createRepo')}</Button>
                    </Link>
                  </VStack>
                </Box>
              ) : (
                <>
                  <VStack spacing={0} align="stretch">
                    {repos.slice(0, 10).map((repo) => (
                      <Link key={`${repo.userName}/${repo.repositoryName}`} href={`/${repo.userName}/${repo.repositoryName}`}>
                        <HStack
                          spacing={3}
                          px={5}
                          py={3}
                          borderBottom="1px solid"
                          borderBottomColor="myGray.100"
                          _hover={{ bg: 'myGray.50' }}
                          transition="all 0.15s"
                          _last={{ borderBottom: 'none' }}
                        >
                          <Icon as={FiBook} w={4} h={4} color="myGray.500" flexShrink={0} />
                          <VStack align="start" spacing={0.5} flex={1}>
                            <Text fontSize="sm" fontWeight="medium" color="primary.600">
                              {repo.userName}/{repo.repositoryName}
                            </Text>
                            {repo.description && (
                              <Text fontSize="xs" color="myGray.600" noOfLines={1}>
                                {repo.description}
                              </Text>
                            )}
                          </VStack>
                          {repo.isPrivate && (
                            <Badge fontSize="2xs" colorScheme="yellow" variant="subtle" flexShrink={0}>
                              P
                            </Badge>
                          )}
                        </HStack>
                      </Link>
                    ))}
                  </VStack>

                  {repos.length > 10 && (
                    <Box px={5} py={2} borderTop="1px" borderColor="myGray.200">
                      <Link href="/repos">
                        <Button variant="whiteBase" size="xs" w="full" borderWidth="1px" borderColor="myGray.200">
                          {t('common.viewAll')}
                        </Button>
                      </Link>
                    </Box>
                  )}
                </>
              )}
            </Box>

            {/* 我的组织 */}
            <Box mt={6} borderWidth="1px" borderColor="myGray.200" borderRadius="lg" overflow="hidden" bg="white">
              <Box px={5} py={3} borderBottom="1px" borderColor="myGray.200">
                <HStack spacing={3}>
                  <Icon as={AiOutlineBank} w={5} h={5} color="myGray.600" />
                  <Heading size="sm" color="myGray.900">{t('home.myOrganizations')}</Heading>
                </HStack>
              </Box>

              {organizations.length === 0 ? (
                <Box py={8} px={5}>
                  <VStack spacing={3}>
                    <Icon as={AiOutlineBank} w={8} h={8} color="myGray.300" />
                    <Text color="myGray.500" fontSize="sm" textAlign="center">{t('home.noOrganizations')}</Text>
                    <HStack spacing={2}>
                      <Link href="/organizations?modal=open">
                        <Button size="xs" variant="primary">{t('home.createOrganization')}</Button>
                      </Link>
                      <Link href="/organizations">
                        <Button size="xs" variant="whiteBase" borderWidth="1px" borderColor="myGray.200">{t('home.browseOrganizations')}</Button>
                      </Link>
                    </HStack>
                  </VStack>
                </Box>
              ) : (
                <>
                  <VStack spacing={0} align="stretch">
                    {organizations.slice(0, 10).map((organization) => (
                      <Link key={organization.userName} href={`/organizations/${organization.userName}`}>
                        <HStack
                          spacing={3}
                          px={5}
                          py={3}
                          borderBottom="1px solid"
                          borderBottomColor="myGray.100"
                          _hover={{ bg: 'myGray.50' }}
                          transition="all 0.15s"
                          _last={{ borderBottom: 'none' }}
                        >
                          <Icon as={AiOutlineBank} w={4} h={4} color="myGray.500" flexShrink={0} />
                          <VStack align="start" spacing={0.5} flex={1}>
                            <Text fontSize="sm" fontWeight="medium" color="primary.600">
                              {organization.userName}
                            </Text>
                            {organization.description && (
                              <Text fontSize="xs" color="myGray.600" noOfLines={1}>
                                {organization.description}
                              </Text>
                            )}
                          </VStack>
                        </HStack>
                      </Link>
                    ))}
                  </VStack>

                  {organizations.length > 10 && (
                    <Box px={5} py={2} borderTop="1px" borderColor="myGray.200">
                      <Link href="/organizations">
                        <Button variant="whiteBase" size="xs" w="full" borderWidth="1px" borderColor="myGray.200">
                          {t('common.viewAll')}
                        </Button>
                      </Link>
                    </Box>
                  )}
                </>
              )}
            </Box>
          </Box>
        </Flex>
      </Container>
    </Box>
  )
}
