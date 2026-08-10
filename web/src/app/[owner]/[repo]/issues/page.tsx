'use client'

import {
  Text,
  Button,
  VStack,
  HStack,
  Box,
  Icon,
  Badge,
  Checkbox,
  Flex,
  Input,
  InputGroup,
  InputLeftElement,
  Avatar,
  useDisclosure,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
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
  FiCheckCircle,
  FiSearch,
  FiChevronDown,
  FiTag,
  FiFlag,
  FiUser,
  FiUsers,
  FiX,
} from 'react-icons/fi'
import { useEffect, useState, useCallback, useMemo } from 'react'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useI18n } from '@/contexts/I18nContext'
import { useGithubToast } from '@/app/providers'
import { api } from '@/lib/api'
import type { Issue, Label, Milestone, User, Participant } from '@/lib/types'
import { IssueListItem } from './components/IssueListItem'

const PAGE_SIZE = 25

type SortKey = 'newest' | 'oldest' | 'recentlyUpdated' | 'leastRecentlyUpdated'

// Extract current state from a query string: 'open' | 'closed' | undefined
function extractState(q: string): 'open' | 'closed' | undefined {
  if (/\bis:closed\b/i.test(q)) return 'closed'
  if (/\bis:open\b/i.test(q)) return 'open'
  return undefined
}

// Replace or add is:open/is:closed in query, returns new query string
function setStateInQuery(q: string, state: 'open' | 'closed'): string {
  const hasState = /\bis:(open|closed)\b/i
  if (hasState.test(q)) {
    return q.replace(hasState, `is:${state}`)
  }
  return `${q} is:${state}`.trim()
}

// Append a filter token (e.g. label:bug) to query if not already present
function appendFilter(q: string, token: string): string {
  if (new RegExp(`\\b${token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\b`, 'i').test(q)) {
    return q
  }
  return `${q} ${token}`.trim()
}

