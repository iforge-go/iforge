'use client'

import { useState, useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import {
  Box,
  Button,
  Card,
  CardBody,
  Flex,
  Heading,
  HStack,
  Stack,
  Text,
  Badge,
  Progress,
  Stat,
  StatGroup,
  StatLabel,
  StatNumber,
  IconButton,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  useToast,
  Spinner,
  Drawer,
  DrawerOverlay,
  DrawerContent,
  DrawerCloseButton,
  DrawerHeader,
  DrawerBody,
  DrawerFooter,
  Input,
  VStack,
  Divider,
} from '@chakra-ui/react'
import { FiArrowLeft, FiEdit, FiTrash2, FiMoreVertical, FiPlus, FiX, FiTarget, FiActivity } from 'react-icons/fi'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import { useProject } from '../../ProjectContext'
import { getStoryStatusColor } from '@/lib/storyStatus'
import ActivityTimeline from '@/components/ActivityTimeline'
import EpicDrawer, { EpicFormData } from '../components/EpicDrawer'
import { Epic, UserStory, EpicSprintInfo } from '@/lib/types'
import { ConfirmDialog } from '@/components/ConfirmDialog'

export default function EpicDetailPage() {
  const router = useRouter()
  const params = useParams()
  const projectSlug = params.slug as string
  const epicSlug = params.epicSlug as string
  const { t } = useI18n()
  const toast = useToast()
  const { user, authLoading } = useCurrentUser()
  const { canEditScrum, members } = useProject()

  const [epic, setEpic] = useState<Epic | null>(null)
  const [loading, setLoading] = useState(true)
  const [stories, setStories] = useState<UserStory[]>([])
  const [sprints, setSprints] = useState<EpicSprintInfo[]>([])
  const [activities, setActivities] = useState<any[]>([])
  const [participants, setParticipants] = useState<any[]>([])

  // 编辑 Drawer
  const [isEditDrawerOpen, setIsEditDrawerOpen] = useState(false)

  // 关联 Story Drawer
  const [isLinkDrawerOpen, setIsLinkDrawerOpen] = useState(false)
  const [allStories, setAllStories] = useState<UserStory[]>([])
  const [linkSearch, setLinkSearch] = useState('')
  const [linkLoading, setLinkLoading] = useState(false)

  const [confirmOpen, setConfirmOpen] = useState(false)
  const [confirmAction, setConfirmAction] = useState<(() => void) | null>(null)

  useEffect(() => {
    if (!user || !epicSlug) return
    Promise.all([
      api.getEpic(epicSlug).then(setEpic).catch(() => setEpic(null)).finally(() => setLoading(false)),
      api.getEpicStories(epicSlug).then(setStories).catch(() => setStories([])),
      api.getEpicSprints(epicSlug).then(setSprints).catch(() => setSprints([])),
      api.getActivities(projectSlug, 'epic', epicSlug).then((data) => {
        setActivities(data?.activities || [])
        setParticipants(data?.participants || [])
      }).catch(() => {
        setActivities([])
        setParticipants([])
      }),
    ])
  }, [user, epicSlug, projectSlug])

  const refreshEpic = async () => {
    try {
      const updated = await api.getEpic(epicSlug)
      setEpic(updated)
    } catch {}
  }

  const handleEditClick = () => setIsEditDrawerOpen(true)

  const handleSave = async (data: EpicFormData) => {
    try {
      await api.updateEpic(epicSlug, {
        title: data.title,
        description: data.description || null,
        goal: data.goal || null,
        status: data.status,
        priority: data.priority,
        startDate: data.startDate || null,
        targetDate: data.targetDate || null,
        ownerName: data.ownerName || null,
      })
      await refreshEpic()
      toast({ title: t('common.success'), description: t('pms.epicUpdated'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
      throw error
    }
  }

  const handleDelete = () => {
    setConfirmAction(() => async () => {
      try {
        await api.deleteEpic(epicSlug)
        router.push(`/projects/${projectSlug}/epics`)
      } catch (error: any) {
        toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
      }
    })
    setConfirmOpen(true)
  }

  const handleConfirm = () => {
    confirmAction?.()
    setConfirmOpen(false)
    setConfirmAction(null)
  }

  const handleOpenLinkDrawer = async () => {
    setIsLinkDrawerOpen(true)
    setLinkSearch('')
    setLinkLoading(true)
    try {
      const all = await api.getUserStories(projectSlug)
      setAllStories(all)
    } catch {
      setAllStories([])
    } finally {
      setLinkLoading(false)
    }
  }

  const handleLinkStory = async (storySlug: string) => {
    try {
      await api.assignStoryToEpic(epicSlug, storySlug)
      // 刷新 stories 列表 + epic 进度
      const refreshed = await api.getEpicStories(epicSlug)
      setStories(refreshed || [])
      await refreshEpic()
      toast({ title: t('common.success'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    }
  }

  const handleUnlinkStory = async (storySlug: string) => {
    try {
      await api.removeStoryFromEpic(epicSlug, storySlug)
      setStories(prev => prev.filter(s => s.slug !== storySlug))
      await refreshEpic()
      toast({ title: t('common.success'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    }
  }

  const getStatusName = (status: string): string => {
    const map: Record<string, string> = {
      open: t('pms.statusOpen'),
      in_progress: t('pms.statusInProgress'),
      done: t('pms.statusDone'),
      closed: t('pms.statusClosed'),
    }
    return map[status] || status
  }

  const getStoryStatusName = (slug: string): string => {
    const camelSlug = slug.split('_').map(part => part.charAt(0).toUpperCase() + part.slice(1)).join('')
    const key = `pms.status${camelSlug}`
    const translated = t(key)
    return translated === key ? slug : translated
  }

  const getPriorityName = (priority: string): string => {
    const map: Record<string, string> = {
      low: t('pms.priorityLow'),
      medium: t('pms.priorityMedium'),
      high: t('pms.priorityHigh'),
      urgent: t('pms.priorityUrgent'),
    }
    return map[priority] || priority
  }

  const getPriorityColor = (priority: string): string => {
    const map: Record<string, string> = {
      low: 'gray.500',
      medium: 'blue.500',
      high: 'orange.500',
      urgent: 'red.500',
    }
    return map[priority] || 'gray.500'
  }

  if (authLoading) return null

  if (!user) {
    return (
      <Box py={10} textAlign="center">
        <Text>{t('common.pleaseLogin')}</Text>
      </Box>
    )
  }

  if (loading) {
    return (
      <Flex justify="center" align="center" minH="60vh">
        <Spinner size="xl" color="blue.500" />
      </Flex>
    )
  }

  if (!epic) {
    return (
      <Box py={10} textAlign="center">
        <Stack spacing={4}>
          <Heading size="lg">{t('common.noData')}</Heading>
          <Button colorScheme="blue" onClick={() => router.push(`/projects/${projectSlug}/epics`)}>
            {t('common.back')}
          </Button>
        </Stack>
      </Box>
    )
  }

  return (
    <>
      <Stack spacing={6}>
        {/* 面包屑 + 标题区 */}
        <Box>
          <HStack fontSize="sm" color="gray.600" mb={3} spacing={1}>
            <Link href={`/projects/${projectSlug}/epics`} style={{ textDecoration: 'none' }}>
              <Text _hover={{ color: 'blue.600' }}>{t('pms.epics')}</Text>
            </Link>
            <Text>/</Text>
            <Text color="gray.800" fontWeight="medium">{epic.title}</Text>
          </HStack>

          <Flex justify="space-between" align="flex-start">
            <HStack align="flex-start" spacing={3}>
              <Box mt={1} color="purple.500"><FiTarget size={24} /></Box>
              <Box>
                <Heading size="lg">{epic.title}</Heading>
                <HStack mt={2} spacing={3}>
                  <Badge
                    borderWidth="1px"
                    borderStyle="solid"
                    bg="transparent"
                    color={getStoryStatusColor(epic.status)}
                    borderColor={getStoryStatusColor(epic.status)}
                  >
                    {getStatusName(epic.status)}
                    {epic.statusManual && <Text as="span" fontSize="2xs" ml={1} opacity={0.7}>({t('pms.manual')})</Text>}
                  </Badge>
                  <HStack spacing={1}>
                    <Box w="8px" h="8px" borderRadius="full" bg={getPriorityColor(epic.priority)} />
                    <Text fontSize="sm" color="gray.600">{getPriorityName(epic.priority)}</Text>
                  </HStack>
                  {(epic.startDate || epic.targetDate) && (
                    <Text fontSize="sm" color="gray.600">
                      📅 {epic.startDate ? new Date(epic.startDate).toLocaleDateString() : '—'}
                      {epic.targetDate && (
                        <>
                          {' '}&rarr;{' '}
                          {new Date(epic.targetDate).toLocaleDateString()}
                        </>
                      )}
                    </Text>
                  )}
                  {epic.ownerName && (
                    <Text fontSize="sm" color="gray.600">👤 {epic.ownerName}</Text>
                  )}
                </HStack>
              </Box>
            </HStack>

            <HStack spacing={2}>
              <Button
                size="sm"
                variant="ghost"
                leftIcon={<FiArrowLeft />}
                onClick={() => router.push(`/projects/${projectSlug}/epics`)}
              >
                {t('common.back')}
              </Button>
              {canEditScrum && (
                <>
                  <Button size="sm" colorScheme="blue" leftIcon={<FiEdit />} onClick={handleEditClick}>
                    {t('common.edit')}
                  </Button>
                  <Menu>
                    <MenuButton as={IconButton} icon={<FiMoreVertical />} variant="ghost" size="sm" aria-label="more" />
                    <MenuList>
                      <MenuItem icon={<FiTrash2 />} color="red.500" onClick={handleDelete}>
                        {t('common.delete')}
                      </MenuItem>
                    </MenuList>
                  </Menu>
                </>
              )}
            </HStack>
          </Flex>
        </Box>

        {/* 进度卡片 */}
        <Card bg="white" borderRadius="lg" border="1px solid" borderColor="myGray.200">
          <CardBody>
            <StatGroup mb={4}>
              <Stat>
                <StatLabel color="gray.500" fontSize="xs">{t('pms.epicProgress')}</StatLabel>
                <StatNumber fontSize="2xl">{epic.progressPercent || 0}%</StatNumber>
              </Stat>
              <Stat>
                <StatLabel color="gray.500" fontSize="xs">{t('pms.epicStories')}</StatLabel>
                <StatNumber fontSize="2xl">{epic.doneStories || 0}/{epic.totalStories || 0}</StatNumber>
              </Stat>
              <Stat>
                <StatLabel color="gray.500" fontSize="xs">{t('pms.totalStoryPoints')}</StatLabel>
                <StatNumber fontSize="2xl" color="purple.500">{epic.doneStoryPoints || 0}/{epic.totalStoryPoints || 0}</StatNumber>
              </Stat>
              <Stat>
                <StatLabel color="gray.500" fontSize="xs">{t('pms.epicSprints')}</StatLabel>
                <StatNumber fontSize="2xl" color="blue.500">{epic.sprintCount || 0}</StatNumber>
              </Stat>
            </StatGroup>
            <Progress value={epic.progressPercent || 0} size="sm" colorScheme="green" borderRadius="full" />

            {/* 跨 Sprint 列表 */}
            {sprints.length > 0 && (
              <Box mt={4}>
                <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={2}>{t('pms.epicSprints')}</Text>
                <HStack spacing={2} flexWrap="wrap">
                  {sprints.map(s => (
                    <Link key={s.sprintSlug} href={`/projects/${projectSlug}/sprint/${s.sprintSlug}`} style={{ textDecoration: 'none' }}>
                      <Badge variant="subtle" colorScheme="blue" cursor="pointer" _hover={{ bg: 'blue.100' }}>
                        {s.title} · {s.taskCount} {t('pms.tasks').toLowerCase()}
                      </Badge>
                    </Link>
                  ))}
                </HStack>
              </Box>
            )}
          </CardBody>
        </Card>

        {/* Goal + Description */}
        {(epic.goal || epic.description) && (
          <Card bg="white" borderRadius="lg" border="1px solid" borderColor="myGray.200">
            <CardBody>
              <Stack spacing={4}>
                {epic.goal && (
                  <Box>
                    <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={2}>{t('pms.epicGoal')}</Text>
                    <Text color="gray.800" whiteSpace="pre-wrap">{epic.goal}</Text>
                  </Box>
                )}
                {epic.goal && epic.description && <Divider />}
                {epic.description && (
                  <Box>
                    <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={2}>{t('pms.epicDescription')}</Text>
                    <Text color="gray.800" whiteSpace="pre-wrap">{epic.description}</Text>
                  </Box>
                )}
              </Stack>
            </CardBody>
          </Card>
        )}

        {/* Stories 区块 */}
        <Card bg="white" borderRadius="lg" border="1px solid" borderColor="myGray.200">
          <CardBody>
            <Flex justify="space-between" align="center" mb={4}>
              <Heading size="sm">
                {t('pms.epicStories')} ({stories.length})
              </Heading>
              {canEditScrum && (
                <Button size="sm" variant="outline" colorScheme="blue" leftIcon={<FiPlus />} onClick={handleOpenLinkDrawer}>
                  {t('pms.linkStory')}
                </Button>
              )}
            </Flex>

            {stories.length === 0 ? (
              <Text color="gray.500" py={6} textAlign="center">{t('pms.noEpicStories')}</Text>
            ) : (
              <Stack spacing={1}>
                {stories.map(story => (
                  <Flex
                    key={story.slug}
                    align="center"
                    py={2}
                    px={3}
                    borderRadius="md"
                    _hover={{ bg: 'gray.50' }}
                    justify="space-between"
                  >
                    <Link href={`/projects/${projectSlug}/story/${story.slug}`} style={{ textDecoration: 'none', flex: 1 }}>
                      <HStack align="center">
                        <Text color="blue.600" _hover={{ color: 'blue.800' }} fontWeight="medium">{story.title}</Text>
                        {story.storyPoints != null && story.storyPoints > 0 && (
                          <Badge colorScheme="purple" variant="subtle" fontSize="2xs">{story.storyPoints} pt</Badge>
                        )}
                      </HStack>
                    </Link>
                    <HStack spacing={2}>
                      <Badge
                        borderWidth="1px"
                        borderStyle="solid"
                        bg="transparent"
                        color={getStoryStatusColor(story.status)}
                        borderColor={getStoryStatusColor(story.status)}
                      >
                        {getStoryStatusName(story.status)}
                      </Badge>
                      {canEditScrum && (
                        <IconButton
                          aria-label={t('pms.removeStoryFromEpic')}
                          icon={<FiX />}
                          size="xs"
                          variant="ghost"
                          color="gray.400"
                          _hover={{ color: 'red.500' }}
                          onClick={() => handleUnlinkStory(story.slug)}
                        />
                      )}
                    </HStack>
                  </Flex>
                ))}
              </Stack>
            )}
          </CardBody>
        </Card>

        {/* 活动时间线 */}
        <Card bg="white" borderRadius="lg" border="1px solid" borderColor="myGray.200">
          <CardBody>
            <HStack mb={4}>
              <FiActivity size={16} />
              <Heading size="sm">{t('pms.activities')}</Heading>
            </HStack>
            <ActivityTimeline activities={activities} participants={participants} />
          </CardBody>
        </Card>
      </Stack>

      {/* 编辑 Drawer */}
      <EpicDrawer
        isOpen={isEditDrawerOpen}
        onClose={() => setIsEditDrawerOpen(false)}
        mode="edit"
        projectSlug={projectSlug}
        initialData={{
          title: epic.title,
          description: epic.description || '',
          goal: epic.goal || '',
          status: epic.status,
          priority: epic.priority,
          startDate: epic.startDate || '',
          targetDate: epic.targetDate || '',
          ownerName: epic.ownerName || '',
        }}
        onSave={handleSave}
        members={members}
      />

      <ConfirmDialog
        isOpen={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={handleConfirm}
        title={t('common.confirm')}
        message={t('pms.deleteEpicConfirm')}
      />

      {/* 关联 Story Drawer */}
      <Drawer isOpen={isLinkDrawerOpen} placement="right" onClose={() => setIsLinkDrawerOpen(false)} size="md" blockScrollOnMount={false}>
        <DrawerOverlay />
        <DrawerContent>
          <DrawerCloseButton />
          <DrawerHeader borderBottomWidth="1px">{t('pms.linkStory')}</DrawerHeader>
          <DrawerBody>
            {linkLoading ? (
              <Flex justify="center" align="center" h="200px"><Spinner size="lg" /></Flex>
            ) : (
              <VStack spacing={6} align="stretch">
                <Box>
                  <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={2}>
                    {t('pms.selectStoryToLink')}
                  </Text>
                  <Input
                    placeholder={t('pms.searchStoryPlaceholder')}
                    value={linkSearch}
                    onChange={(e) => setLinkSearch(e.target.value)}
                  />
                  {linkSearch.trim() && (
                    <Box mt={2} maxH="400px" overflowY="auto" borderWidth="1px" borderRadius="md" borderColor="gray.200">
                      {allStories
                        .filter((s) => {
                          // 已关联的不显示
                          if (stories.some(es => es.slug === s.slug)) return false
                          const q = linkSearch.toLowerCase().trim()
                          return s.title.toLowerCase().includes(q)
                        })
                        .slice(0, 30)
                        .map((s) => (
                          <Box
                            key={s.slug}
                            px={3}
                            py={2}
                            borderBottomWidth="1px"
                            borderColor="gray.100"
                            cursor="pointer"
                            _hover={{ bg: 'blue.50' }}
                            onClick={() => {
                              handleLinkStory(s.slug)
                              setLinkSearch('')
                            }}
                          >
                            <Flex justify="space-between" align="center">
                              <Box>
                                <Text fontSize="sm" fontWeight="medium">{s.title}</Text>
                                <Text fontSize="xs" color="gray.500">
                                  {s.priority} · {s.status}
                                  {s.storyPoints ? ` · ${s.storyPoints} pt` : ''}
                                </Text>
                              </Box>
                              <FiPlus size={14} color="#3182ce" />
                            </Flex>
                          </Box>
                        ))}
                      {allStories.filter((s) => {
                        if (stories.some(es => es.slug === s.slug)) return false
                        const q = linkSearch.toLowerCase().trim()
                        return s.title.toLowerCase().includes(q)
                      }).length === 0 && (
                        <Text color="gray.400" fontSize="sm" textAlign="center" py={4}>
                          {t('pms.noStoriesToLink')}
                        </Text>
                      )}
                    </Box>
                  )}
                </Box>
                <Divider />
                <Box>
                  <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={2}>
                    {t('pms.linkedStories')} ({stories.length})
                  </Text>
                  {stories.length > 0 ? (
                    <Stack spacing={2}>
                      {stories.map(s => (
                        <Box key={s.slug} p={3} bg="gray.50" borderRadius="md">
                          <Flex justify="space-between" align="center">
                            <Box>
                              <Text fontSize="sm" fontWeight="medium">{s.title}</Text>
                              <Text fontSize="xs" color="gray.500">{s.priority} · {s.status}</Text>
                            </Box>
                            <IconButton
                              aria-label={t('pms.unlinkStory')}
                              icon={<FiX />}
                              size="xs"
                              variant="ghost"
                              color="gray.400"
                              _hover={{ color: 'red.500' }}
                              onClick={() => handleUnlinkStory(s.slug)}
                            />
                          </Flex>
                        </Box>
                      ))}
                    </Stack>
                  ) : (
                    <Text color="gray.400" fontSize="sm" textAlign="center" py={4}>
                      {t('pms.noLinkedStories')}
                    </Text>
                  )}
                </Box>
              </VStack>
            )}
          </DrawerBody>
          <DrawerFooter borderTopWidth="1px">
            <Button variant="ghost" onClick={() => setIsLinkDrawerOpen(false)}>{t('common.close')}</Button>
          </DrawerFooter>
        </DrawerContent>
      </Drawer>
    </>
  )
}
