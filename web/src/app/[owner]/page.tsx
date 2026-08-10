'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  VStack,
  HStack,
  Avatar,
  Icon,
  Badge,
  SimpleGrid,
  Spinner,
  Button,
  Input,
  InputGroup,
  InputLeftElement,
  Tabs,
  TabList,
  Tab,
  TabPanels,
  TabPanel,
} from '@chakra-ui/react'
import { useEffect, useState, useMemo, useCallback, Suspense } from 'react'
import { useParams, useSearchParams, useRouter } from 'next/navigation'
import { api, User, Repository, Organization } from '@/lib/api'
import { FiBook, FiCalendar, FiMapPin, FiPlus, FiRss, FiSearch, FiStar, FiGrid } from 'react-icons/fi'
import { AiOutlineBank } from 'react-icons/ai'
import Link from 'next/link'
import { ContributionGraph } from '@/components/ContributionGraph'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'

type OwnerType = 'user' | 'organization' | null
type TabKey = 'overview' | 'repositories' | 'stars'

const TAB_MAP: TabKey[] = ['overview', 'repositories', 'stars']

// 内部组件，使用 useSearchParams
function OwnerPageContent() {
  const params = useParams()
  const owner = (params.owner as string) || ''
  const searchParams = useSearchParams()
  const router = useRouter()
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'

  const [ownerType, setOwnerType] = useState<OwnerType>(null)
  const [user, setUser] = useState<User | null>(null)
  const [organization, setOrganization] = useState<Organization | null>(null)
  const [repos, setRepos] = useState<Repository[]>([])
  const [starredRepos, setStarredRepos] = useState<Repository[]>([])
  const [loading, setLoading] = useState(true)
  const { user: currentUser } = useCurrentUser()
  const [canCreateRepo, setCanCreateRepo] = useState(false)
  const [repoSearch, setRepoSearch] = useState('')

  // 从 URL 读取当前 tab
  const tabParam = searchParams.get('tab') as TabKey | null
  const activeTab: TabKey = tabParam === 'repositories' || tabParam === 'stars' ? tabParam : 'overview'
  const tabIndex = useMemo(() => {
    const index = TAB_MAP.indexOf(activeTab)
    return index >= 0 ? index : 0
  }, [activeTab])

  // 处理 tab 切换
  const handleTabChange = (index: number) => {
    const newTab = TAB_MAP[index]
    if (!newTab) return
    const params = new URLSearchParams(searchParams.toString())
    if (newTab === 'overview') {
      params.delete('tab')
    } else {
      params.set('tab', newTab)
    }
    const qs = params.toString()
    router.push(`/${owner}${qs ? `?${qs}` : ''}`)
  }

  useEffect(() => {
    if (!currentUser) return
    if (ownerType === 'user' && user) {
      setCanCreateRepo(currentUser.userName === owner)
    } else if (ownerType === 'organization') {
      api.listMyOrganizations().then(organizations => {
        setCanCreateRepo((organizations || []).some(g => g.userName === owner) || currentUser.isAdmin)
      }).catch(() => setCanCreateRepo(false))
    }
  }, [currentUser, ownerType, owner, user])

  // Load owner info and repos
  useEffect(() => {
    setLoading(true)
    setStarredRepos([])
    // Try user first, then organization
    api.getUserByUsername(owner)
      .then((userData) => {
        // Check if it's actually an organization account
        if (userData.isOrganization) {
          return api.getOrganization(owner).then((organizationData) => {
            setOwnerType('organization')
            setOrganization(organizationData)
            return api.getOrganizationRepos(owner)
          })
        }
        setOwnerType('user')
        setUser(userData)
        return api.listUserRepos(owner)
      })
      .then((reposData) => {
        if (reposData) {
          setRepos(reposData || [])
        }
        setLoading(false)
      })
      .catch(() => {
        // Not a user, try organization
        api.getOrganization(owner)
          .then((organizationData) => {
            setOwnerType('organization')
            setOrganization(organizationData)
            return api.getOrganizationRepos(owner)
          })
          .then((reposData) => {
            setRepos(reposData || [])
            setLoading(false)
          })
          .catch(() => {
            setOwnerType(null)
            setLoading(false)
          })
      })
  }, [owner])

  // Load starred repos for users (lazy, only when needed)
  const loadStarredRepos = useCallback(() => {
    if (ownerType === 'user' && starredRepos.length === 0) {
      api.listStarredRepos(owner)
        .then(res => setStarredRepos(res.repos || []))
        .catch(() => setStarredRepos([]))
    }
  }, [ownerType, owner, starredRepos.length])

  useEffect(() => {
    if (activeTab === 'stars' && ownerType === 'user') {
      loadStarredRepos()
    }
  }, [activeTab, ownerType, loadStarredRepos])

  // Filtered repos for the Repositories tab
  const filteredRepos = useMemo(() => {
    if (!repoSearch.trim()) return repos
    const q = repoSearch.toLowerCase()
    return repos.filter(r =>
      r.repositoryName.toLowerCase().includes(q) ||
      (r.description?.toLowerCase().includes(q) ?? false)
    )
  }, [repos, repoSearch])

  // Popular repos for Overview tab (top 6 by last activity)
  const popularRepos = useMemo(() => {
    return [...repos]
      .sort((a, b) => new Date(b.lastActivityDate).getTime() - new Date(a.lastActivityDate).getTime())
      .slice(0, 6)
  }, [repos])

  if (loading) {
    return (
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={20} textAlign="center">
          <Spinner size="lg" />
        </Container>
      </Box>
    )
  }

  if (!ownerType) {
    return (
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={20} textAlign="center">
          <Text fontSize="lg" color="myGray.600">{t('owner.notFound', { name: owner })}</Text>
        </Container>
      </Box>
    )
  }

  // Organization page: keep simple layout (no tabs)
  if (ownerType === 'organization' && organization) {
    return (
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={8}>
          <HStack spacing={8} align="start">
            <VStack spacing={4} w="300px" align="stretch" flexShrink={0}>
              <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
                <VStack spacing={4}>
                  <Avatar size="2xl" name={organization.fullName || organization.userName} src={organization.image} />
                  <VStack spacing={1}>
                    <HStack spacing={2}>
                      <Icon as={AiOutlineBank} color="myGray.500" />
                      <Heading size="md">{organization.fullName || organization.userName}</Heading>
                    </HStack>
                    <Text color="myGray.600" fontSize="sm">@{organization.userName}</Text>
                  </VStack>
                  {organization.description && (
                    <Text fontSize="sm" color="myGray.700" textAlign="center">{organization.description}</Text>
                  )}
                </VStack>
              </Box>
            </VStack>
            <RepoList
              repos={repos}
              filteredRepos={repos}
              repoSearch={repoSearch}
              setRepoSearch={setRepoSearch}
              ownerName={organization.userName}
              canCreateRepo={canCreateRepo}
              t={t}
            />
          </HStack>
        </Container>
      </Box>
    )
  }

  if (user) {
    return (
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={8}>
          <HStack spacing={8} align="start">
            {/* Left sidebar: user profile card */}
            <VStack spacing={4} w="300px" align="stretch" flexShrink={0}>
              <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
                <VStack spacing={4}>
                  <Avatar size="2xl" name={user.fullName || user.userName} src={user.image ?? undefined} />
                  <VStack spacing={1}>
                    <Heading size="md">{user.fullName || user.userName}</Heading>
                    <Text color="myGray.600" fontSize="sm">@{user.userName}</Text>
                  </VStack>
                  {user.description && (
                    <Text fontSize="sm" color="myGray.700" textAlign="center">{user.description}</Text>
                  )}
                  <VStack spacing={2} w="full" align="start">
                    {user.location && (
                      <HStack fontSize="sm" color="myGray.600">
                        <Icon as={FiMapPin} />
                        <Text>{user.location}</Text>
                      </HStack>
                    )}
                    <HStack fontSize="sm" color="myGray.600">
                      <Icon as={FiCalendar} />
                      <Text>{t('owner.joinedAt', { date: new Date(user.createdAt || user.registeredDate).toLocaleDateString(dateLocale) })}</Text>
                    </HStack>
                    <Box as="a" href={api.getUserAtomUrl(user.userName)} target="_blank" _hover={{ textDecoration: 'none' }} w="full">
                      <HStack fontSize="sm" color="primary.600" _hover={{ color: 'primary.700' }} cursor="pointer">
                        <Icon as={FiRss} />
                        <Text>{t('repo.userAtomFeed')}</Text>
                      </HStack>
                    </Box>
                  </VStack>
                </VStack>
              </Box>
            </VStack>

            {/* Right main area: tabs + content */}
            <VStack spacing={0} flex={1} align="stretch" minW={0}>
              <Tabs
                variant="line"
                colorScheme="primary"
                index={tabIndex}
                onChange={handleTabChange}
                isLazy
              >
                <TabList borderBottomColor="myGray.200">
                  <Tab>
                    <HStack spacing={2}>
                      <Icon as={FiGrid} />
                      <Text>{t('owner.tabOverview')}</Text>
                    </HStack>
                  </Tab>
                  <Tab>
                    <HStack spacing={2}>
                      <Icon as={FiBook} />
                      <Text>{t('owner.tabRepositories')}</Text>
                      {repos.length > 0 && (
                        <Badge colorScheme="gray" fontSize="xs" variant="subtle" borderRadius="full" px={2}>
                          {repos.length}
                        </Badge>
                      )}
                    </HStack>
                  </Tab>
                  <Tab>
                    <HStack spacing={2}>
                      <Icon as={FiStar} />
                      <Text>{t('owner.tabStars')}</Text>
                      {starredRepos.length > 0 && (
                        <Badge colorScheme="gray" fontSize="xs" variant="subtle" borderRadius="full" px={2}>
                          {starredRepos.length}
                        </Badge>
                      )}
                    </HStack>
                  </Tab>
                </TabList>

                <TabPanels>
                  {/* Overview */}
                  <TabPanel px={0} pt={4}>
                    <VStack spacing={4} align="stretch">
                      {/* Contribution graph */}
                      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
                        <ContributionGraph username={user.userName} />
                      </Box>

                      {/* Popular repositories */}
                      {popularRepos.length > 0 && (
                        <VStack spacing={3} align="stretch">
                          <Heading size="sm">{t('owner.popularRepos')}</Heading>
                          <SimpleGrid columns={2} spacing={4}>
                            {popularRepos.map((repo) => (
                              <RepoCard key={`${repo.userName}/${repo.repositoryName}`} repo={repo} t={t} />
                            ))}
                          </SimpleGrid>
                        </VStack>
                      )}
                    </VStack>
                  </TabPanel>

                  {/* Repositories */}
                  <TabPanel px={0} pt={4}>
                    <RepoList
                      repos={repos}
                      filteredRepos={filteredRepos}
                      repoSearch={repoSearch}
                      setRepoSearch={setRepoSearch}
                      ownerName={user.userName}
                      canCreateRepo={canCreateRepo}
                      t={t}
                    />
                  </TabPanel>

                  {/* Stars */}
                  <TabPanel px={0} pt={4}>
                    <VStack spacing={4} align="stretch">
                      {starredRepos.length === 0 ? (
                        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" py={12}>
                          <VStack spacing={4}>
                            <Icon as={FiStar} w={12} h={12} color="myGray.400" />
                            <Text color="myGray.600">{t('owner.noStarredRepos')}</Text>
                          </VStack>
                        </Box>
                      ) : (
                        <SimpleGrid columns={2} spacing={4}>
                          {starredRepos.map((repo) => (
                            <RepoCard key={`${repo.userName}/${repo.repositoryName}`} repo={repo} t={t} />
                          ))}
                        </SimpleGrid>
                      )}
                    </VStack>
                  </TabPanel>
                </TabPanels>
              </Tabs>
            </VStack>
          </HStack>
        </Container>
      </Box>
    )
  }

  return null
}

