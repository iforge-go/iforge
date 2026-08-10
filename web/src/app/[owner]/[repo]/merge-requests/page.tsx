'use client'

import {
  Text,
  Button,
  VStack,
  HStack,
  Box,
  Icon,
  Badge,
  Input,
  InputGroup,
  InputLeftElement,
  Avatar,
  Spinner,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  MenuGroup,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import {
  FiPlus,
  FiGitPullRequest,
  FiSearch,
  FiChevronDown,
  FiUser,
  FiUsers,
  FiTag,
  FiFlag,
  FiCheckCircle,
  FiX,
} from 'react-icons/fi'
import { useEffect, useState, useCallback } from 'react'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useI18n } from '@/contexts/I18nContext'
import { api } from '@/lib/api'
import { formatRelativeTime } from '@/lib/time'
import type { Issue, Label, Milestone, User } from '@/lib/types'

const PAGE_SIZE = 25

type SortKey = 'newest' | 'oldest' | 'recentlyUpdated' | 'leastRecentlyUpdated'
type MRState = 'open' | 'merged' | 'closed'

function getTextColor(bgColor: string): string {
  const hex = bgColor.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  const brightness = (r * 299 + g * 587 + b * 114) / 1000
  return brightness > 128 ? 'myGray.800' : 'white'
}

// Extract current state from a query string
function extractState(q: string): MRState | undefined {
  if (/\bis:merged\b/i.test(q)) return 'merged'
  if (/\bis:closed\b/i.test(q)) return 'closed'
  if (/\bis:open\b/i.test(q)) return 'open'
  return undefined
}

// Replace or add is:open/is:closed/is:merged in query
function setStateInQuery(q: string, state: MRState): string {
  const hasState = /\bis:(open|closed|merged)\b/i
  if (hasState.test(q)) {
    return q.replace(hasState, `is:${state}`)
  }
  return `${q} is:${state}`.trim()
}

// Append a filter token if not already present
function appendFilter(q: string, token: string): string {
  if (new RegExp(`\\b${token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\b`, 'i').test(q)) {
    return q
  }
  return `${q} ${token}`.trim()
}