export default function IssuesPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const { refreshData, userRole, canCreateIssue } = useRepo()
  const { t } = useI18n()
  const toast = useGithubToast()

  const canBatchOperate = userRole === 'owner' || userRole === 'member'

  // Search state
  const [queryInput, setQueryInput] = useState('is:open')
  const [appliedQuery, setAppliedQuery] = useState('is:open')
  const [issues, setIssues] = useState<Issue[]>([])
  const [participants, setParticipants] = useState<Participant[]>([])
  const [openCount, setOpenCount] = useState(0)
  const [closedCount, setClosedCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [sort, setSort] = useState<SortKey>('recentlyUpdated')

  // Labels & milestones for dropdowns
  const [labels, setLabels] = useState<Label[]>([])
  const [milestones, setMilestones] = useState<Milestone[]>([])
  const [authors, setAuthors] = useState<User[]>([])
  const [assignees, setAssignees] = useState<User[]>([])

  // Batch state
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  const [batchLoading, setBatchLoading] = useState(false)
  const [batchAction, setBatchAction] = useState<'close' | 'reopen' | null>(null)
  const { isOpen: isBatchOpen, onOpen: onBatchOpen, onClose: onBatchClose } = useDisclosure()

  const currentState = extractState(appliedQuery) || 'open'
  const visibleIds = issues.map(i => i.issueId)
  const allVisibleSelected = visibleIds.length > 0 && visibleIds.every(id => selectedIds.includes(id))

  // 参与者用户名 -> {fullName, image} 映射，用于列表页头像渲染
  const participantsMap = useMemo(() => {
    const m = new Map<string, Participant>()
    for (const p of participants) {
      m.set(p.userName, p)
    }
    return m
  }, [participants])

  const handleSelectAllVisible = () => {
    if (allVisibleSelected) {
      setSelectedIds(prev => prev.filter(id => !visibleIds.includes(id)))
    } else {
      setSelectedIds(prev => Array.from(new Set([...prev, ...visibleIds])))
    }
  }

  const openBatchConfirm = (action: 'close' | 'reopen') => {
    if (selectedIds.length === 0) {
      toast({ title: t('repo.noIssuesSelected'), status: 'warning', duration: 2000 })
      return
    }
    setBatchAction(action)
    onBatchOpen()
  }

  const handleBatchUpdate = async () => {
    if (!batchAction || selectedIds.length === 0) return
    setBatchLoading(true)
    try {
      const result = await api.batchUpdateIssues(owner, repoName, {
        issueIds: selectedIds,
        closed: batchAction === 'close',
      })
      toast({
        title: t('repo.batchUpdateSuccess', { count: result.updated || selectedIds.length }),
        status: 'success',
        duration: 2000,
      })
      setSelectedIds([])
      onBatchClose()
      await refreshData()
      // Re-run current search to reflect the update
      runSearch(appliedQuery)
    } catch (error: any) {
      toast({
        title: t('repo.batchUpdateFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setBatchLoading(false)
    }
  }

  // Core search function — calls searchIssues API and updates state
  const runSearch = useCallback(async (q: string) => {
    setLoading(true)
    try {
      // Build effective query: ensure a state token exists (default is:open)
      let effective = q.trim()
      if (!extractState(effective)) {
        effective = setStateInQuery(effective, 'open')
      }
      // Add sort as keyword-free query modifier is not supported; sort is
      // applied by the backend via "sort:" token. Translate sort key.
      const sortTokenMap: Record<SortKey, string> = {
        newest: 'sort:created-desc',
        oldest: 'sort:created-asc',
        recentlyUpdated: 'sort:updated-desc',
        leastRecentlyUpdated: 'sort:updated-asc',
      }
      // Strip any existing sort: tokens then append the chosen one.
      effective = effective.replace(/\bsort:\S+/gi, '').trim()
      effective = `${effective} ${sortTokenMap[sort]}`.trim()

      const result = await api.searchIssues(owner, repoName, effective, 1, PAGE_SIZE)
      setIssues(result.issues || [])
      setParticipants(result.participants || [])
    } catch (err) {
      console.error('Failed to search issues:', err)
      setIssues([])
      setParticipants([])
    } finally {
      setLoading(false)
    }
  }, [owner, repoName, sort])

  // Load open/closed counts and labels/milestones once on mount
  useEffect(() => {
    const loadInitial = async () => {
      try {
        const [openRes, closedRes, labelsData, milestonesData, authorsData, assigneesData] = await Promise.all([
          api.searchIssues(owner, repoName, 'is:open', 1, 1).catch(() => ({ issues: [], total: 0, limit: 1, offset: 0, participants: [] })),
          api.searchIssues(owner, repoName, 'is:closed', 1, 1).catch(() => ({ issues: [], total: 0, limit: 1, offset: 0, participants: [] })),
          api.listLabels(owner, repoName).catch(() => []),
          api.listMilestones(owner, repoName).catch(() => []),
          api.listIssueAuthors(owner, repoName).catch(() => []),
          api.listIssueAssignees(owner, repoName).catch(() => []),
        ])
        setOpenCount((openRes as any).total || 0)
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

  // Reload counts when issues change (after batch ops etc.)
  const reloadCounts = async () => {
    try {
      const [openRes, closedRes] = await Promise.all([
        api.searchIssues(owner, repoName, 'is:open', 1, 1).catch(() => ({ issues: [], total: 0, limit: 1, offset: 0, participants: [] })),
        api.searchIssues(owner, repoName, 'is:closed', 1, 1).catch(() => ({ issues: [], total: 0, limit: 1, offset: 0, participants: [] })),
      ])
      setOpenCount((openRes as any).total || 0)
      setClosedCount((closedRes as any).total || 0)
    } catch {}
  }

  // Handlers
  const handleSearchSubmit = () => {
    setAppliedQuery(queryInput)
    reloadCounts()
  }

  const handleStateTab = (state: 'open' | 'closed') => {
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
    const cleared = 'is:open'
    setQueryInput(cleared)
    setAppliedQuery(cleared)
  }

  const hasActiveFilters = (() => {
    const q = appliedQuery.toLowerCase()
    return q.includes('label:') || q.includes('author:') || q.includes('milestone:') || q.includes('assignee:') ||
      (q.replace(/\bis:(open|closed)\b/g, '').trim().length > 0 && !q.includes('sort:'))
  })()

  return (
    <VStack spacing={4} align="stretch">
      {/* Issues / Milestones sub-tabs */}
      <HStack spacing={1} borderBottom="1px solid" borderColor="myGray.200">
        <Link href={`/${owner}/${repoName}/issues`}>
          <Button
            variant="ghost"
            size="sm"
            fontWeight="semibold"
            borderBottom="2px solid"
            borderColor="primary.600"
            mb="-1px"
            borderRadius="0"
          >
            <Icon as={FiCheckCircle} mr={1} />
            {t('repo.issues')}
          </Button>
        </Link>
        <Link href={`/${owner}/${repoName}/milestones`}>
          <Button
            variant="ghost"
            size="sm"
            fontWeight="medium"
            color="myGray.500"
            mb="-1px"
            borderRadius="0"
            _hover={{ color: 'myGray.900' }}
          >
            <Icon as={FiFlag} mr={1} />
            {t('repo.milestones')}
            {milestones.length > 0 && (
              <Badge ml={1} colorScheme="gray" fontSize="xs">{milestones.length}</Badge>
            )}
          </Button>
        </Link>
      </HStack>

      {/* Top toolbar: Open/Closed tabs + search box + filter dropdowns */}
      <HStack spacing={2} flexWrap="wrap">
        <HStack spacing={1}>
          <Button
            variant="ghost"
            size="sm"
            fontWeight={currentState === 'open' ? 'semibold' : 'medium'}
            color={currentState === 'open' ? 'myGray.900' : 'myGray.500'}
            onClick={() => handleStateTab('open')}
            _hover={{ bg: 'myGray.100' }}
          >
            {openCount} {t('repo.open')}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            fontWeight={currentState === 'closed' ? 'semibold' : 'medium'}
            color={currentState === 'closed' ? 'myGray.900' : 'myGray.500'}
            onClick={() => handleStateTab('closed')}
            _hover={{ bg: 'myGray.100' }}
          >
            {closedCount} {t('repo.closed')}
          </Button>
        </HStack>

        <HStack spacing={2} flex={1} minW="300px">
          <InputGroup size="sm">
            <InputLeftElement pointerEvents="none">
              <Icon as={FiSearch} color="myGray.400" />
            </InputLeftElement>
            <Input
              placeholder={t('repo.searchPlaceholder')}
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
            {queryInput !== 'is:open' && (
              <Box
                as="button"
                position="absolute"
                right={2}
                top="50%"
                transform="translateY(-50%)"
                color="myGray.400"
                _hover={{ color: 'myGray.700' }}
                onClick={() => { setQueryInput('is:open'); setAppliedQuery('is:open'); reloadCounts() }}
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

          <Link href={`/${owner}/${repoName}/issues/new`}>
            <Button leftIcon={<Icon as={FiPlus} />} variant="primary" size="sm">
              {t('repo.newIssue')}
            </Button>
          </Link>
        </HStack>
      </HStack>

      {/* Active filter chips + clear */}
      {hasActiveFilters && (
        <HStack spacing={2} flexWrap="wrap">
          {appliedQuery.split(/\s+/).filter(Boolean).map((token, idx) => {
            if (/^is:(open|closed)$/i.test(token)) return null
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
                      const newQuery = appliedQuery.split(/\s+/).filter((p, i) => i !== idx).join(' ') || 'is:open'
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

      {/* Batch action bar */}
      {selectedIds.length > 0 && canBatchOperate && (
        <Flex
          borderWidth="1px"
          borderRadius="lg"
          overflow="hidden"
          bg="primary.50"
          borderColor="primary.200"
          px={4}
          py={2}
          align="center"
          justify="space-between"
        >
          <HStack spacing={3}>
            <Text fontSize="sm" color="primary.700" fontWeight="medium">
              {t('repo.selectedCount', { count: selectedIds.length })}
            </Text>
            <Button
              size="xs"
              variant="ghost"
              colorScheme="primary"
              onClick={handleSelectAllVisible}
            >
              {allVisibleSelected ? t('common.cancel') : t('repo.selectAll')}
            </Button>
          </HStack>
          <HStack spacing={2}>
            <Button
              size="xs"
              variant="ghost"
              colorScheme="purple"
              onClick={() => openBatchConfirm('close')}
            >
              {t('repo.batchClose')}
            </Button>
            <Button
              size="xs"
              variant="ghost"
              colorScheme="green"
              onClick={() => openBatchConfirm('reopen')}
            >
              {t('repo.batchReopen')}
            </Button>
          </HStack>
        </Flex>
      )}

      {/* Issues list */}
      {loading ? (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
          <Box py={12}>
            <VStack spacing={4}>
              <Spinner size="md" color="myGray.400" />
              <Text color="myGray.500" fontSize="sm">{t('common.loading')}</Text>
            </VStack>
          </Box>
        </Box>
      ) : issues.length === 0 ? (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
          <Box py={12}>
            <VStack spacing={4}>
              <Text color="myGray.500" fontSize="sm">
                {hasActiveFilters ? t('repo.noIssuesMatch') : t('repo.noIssues')}
              </Text>
              {!hasActiveFilters && openCount + closedCount === 0 && (
                <Link href={`/${owner}/${repoName}/issues/new`}>
                  <Button variant="primaryOutline" size="sm">
                    {t('repo.createFirstIssue')}
                  </Button>
                </Link>
              )}
            </VStack>
          </Box>
        </Box>
      ) : (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
          {/* Header row with select-all checkbox */}
          {canBatchOperate && (
            <Box px={4} py={2} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
              <Checkbox
                isChecked={allVisibleSelected}
                onChange={handleSelectAllVisible}
                colorScheme="primary"
              >
                <Text fontSize="xs" color="myGray.500">
                  {t('repo.issuesWithLabel', { count: issues.length })}
                </Text>
              </Checkbox>
            </Box>
          )}
          <VStack spacing={0} align="stretch">
            {issues.map((issue) => {
              const isSelected = selectedIds.includes(issue.issueId)
              return (
                <IssueListItem
                  key={issue.issueId}
                  issue={issue}
                  owner={owner}
                  repoName={repoName}
                  isSelected={isSelected}
                  canBatchOperate={canBatchOperate}
                  participantsMap={participantsMap}
                  onSelectIssue={(id, checked) => {
                    if (checked) {
                      setSelectedIds(prev => [...prev, id])
                    } else {
                      setSelectedIds(prev => prev.filter(prevId => prevId !== id))
                    }
                  }}
                />
              )
            })}
          </VStack>
        </Box>
      )}

      {/* Batch confirm modal */}
      <Modal isOpen={isBatchOpen} onClose={onBatchClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('repo.batchActions')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text fontSize="sm" color="myGray.600">
              {t('repo.selectedCount', { count: selectedIds.length })}
              {' — '}
              {batchAction === 'close' ? t('repo.batchClose') : t('repo.batchReopen')}
            </Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onBatchClose}>{t('common.cancel')}</Button>
            <Button
              colorScheme={batchAction === 'close' ? 'purple' : 'green'}
              variant="solid"
              onClick={handleBatchUpdate}
              isLoading={batchLoading}
            >
              {batchLoading ? <Spinner size="sm" /> : t('common.confirm')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </VStack>
  )
}