// 主页面组件，用 Suspense 包裹
export default function OwnerPage() {
  const { t } = useI18n()
  return (
    <Suspense fallback={
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={20} textAlign="center">
          <Spinner size="lg" />
        </Container>
      </Box>
    }>
      <OwnerPageContent />
    </Suspense>
  )
}

// Repository card for grid display
function RepoCard({ repo, t }: { repo: Repository; t: (key: string, options?: any) => string }) {
  return (
    <Link href={`/${repo.userName}/${repo.repositoryName}`}>
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={4} _hover={{ shadow: 'md', cursor: 'pointer' }} h="full">
        <VStack spacing={2} align="start">
          <HStack>
            <Icon as={FiBook} color="myGray.600" />
            <Heading size="sm" color="primary.600">{repo.repositoryName}</Heading>
            {repo.isPrivate && <Badge size="sm" colorScheme="gray">{t('common.private')}</Badge>}
          </HStack>
          {repo.description && (
            <Text fontSize="sm" color="myGray.600" noOfLines={2}>{repo.description}</Text>
          )}
          <HStack spacing={4} fontSize="xs" color="myGray.500">
            <HStack spacing={1}>
              <Box w={3} h={3} borderRadius="full" bg="blue.500" />
              <Text>{repo.defaultBranch}</Text>
            </HStack>
          </HStack>
        </VStack>
      </Box>
    </Link>
  )
}