export default function MergeRequestsPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const { canCreateIssue } = useRepo()
  const { t } = useI18n()

  const canCreateMR = canCreateIssue

  // Search state — base query always includes is:mr
  const [queryInput, setQueryInput] = useState('is:mr is:open')
  const [appliedQuery, setAppliedQuery] = useState('is:mr is:open')
  const [mergeRequests, setMergeRequests] = useState<Issue[]>([])
  const [openCount, setOpenCount] = useState(0)
  const [mergedCount, setMergedCount] = useState(0)
  const [closedCount, setClosedCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [sort, setSort] = useState<SortKey>('recentlyUpdated')

  // Filter dropdown data
  const [labels, setLabels] = useState<Label[]>([])
  const [milestones, setMilestones] = useState<Milestone[]>([])
  const [authors, setAuthors] = useState<User[]>([])
  const [assignees, setAssignees] = useState<User[]>([])

  const currentState = extractState(appliedQuery) || 'open'

  const runSearch = useCallback(async (q: string) => {
    setLoading(true)
    try {
      let effective = q.trim()
      if (!extractState(effective)) {
        effective = setStateInQuery(effective, 'open')
      }
      const sortTokenMap: Record<SortKey, string> = {
        newest: 'sort:created-desc',
        oldest: 'sort:created-asc',
        recentlyUpdated: 'sort:updated-desc',
        leastRecentlyUpdated: 'sort:updated-asc',
      }
      effective = effective.replace(/\bsort:\S+/gi, '').trim()
      effective = `${effective} ${sortTokenMap[sort]}`.trim()

      const result = await api.searchIssues(owner, repoName, effective, 1, PAGE_SIZE)
      setMergeRequests(result.issues || [])
    } catch (err) {
      console.error('Failed to search merge requests:', err)
      setMergeRequests([])
    } finally {
      setLoading(false)
    }
  }, [owner, repoName, sort])

  // Load counts and filter data once on mount
  useEffect(() => {
    const loadInitial = async () => {
      try {
        const [openRes, mergedRes, closedRes, labelsData, milestonesData, authorsData, assigneesData] = await Promise.all([
          api.searchIssues(owner, repoName, 'is:mr is:open', 1, 1).catch(() => ({ issues: [], total: 0, limit: 1, offset: 0, participants: [] })),
          api.searchIssues(owner, repoName, 'is:mr is:merged', 1, 1).catch(() => ({ issues: [], total: 0, limit: 1, offset: 0, participants: [] })),
          api.searchIssues(owner, repoName, 'is:mr is:closed', 1, 1).catch(() => ({ issues: [], total: 0, limit: 1, offset: 0, participants: [] })),
          api.listLabels(owner, repoName).catch(() => []),
          api.listMilestones(owner, repoName).catch(() => []),
          api.listIssueAuthors(owner, repoName, 'mr').catch(() => []),
          api.listIssueAssignees(owner, repoName, 'mr').catch(() => []),
        ])
        setOpenCount((openRes as any).total || 0)
        setMergedCount((mergedRes as any).total || 0)
        setClosedCount((closedRes as any).total || 0)
        setLabels(labelsData || [])
        setMilestones(milestonesData || [])
        setAuthors(authorsData || [])
        setAssignees(assigneesData || [])
      } catch (err) {
        console.error('Failed to load initial data:', err)
      }
    }
    loadInitial()
  }, [owner, repoName])

  // Run search whenever appliedQuery or sort changes
  useEffect(() => {
    runSearch(appliedQuery)
  }, [appliedQuery, runSearch])

  const reloadCounts = async () => {
    try {
      const [openRes, mergedRes, closedRes] = await Promise.all([
        api.searchIssues(owner, repoName, 'is:mr is:open', 1, 1).catch(() => ({ issues: [], total: 0, limit: 1, offset: 0, participants: [] })),
        api.searchIssues(owner, repoName, 'is:mr is:merged', 1, 1).catch(() => ({ issues: [], total: 0, limit: 1, offset: 0, participants: [] })),
        api.searchIssues(owner, repoName, 'is:mr is:closed', 1, 1).catch(() => ({ issues: [], total: 0, limit: 1, offset: 0, participants: [] })),
      ])
      setOpenCount((openRes as any).total || 0)
      setMergedCount((mergedRes as any).total || 0)
      setClosedCount((closedRes as any).total || 0)
    } catch {}
  }

  const handleSearchSubmit = () => {
    setAppliedQuery(queryInput)
    reloadCounts()
  }

  const handleStateTab = (state: MRState) => {
    const newQuery = setStateInQuery(queryInput, state)
    setQueryInput(newQuery)
    setAppliedQuery(newQuery)
  }

  const handleAddFilter = (token: string) => {
    const newQuery = appendFilter(queryInput, token)
    setQueryInput(newQuery)
    setAppliedQuery(newQuery)
  }

  const handleClearFilters = () => {
    const cleared = 'is:mr is:open'
    setQueryInput(cleared)
    setAppliedQuery(cleared)
  }

  const hasActiveFilters = (() => {
    const q = appliedQuery.toLowerCase()
    return q.includes('label:') || q.includes('author:') || q.includes('milestone:') || q.includes('assignee:') ||
      (q.replace(/\bis:(open|closed|merged)\b/g, '').replace(/\bis:mr\b/g, '').trim().length > 0 && !q.includes('sort:'))
  })()

  const tabs: { key: MRState; label: string; count: number }[] = [
    { key: 'open', label: t('repo.open'), count: openCount },
    { key: 'merged', label: t('repo.merged'), count: mergedCount },
    { key: 'closed', label: t('repo.closed'), count: closedCount },
  ]

  return (
    <VStack spacing={4} align="stretch">
      {/* Top toolbar: state tabs + search box + filter dropdowns */}
      <HStack spacing={2} flexWrap="wrap">
        <HStack spacing={1}>
          {tabs.map((tab) => (
            <Button
              key={tab.key}
              variant="ghost"
              size="sm"
              fontWeight={currentState === tab.key ? 'semibold' : 'medium'}
              color={currentState === tab.key ? 'myGray.900' : 'myGray.500'}
              onClick={() => handleStateTab(tab.key)}
              _hover={{ bg: 'myGray.100' }}
            >
              {tab.count} {tab.label}
            </Button>
          ))}
        </HStack>

        <HStack spacing={2} flex={1} minW="300px">
          <InputGroup size="sm">
            <InputLeftElement pointerEvents="none">
              <Icon as={FiSearch} color="myGray.400" />
            </InputLeftElement>
            <Input
              placeholder={t('repo.searchMRPlaceholder')}
              value={queryInput}
              onChange={(e) => setQueryInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSearchSubmit()
              }}
              bg="white"
              borderWidth="1px"
              borderColor="myGray.200"
              borderRadius="md"
              pr={8}
            />
            {queryInput !== 'is:mr is:open' && (
              <Box
                as="button"
                position="absolute"
                right={2}
                top="50%"
                transform="translateY(-50%)"
                color="myGray.400"
                _hover={{ color: 'myGray.700' }}
                onClick={() => { setQueryInput('is:mr is:open'); setAppliedQuery('is:mr is:open'); reloadCounts() }}
                aria-label={t('repo.clearFilters')}
              >
                <Icon as={FiX} />
              </Box>
            )}
          </InputGroup>
        </HStack>

        <HStack spacing={2}>
          {/* Author dropdown */}
          <Menu>
            <MenuButton as={Button} rightIcon={<Icon as={FiChevronDown} w={3} h={3} />} leftIcon={<Icon as={FiUser} />} size="sm" variant="whiteBase">
              {t('repo.author')}
            </MenuButton>
            <MenuList maxH="400px" overflowY="auto" zIndex={10}>
              {authors.length === 0 ? (
                <MenuItem isDisabled>{t('repo.noAuthor')}</MenuItem>
              ) : (
                authors.map((author) => (
                  <MenuItem key={author.userName} onClick={() => handleAddFilter(`author:${author.userName}`)}>
                    <HStack spacing={2} flex={1}>
                      <Avatar size="2xs" name={author.userName} src={author.image || undefined} />
                      <Text fontSize="sm">{author.fullName || author.userName}</Text>
                    </HStack>
                  </MenuItem>
                ))
              )}
            </MenuList>
          </Menu>

          {/* Assignee dropdown */}
          <Menu>
            <MenuButton as={Button} rightIcon={<Icon as={FiChevronDown} w={3} h={3} />} leftIcon={<Icon as={FiUsers} />} size="sm" variant="whiteBase">
              {t('repo.assignee')}
            </MenuButton>
            <MenuList maxH="400px" overflowY="auto" zIndex={10}>
              {assignees.length === 0 ? (
                <MenuItem isDisabled>{t('repo.noAssignee')}</MenuItem>
              ) : (
                assignees.map((assignee) => (
                  <MenuItem key={assignee.userName} onClick={() => handleAddFilter(`assignee:${assignee.userName}`)}>
                    <HStack spacing={2} flex={1}>
                      <Avatar size="2xs" name={assignee.userName} src={assignee.image || undefined} />
                      <Text fontSize="sm">{assignee.fullName || assignee.userName}</Text>
                    </HStack>
                  </MenuItem>
                ))
              )}
            </MenuList>
          </Menu>

          {/* Labels dropdown */}
          <Menu>
            <MenuButton as={Button} rightIcon={<Icon as={FiChevronDown} w={3} h={3} />} leftIcon={<Icon as={FiTag} />} size="sm" variant="whiteBase">
              {t('repo.labels')}
            </MenuButton>
            <MenuList maxH="400px" overflowY="auto" zIndex={10}>
              {labels.length === 0 ? (
                <MenuItem isDisabled>{t('repo.noLabel')}</MenuItem>
              ) : (
                labels.map((label) => (
                  <MenuItem key={label.labelId} onClick={() => handleAddFilter(`label:${label.labelName}`)}>
                    <HStack spacing={2} flex={1}>
                      <Box w={3} h={3} borderRadius="full" bg={`#${label.color}`} flexShrink={0} />
                      <Text fontSize="sm">{label.labelName}</Text>
                    </HStack>
                  </MenuItem>
                ))
              )}
            </MenuList>
          </Menu>

          {/* Milestones dropdown */}
          <Menu>
            <MenuButton as={Button} rightIcon={<Icon as={FiChevronDown} w={3} h={3} />} leftIcon={<Icon as={FiFlag} />} size="sm" variant="whiteBase">
              {t('repo.milestones')}
            </MenuButton>
            <MenuList maxH="400px" overflowY="auto" zIndex={10}>
              {milestones.length === 0 ? (
                <MenuItem isDisabled>{t('repo.noMilestone')}</MenuItem>
              ) : (
                milestones.map((milestone) => (
                  <MenuItem key={milestone.milestoneId} onClick={() => handleAddFilter(`milestone:${milestone.title}`)}>
                    <Text fontSize="sm">{milestone.title}</Text>
                  </MenuItem>
                ))
              )}
            </MenuList>
          </Menu>

          {/* Sort dropdown */}
          <Menu>
            <MenuButton as={Button} rightIcon={<Icon as={FiChevronDown} w={3} h={3} />} size="sm" variant="whiteBase">
              {t('repo.sort')}
            </MenuButton>
            <MenuList zIndex={10}>
              <MenuGroup title={t('repo.sort')}>
                <MenuItem onClick={() => setSort('recentlyUpdated')} icon={sort === 'recentlyUpdated' ? <Icon as={FiCheckCircle} color="primary.500" /> : undefined}>
                  {t('repo.recentlyUpdated')}
                </MenuItem>
                <MenuItem onClick={() => setSort('leastRecentlyUpdated')} icon={sort === 'leastRecentlyUpdated' ? <Icon as={FiCheckCircle} color="primary.500" /> : undefined}>
                  {t('repo.leastRecentlyUpdated')}
                </MenuItem>
                <MenuItem onClick={() => setSort('newest')} icon={sort === 'newest' ? <Icon as={FiCheckCircle} color="primary.500" /> : undefined}>
                  {t('repo.newest')}
                </MenuItem>
                <MenuItem onClick={() => setSort('oldest')} icon={sort === 'oldest' ? <Icon as={FiCheckCircle} color="primary.500" /> : undefined}>
                  {t('repo.oldest')}
                </MenuItem>
              </MenuGroup>
            </MenuList>
          </Menu>

          {canCreateMR && (
            <Link href={`/${owner}/${repoName}/merge-requests/new`}>
              <Button leftIcon={<Icon as={FiPlus} />} variant="primary" size="sm">
                {t('repo.newMergeRequest')}
              </Button>
            </Link>
          )}
        </HStack>
      </HStack>

      {/* Active filter chips + clear */}
      {hasActiveFilters && (
        <HStack spacing={2} flexWrap="wrap">
          {appliedQuery.split(/\s+/).filter(Boolean).map((token, idx) => {
            if (/^is:(open|closed|merged|mr)$/i.test(token)) return null
            if (/^sort:/i.test(token)) return null
            return (
              <Badge key={`filter-${idx}`} colorScheme="blue" variant="subtle" px={2} py={1} borderRadius="full">
                <HStack spacing={1}>
                  <Text fontSize="xs">{token}</Text>
                  <Icon
                    as={FiX}
                    w={3}
                    h={3}
                    cursor="pointer"
                    onClick={() => {
                      const newQuery = appliedQuery.split(/\s+/).filter((p, i) => i !== idx).join(' ') || 'is:mr is:open'
                      setQueryInput(newQuery)
                      setAppliedQuery(newQuery)
                    }}
                  />
                </HStack>
              </Badge>
            )
          })}
          <Button size="xs" variant="ghost" colorScheme="blue" onClick={handleClearFilters}>
            {t('repo.clearFilters')}
          </Button>
        </HStack>
      )}

      {/* MR list */}
      {loading ? (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
          <Box py={12}>
            <VStack spacing={4}>
              <Spinner size="md" color="myGray.400" />
              <Text color="myGray.500" fontSize="sm">{t('common.loading')}</Text>
            </VStack>
          </Box>
        </Box>
      ) : mergeRequests.length === 0 ? (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
          <Box py={12}>
            <VStack spacing={4}>
              <Icon as={FiGitPullRequest} w={8} h={8} color="myGray.300" />
              <Text color="myGray.500" fontSize="sm">
                {hasActiveFilters ? t('repo.noMergeRequestsMatch') : t('repo.noMergeRequests')}
              </Text>
              {!hasActiveFilters && canCreateMR && openCount + mergedCount + closedCount === 0 && (
                <Link href={`/${owner}/${repoName}/merge-requests/new`}>
                  <Button variant="primaryOutline" size="sm">
                    {t('repo.createFirstMergeRequest')}
                  </Button>
                </Link>
              )}
            </VStack>
          </Box>
        </Box>
      ) : (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
          <Box px={4} py={2} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
            <Text fontSize="xs" color="myGray.500">
              {t('repo.mergeRequestsWithLabel', { count: mergeRequests.length })}
            </Text>
          </Box>
          <VStack spacing={0} align="stretch">
            {mergeRequests.map((mr) => (
              <Box
                key={mr.issueId}
                py={3}
                px={4}
                borderBottom="1px"
                borderColor="myGray.100"
                _last={{ borderBottom: 'none' }}
                _hover={{ bg: 'myGray.50' }}
              >
                <HStack spacing={3} align="start">
                  <Icon
                    as={FiGitPullRequest}
                    color={mr.closed ? (currentState === 'merged' ? 'purple.500' : 'myGray.400') : 'green.500'}
                    w={4}
                    h={4}
                    mt={1}
                    flexShrink={0}
                  />
                  <VStack align="start" spacing={1} flex={1}>
                    <HStack spacing={2} flexWrap="wrap">
                      <Link href={`/${owner}/${repoName}/merge-requests/${mr.issueId}`}>
                        <Text fontSize="sm" fontWeight="medium" color="myGray.900" _hover={{ color: 'primary.600' }}>
                          {mr.title}
                        </Text>
                      </Link>
                      {mr.labels && mr.labels.length > 0 && (
                        <HStack spacing={1}>
                          {mr.labels.map((label) => (
                            <Badge
                              key={label.labelId}
                              bg={`#${label.color}`}
                              color={getTextColor(label.color)}
                              fontSize="10px"
                              px={1.5}
                              py={0.5}
                              borderRadius="full"
                              fontWeight="normal"
                            >
                              {label.labelName}
                            </Badge>
                          ))}
                        </HStack>
                      )}
                    </HStack>
                    <Text fontSize="xs" color="myGray.500">
                      {t('repo.openedBy', {
                        id: mr.issueId,
                        user: mr.openedUserName,
                        date: formatRelativeTime(mr.registeredDate, t),
                      })}
                    </Text>
                  </VStack>
                  {mr.commentsCount > 0 && (
                    <HStack spacing={1} mt={1}>
                      <Icon as={FiGitPullRequest} w={3} h={3} color="myGray.400" />
                      <Text fontSize="xs" color="myGray.500">{mr.commentsCount}</Text>
                    </HStack>
                  )}
                </HStack>
              </Box>
            ))}
          </VStack>
        </Box>
      )}
    </VStack>
  )
}