// Full repository list with search and header
function RepoList({
  repos,
  filteredRepos,
  repoSearch,
  setRepoSearch,
  ownerName,
  canCreateRepo,
  t,
}: {
  repos: Repository[]
  filteredRepos: Repository[]
  repoSearch: string
  setRepoSearch: (v: string) => void
  ownerName: string
  canCreateRepo: boolean
  t: (key: string, options?: any) => string
}) {
  return (
    <VStack spacing={4} flex={1} align="stretch">
      <HStack justify="space-between">
        <HStack spacing={3}>
          <Heading size="md">{t('owner.repos')}</Heading>
          <Badge colorScheme="blue" fontSize="md">{repos.length}</Badge>
        </HStack>
        {canCreateRepo && (
          <Link href={`/new?owner=${ownerName}`}>
            <Button leftIcon={<Icon as={FiPlus} />} variant="primary" size="sm">
              {t('owner.newRepo')}
            </Button>
          </Link>
        )}
      </HStack>

      {repos.length > 0 && (
        <InputGroup size="sm" maxW="400px">
          <InputLeftElement pointerEvents="none">
            <Icon as={FiSearch} color="myGray.400" />
          </InputLeftElement>
          <Input
            placeholder={t('owner.searchRepos')}
            value={repoSearch}
            onChange={(e) => setRepoSearch(e.target.value)}
            bg="white"
            borderWidth="1px"
            borderColor="myGray.200"
            borderRadius="md"
          />
        </InputGroup>
      )}

      {repos.length === 0 ? (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" py={12}>
          <VStack spacing={4}>
            <Icon as={FiBook} w={12} h={12} color="myGray.400" />
            <Text color="myGray.600">{t('owner.noRepos')}</Text>
            {canCreateRepo && (
              <Link href={`/new?owner=${ownerName}`}>
                <Button variant="primary">{t('owner.createFirstRepo')}</Button>
              </Link>
            )}
          </VStack>
        </Box>
      ) : filteredRepos.length === 0 ? (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" py={12}>
          <VStack spacing={4}>
            <Text color="myGray.600">{t('owner.noMatchRepos')}</Text>
          </VStack>
        </Box>
      ) : (
        <SimpleGrid columns={2} spacing={4}>
          {filteredRepos.map((repo) => (
            <RepoCard key={`${repo.userName}/${repo.repositoryName}`} repo={repo} t={t} />
          ))}
        </SimpleGrid>
      )}
    </VStack>
  )
}
