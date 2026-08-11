'use client'

import { useState, useEffect, useRef, useMemo } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import {
  Box,
  Button,
  Card,
  CardBody,
  Flex,
  Heading,
  Stack,
  Text,
  Badge,
  Divider,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  Drawer,
  DrawerOverlay,
  DrawerContent,
  DrawerCloseButton,
  DrawerHeader,
  DrawerBody,
  DrawerFooter,
  Input,
  useToast,
  IconButton,
  HStack,
  VStack,
  Spinner,
  Stat,
  StatLabel,
  StatNumber,
  StatGroup,
  Progress,
  Avatar,
  Tooltip,
} from '@chakra-ui/react'
import { FiCheckSquare, FiCalendar, FiTrendingDown, FiPlus, FiLink, FiX, FiBook, FiActivity, FiChevronDown, FiChevronRight, FiBarChart2, FiTarget } from 'react-icons/fi'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import type { SprintReport } from '@/lib/types'
import BurndownChart from '@/components/BurndownChart'
import SprintCalendar from '@/components/SprintCalendar'
import ActivityTimeline from '@/components/ActivityTimeline'
import TaskDrawer from '@/components/TaskDrawer'
import SprintDrawer, { SprintFormData } from '@/components/SprintDrawer'
import SprintHeader from './components/SprintHeader'
import SprintProgressCard from './components/SprintProgressCard'
import { SprintWorkflowProgress } from './components/SprintWorkflowProgress'
import { useProject } from '../../ProjectContext'
import { getStatusHexColor } from '@/lib/taskStatus'
import { getStoryStatusColor } from '@/lib/storyStatus'
import { ConfirmDialog } from '@/components/ConfirmDialog'

export default function SprintDetailPage() {
  const router = useRouter()
  const params = useParams()
  const projectSlug = params.slug as string
  const sprintSlug = params.sprintSlug as string
  const { t } = useI18n()
  const toast = useToast()
  const { user, authLoading } = useCurrentUser()
  const { canEditScrum } = useProject()
  const [sprint, setSprint] = useState<any>(null)
  const [sprintLoading, setSprintLoading] = useState(true)
  const [tasks, setTasks] = useState<any[]>([])
  const [allSprints, setAllSprints] = useState<any[]>([])
  const [tabIndex, setTabIndex] = useState(0)
  // Sprint 编辑 Drawer：复用共享 SprintDrawer 组件
  const [isSprintDrawerOpen, setIsSprintDrawerOpen] = useState(false)

  // 新建任务抽屉
  const [isNewTaskDrawerOpen, setIsNewTaskDrawerOpen] = useState(false)
  const [, setAllUserStories] = useState<any[]>([])
  const [taskStatuses, setTaskStatuses] = useState<any[]>([])

  // 任务查看抽屉（点击任务以抽屉模式显示详情）
  const [viewTaskId, setViewTaskId] = useState<number | null>(null)
  const [isViewDrawerOpen, setIsViewDrawerOpen] = useState(false)

  // 关联已存在任务抽屉
  const [isLinkTaskDrawerOpen, setIsLinkTaskDrawerOpen] = useState(false)
  const [allProjectTasks, setAllProjectTasks] = useState<any[]>([])
  const [linkSearch, setLinkSearch] = useState('')
  const [linkTaskLoading, setLinkTaskLoading] = useState(false)

  // Sprint stories
  const [sprintStories, setSprintStories] = useState<any[]>([])

  // Story 展开/折叠状态（默认全部展开，与需求池页面一致）
  const [expandedStories, setExpandedStories] = useState<Set<string>>(new Set())
  const expandInitialized = useRef(false)

  // Sprint activities（后端 sideload participants：批量预取操作者头像信息）
  const [sprintActivities, setSprintActivities] = useState<any[]>([])
  const [activityParticipants, setActivityParticipants] = useState<any[]>([])

  // Sprint Report（范围变更追踪 + 承诺 vs 实际，对齐 Jira Sprint Report）
  const [sprintReport, setSprintReport] = useState<SprintReport | null>(null)
  const [reportLoading, setReportLoading] = useState(false)
  const reportLoadedRef = useRef(false)

  const [confirmOpen, setConfirmOpen] = useState(false)
  const [confirmAction, setConfirmAction] = useState<(() => void) | null>(null)

  // 切换到"报告"tab 时懒加载 Sprint Report 数据
  const REPORT_TAB_INDEX = 2 // Stories(0) → Burndown(1) → Report(2) → Calendar(3) → Activities(4)
  useEffect(() => {
    if (tabIndex !== REPORT_TAB_INDEX || reportLoadedRef.current || !sprintSlug) return
    reportLoadedRef.current = true
    setReportLoading(true)
    api.getSprintReport(sprintSlug)
      .then(setSprintReport)
      .catch(() => setSprintReport(null))
      .finally(() => setReportLoading(false))
  }, [tabIndex, sprintSlug])

  useEffect(() => {
    if (!user || !sprintSlug) return
    Promise.all([
      api.getSprint(sprintSlug).then(setSprint).finally(() => setSprintLoading(false)),
      // 使用 getTasks (sideload 模式) 替代 getSprintTasks：后端在 ListTasks 中
      // 批量预取 assignee 用户信息，前端无需逐条请求 /users/:name。
      // 返回 tasks 已附带 assignees: {userName, fullName?, image?}[]
      api.getTasks(projectSlug, { sprintSlug }).then((data) => setTasks(data || [])).catch((err) => console.error('Failed to get sprint tasks:', err)),
      api.getSprints(projectSlug).then((data) => setAllSprints(data || [])).catch(() => {}),
      api.getUserStories(projectSlug).then((data) => setAllUserStories(data || [])).catch(() => {}),
      api.getTaskStatuses(projectSlug).then((data) => setTaskStatuses(data || [])).catch(() => {}),
      api.getSprintStories(sprintSlug).then((data) => {
        setSprintStories(data || [])
      }).catch((err) => console.error('Failed to get sprint stories:', err)),
      api.getActivities(projectSlug, 'sprint', sprintSlug).then((data) => {
        setSprintActivities(data?.activities || [])
        setActivityParticipants(data?.participants || [])
      }).catch(() => {
        setSprintActivities([])
        setActivityParticipants([])
      }),
    ])
  }, [user, sprintSlug, projectSlug])

  // sprintStories 首次加载后默认全部展开
  useEffect(() => {
    if (!expandInitialized.current && sprintStories.length > 0) {
      setExpandedStories(new Set(sprintStories.map(s => s.slug)))
      expandInitialized.current = true
    }
  }, [sprintStories])

  const toggleStoryExpand = (slug: string) => {
    setExpandedStories(prev => {
      const next = new Set(prev)
      if (next.has(slug)) next.delete(slug)
      else next.add(slug)
      return next
    })
  }

  // 未关联故事的任务（或故事不在本 Sprint 内的任务），单独展示在"独立任务"区域
  const independentTasks = useMemo(() => {
    const storySlugs = new Set(sprintStories.map(s => s.slug))
    return tasks.filter(tk => !tk.userStorySlug || !storySlugs.has(tk.userStorySlug))
  }, [tasks, sprintStories])

  const handleEditClick = () => {
    if (sprint) setIsSprintDrawerOpen(true)
  }

  // 保存编辑：作为 SprintDrawer 的 onSave 回调，由共享组件在关闭抽屉前调用
  const handleSprintSave = async (data: SprintFormData) => {
    if (!sprint) return
    try {
      const updated = await api.updateSprint(sprintSlug, {
        title: data.title,
        description: data.description || null,
        goal: data.goal || null,
        status: data.status,
        startDate: data.startDate || null,
        endDate: data.endDate || null,
      })
      setSprint(updated)
      toast({
        title: t('common.success'),
        description: t('pms.sprintUpdated'),
        status: 'success',
        duration: 2000,
      })
    } catch (error: any) {
      toast({
        title: t('common.error'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleDelete = () => {
    if (!sprint) return
    setConfirmAction(() => async () => {
      try {
        await api.deleteSprint(sprintSlug)
        router.push(`/projects/${projectSlug}`)
      } catch (error: any) {
        toast({
          title: t('common.error'),
          description: error.message,
          status: 'error',
          duration: 3000,
        })
      }
    })
    setConfirmOpen(true)
  }

  const handleConfirm = () => {
    confirmAction?.()
    setConfirmOpen(false)
    setConfirmAction(null)
  }

  // 开始迭代：将 sprint 状态改为 active
  const handleStart = async () => {
    if (!sprint) return
    try {
      const updated = await api.updateSprint(sprintSlug, { status: 'active' })
      setSprint(updated)
      toast({
        title: t('common.success'),
        description: t('pms.sprintStarted'),
        status: 'success',
        duration: 2000,
      })
    } catch (error: any) {
      toast({
        title: t('common.error'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
      throw error
    }
  }

  // 完成迭代：将 sprint 状态改为 closed（由 SprintHeader 的确认对话框触发）
  const handleComplete = async () => {
    if (!sprint) return
    try {
      const updated = await api.updateSprint(sprintSlug, { status: 'closed' })
      setSprint(updated)
      toast({
        title: t('common.success'),
        description: t('pms.sprintCompleted'),
        status: 'success',
        duration: 2000,
      })
    } catch (error: any) {
      toast({
        title: t('common.error'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
      throw error
    }
  }

  // 打开关联任务抽屉
  async function handleOpenLinkDrawer() {
    setIsLinkTaskDrawerOpen(true)
    setLinkSearch('')
    setLinkTaskLoading(true)
    try {
      const allTasks = await api.getTasks(projectSlug)
      setAllProjectTasks(allTasks)
    } catch {
      setAllProjectTasks([])
    } finally {
      setLinkTaskLoading(false)
    }
  }

  // 关联已存在任务
  async function handleLinkTask(taskId: number) {
    try {
      const res: any = await api.updateTask(projectSlug, taskId, { sprintSlug: sprintSlug })
      // updateTask 返回 { task, assignees }，合并后塞入列表以保持经办人头像能直接渲染
      const updated = res.task ? { ...res.task, assignees: res.assignees || [] } : res
      setTasks((prev) => [...prev, updated])
      toast({ title: t('common.success'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    }
  }

  // 取消关联任务
  async function handleUnlinkTask(taskId: number) {
    try {
      await api.updateTask(projectSlug, taskId, { sprintSlug: '' })
      setTasks((prev) => prev.filter((t) => t.taskId !== taskId))
      toast({ title: t('common.success'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    }
  }

  // 点击任务：以抽屉模式打开任务详情
  const handleTaskClick = (taskId: number) => {
    setViewTaskId(taskId)
    setIsViewDrawerOpen(true)
  }

  // 将 task status slug 翻译为本地化名称（使用 pms.taskStatus* 前缀）
  const getTaskStatusName = (slug: string): string => {
    const camelSlug = slug.split('_').map(part => part.charAt(0).toUpperCase() + part.slice(1)).join('')
    const key = `pms.taskStatus${camelSlug}`
    const translated = t(key)
    return translated === key ? slug : translated
  }

  // 将 story status slug 翻译为本地化名称（使用 pms.status* 前缀）
  const getStoryStatusName = (slug: string): string => {
    const camelSlug = slug.split('_').map(part => part.charAt(0).toUpperCase() + part.slice(1)).join('')
    const key = `pms.status${camelSlug}`
    const translated = t(key)
    return translated === key ? slug : translated
  }

  // 任务行渲染：与需求池页面 renderTaskRow 风格一致（扁平行，无边框，紧凑）
  const renderTaskRow = (task: any, index: number = 0) => (
    <Flex
      key={task.taskId || `task-${index}`}
      align="center"
      py={1}
      px={2}
      borderRadius="md"
      _hover={{ bg: 'gray.100' }}
    >
      <Flex align="center" flex={1} opacity={task.status === 'done' || task.status === 'closed' ? 0.5 : 1}>
        {/* 标题 + 故事点 */}
        <Flex
          align="center"
          flex={1}
          cursor="pointer"
          onClick={() => handleTaskClick(task.taskId)}
        >
          <Text fontSize="sm" color="gray.700" _hover={{ color: 'blue.600' }}>{task.title}</Text>
          {task.storyPoints > 0 && (
            <Badge ml={2} colorScheme="purple" variant="subtle" fontSize="2xs">
              {task.storyPoints} pt
            </Badge>
          )}
        </Flex>
        {/* 右侧：状态圆点 + 经办人 + 取消关联 */}
        <HStack spacing={2}>
          <HStack spacing={1}>
            <Box w="7px" h="7px" borderRadius="full" bg={getStatusHexColor(task.status, taskStatuses)} flexShrink={0} />
            <Text fontSize="2xs" color="gray.600">{getTaskStatusName(task.status || 'todo')}</Text>
          </HStack>
          {task.assignees && task.assignees.length > 0 && (
            <HStack spacing="6px">
              {task.assignees.slice(0, 3).map((a: any, i: number) => (
                <Tooltip key={a.userName + i} label={a.fullName || a.userName} fontSize="xs">
                  <Avatar
                    size="xs"
                    name={a.fullName || a.userName}
                    src={a.image || undefined}
                    fontSize="9px"
                    ml={i > 0 ? '-6px' : '0'}
                    borderWidth="1px"
                    borderColor="white"
                  />
                </Tooltip>
              ))}
            </HStack>
          )}
          {canEditScrum && (
            <IconButton
              aria-label={t('pms.unlinkTask')}
              icon={<FiX />}
              size="xs"
              variant="ghost"
              color="gray.400"
              _hover={{ color: 'red.500' }}
              onClick={() => handleUnlinkTask(task.taskId)}
            />
          )}
        </HStack>
      </Flex>
    </Flex>
  )

  // 生成燃尽图数据
  // 算法：用 task.closedAt 按日聚合已完成 story points，累积得到每日剩余量。
  // 参考 laravel-gitscrum Helper::burndown 用 Issue.closed_at 按日统计的成熟算法。
  // closedAt 由后端 task_service.UpdateStatus 在状态变为 done/closed/completed/archived 时自动维护。
  const generateBurndownData = () => {
    if (!sprint || !sprint.startDate || !sprint.endDate || tasks.length === 0) return []

    const startDate = new Date(sprint.startDate)
    startDate.setHours(0, 0, 0, 0)
    const endDate = new Date(sprint.endDate)
    endDate.setHours(0, 0, 0, 0)
    const today = new Date()
    today.setHours(0, 0, 0, 0)

    const totalDays = Math.max(1, Math.ceil((endDate.getTime() - startDate.getTime()) / (1000 * 60 * 60 * 24)))
    const totalPoints = tasks.reduce((sum, t) => sum + (t.storyPoints || 0), 0)

    // 按"日"桶聚合已完成 story points：closedAt 落在某天 → 该日完成 (storyPoints) 点
    const closedPointsByDate = new Map<string, number>()
    for (const t of tasks) {
      if (!t.closedAt) continue
      const d = new Date(t.closedAt)
      d.setHours(0, 0, 0, 0)
      const key = d.toISOString().split('T')[0]
      closedPointsByDate.set(key, (closedPointsByDate.get(key) || 0) + (t.storyPoints || 0))
    }

    // 实际线只画到 min(today, endDate)，之后不画（BurndownChart 会跳过 null）
    const actualEndDate = today < endDate ? today : endDate

    // 初始累积：统计 Sprint 开始之前已关闭的 task（数据容错，避免遗漏）
    let cumulativeClosed = 0
    for (const t of tasks) {
      if (!t.closedAt) continue
      const d = new Date(t.closedAt)
      d.setHours(0, 0, 0, 0)
      if (d < startDate) cumulativeClosed += (t.storyPoints || 0)
    }

    const data = []
    for (let i = 0; i <= totalDays; i++) {
      const date = new Date(startDate)
      date.setDate(date.getDate() + i)
      date.setHours(0, 0, 0, 0)
      const key = date.toISOString().split('T')[0]

      // 理想线：从 totalPoints 线性下降到 0
      const expectedPoints = Math.max(0, totalPoints * (1 - i / totalDays))

      // 实际线：累积当日及之前完成的 points，剩余 = totalPoints - 累积完成（阶梯式下降）
      let actualPoints: number | null = null
      if (date <= actualEndDate) {
        cumulativeClosed += closedPointsByDate.get(key) || 0
        actualPoints = Math.max(0, totalPoints - cumulativeClosed)
      }

      data.push({
        date: key,
        expectedPoints,
        actualPoints,
        isToday: date.getTime() === today.getTime(),
      })
    }

    return data
  }

  if (authLoading) return null

  if (!user) {
    return (
      <Box py={10}>
        <Stack spacing={6} textAlign="center">
          <Heading size="lg">{t('common.pleaseLogin')}</Heading>
          <Button colorScheme="blue" onClick={() => router.push('/login')}>
            {t('auth.login')}
          </Button>
        </Stack>
      </Box>
    )
  }

  if (sprintLoading) {
    return (
      <Box py={10}>
        <Stack spacing={4}>
          <Text color="gray.600">{t('common.loading')}</Text>
        </Stack>
      </Box>
    )
  }

  if (!sprint) {
    return (
      <Box py={10}>
        <Stack spacing={6} textAlign="center">
          <Heading size="lg">{t('common.noData')}</Heading>
          <Button colorScheme="blue" onClick={() => router.push(`/projects/${projectSlug}`)}>
            {t('common.back')}
          </Button>
        </Stack>
      </Box>
    )
  }

  return (
    <>
      <Stack spacing={6}>
        <SprintHeader
          sprint={sprint}
          tasks={tasks}
          projectSlug={projectSlug}
          taskStatuses={taskStatuses}
          onEditClick={handleEditClick}
          onDelete={handleDelete}
          onStart={handleStart}
          onComplete={handleComplete}
          canEditScrum={canEditScrum}
        />

        {/* Sprint 进度概览：环形图 + 工作量统计（task 维度） */}
        <SprintProgressCard tasks={tasks} />

        {/* 研发流程进度条（story 维度聚合）：展示 Sprint 内所有 story 在 6 个流程节点的进展 */}
        <SprintWorkflowProgress stories={sprintStories} tasks={tasks} />

        {/* 详情标签页 */}
        <Card bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1">
          <CardBody p={0}>
            <Tabs variant="enclosed" index={tabIndex} onChange={(index) => setTabIndex(index)}>
              <TabList>
                <Tab>
                  <FiBook size={16} style={{ marginRight: 8 }} />
                  {t('pms.stories')} ({sprintStories.length})
                </Tab>
                <Tab>
                  <FiTrendingDown size={16} style={{ marginRight: 8 }} />
                  {t('pms.burndownChart')}
                </Tab>
                <Tab>
                  <FiBarChart2 size={16} style={{ marginRight: 8 }} />
                  {t('pms.sprintReport')}
                </Tab>
                <Tab>
                  <FiCalendar size={16} style={{ marginRight: 8 }} />
                  {t('pms.calendar')}
                </Tab>
                <Tab>
                  <FiActivity size={16} style={{ marginRight: 8 }} />
                  {t('pms.activities')}
                </Tab>
              </TabList>
              <TabPanels>
                {/* Sprint 故事（含嵌套任务） */}
                <TabPanel p={4}>
                  {/* 头部：统计摘要 + 操作按钮 */}
                  <Flex justify="space-between" align="center" mb={4}>
                    <Text fontSize="sm" color="gray.600">
                      {t('pms.stories')} ({sprintStories.length}) · {t('pms.tasks')} ({tasks.length})
                    </Text>
                    <HStack spacing={2}>
                      {canEditScrum && (
                        <>
                          <Button
                            size="sm"
                            variant="outline"
                            colorScheme="blue"
                            leftIcon={<FiLink />}
                            onClick={handleOpenLinkDrawer}
                          >
                            {t('pms.linkTask')}
                          </Button>
                          <Button
                            size="sm"
                            colorScheme="blue"
                            leftIcon={<FiPlus />}
                            onClick={() => setIsNewTaskDrawerOpen(true)}
                          >
                            {t('pms.newTask')}
                          </Button>
                        </>
                      )}
                    </HStack>
                  </Flex>

                  {/* Sprint 容量统计 */}
                  {sprintStories.length > 0 && (() => {
                    const totalPoints = sprintStories.reduce((sum: number, s: any) => sum + (s.storyPoints || 0), 0)
                    const completedPoints = sprintStories
                      .filter((s: any) => s.status === 'done' || s.status === 'completed')
                      .reduce((sum: number, s: any) => sum + (s.storyPoints || 0), 0)
                    const progressPercent = totalPoints > 0 ? Math.round((completedPoints / totalPoints) * 100) : 0
                    return (
                      <Box mb={4} borderWidth="1px" borderRadius="lg" p={3} bg="gray.50" borderColor="gray.200">
                        <StatGroup mb={2}>
                          <Stat>
                            <StatLabel color="gray.500" fontSize="xs">{t('pms.sprintStories')}</StatLabel>
                            <StatNumber fontSize="lg">{sprintStories.length}</StatNumber>
                          </Stat>
                          <Stat>
                            <StatLabel color="gray.500" fontSize="xs">{t('pms.totalStoryPoints')}</StatLabel>
                            <StatNumber fontSize="lg" color="purple.500">{totalPoints}</StatNumber>
                          </Stat>
                          <Stat>
                            <StatLabel color="gray.500" fontSize="xs">{t('pms.completedPoints')}</StatLabel>
                            <StatNumber fontSize="lg" color="green.500">{completedPoints}</StatNumber>
                          </Stat>
                          <Stat>
                            <StatLabel color="gray.500" fontSize="xs">{t('pms.progress')}</StatLabel>
                            <StatNumber fontSize="lg">{progressPercent}%</StatNumber>
                          </Stat>
                        </StatGroup>
                        <Progress value={progressPercent} size="sm" colorScheme="green" borderRadius="full" />
                      </Box>
                    )
                  })()}

                  {/* 故事列表（统一容器包裹，与需求池页面风格一致） */}
                  {sprintStories.length === 0 && independentTasks.length === 0 ? (
                    <Text color="gray.500">{t('pms.noSprintStories')}</Text>
                  ) : (
                    <Box borderRadius="lg" border="1px solid #DFE2EA" bg="white" overflow="hidden">
                      {sprintStories.map((story: any, idx: number) => {
                        const storyTasks = tasks.filter((tk: any) => tk.userStorySlug === story.slug) || []
                        const isExpanded = expandedStories.has(story.slug)
                        const isLastStory = idx === sprintStories.length - 1
                        return (
                          <Box
                            key={story.slug}
                            borderBottom={(!isLastStory || independentTasks.length > 0) ? '1px solid #F4F6F8' : 'none'}
                          >
                            {/* Story 行（扁平行，点击展开/折叠） */}
                            <Flex
                              px={3}
                              py={2}
                              bg="white"
                              cursor="pointer"
                              _hover={{ bg: 'gray.50' }}
                              justify="space-between"
                              align="center"
                              onClick={() => toggleStoryExpand(story.slug)}
                            >
                              <Flex align="center" flex={1}>
                                <Box mr={2} color="gray.400" flexShrink={0}>
                                  {isExpanded ? <FiChevronDown size={14} /> : <FiChevronRight size={14} />}
                                </Box>
                                <Link href={`/projects/${projectSlug}/story/${story.slug}`} style={{ textDecoration: 'none', flex: 1 }} onClick={(e) => e.stopPropagation()}>
                                  <Box cursor="pointer">
                                    <Text fontWeight="medium">{story.title}</Text>
                                    <Flex gap={4} mt={1} align="center">
                                      {story.storyPoints != null && story.storyPoints > 0 && (
                                        <Badge colorScheme="purple" variant="subtle" fontSize="xs">
                                          {story.storyPoints} pt
                                        </Badge>
                                      )}
                                      {storyTasks.length > 0 && (
                                        <Text fontSize="sm" color="gray.500">
                                          <FiCheckSquare size={12} style={{ marginRight: 4, display: 'inline' }} />{storyTasks.length}
                                        </Text>
                                      )}
                                    </Flex>
                                  </Box>
                                </Link>
                              </Flex>
                              <Badge borderWidth="1px" borderStyle="solid" bg="transparent" color={getStoryStatusColor(story.status)} borderColor={getStoryStatusColor(story.status)}>
                                {getStoryStatusName(story.status)}
                              </Badge>
                            </Flex>
                            {/* 嵌套任务列表（展开时显示，缩进 + 浅灰底） */}
                            {isExpanded && storyTasks.length > 0 && (
                              <Box pl={8} pr={3} pb={2} bg="gray.50">
                                <Stack spacing={1}>
                                  {storyTasks.map((task: any, i: number) => renderTaskRow(task, i))}
                                </Stack>
                              </Box>
                            )}
                          </Box>
                        )
                      })}

                      {/* 独立任务（未关联故事的任务） */}
                      {independentTasks.length > 0 && (
                        <Box>
                          {sprintStories.length > 0 && (
                            <Flex px={3} py={1} bg="gray.50" borderTop="1px solid #F4F6F8">
                              <Text fontSize="sm" fontWeight="medium" color="gray.600">
                                {t('pms.independentTasks')} ({independentTasks.length})
                              </Text>
                            </Flex>
                          )}
                          {sprintStories.length === 0 && (
                            <Flex px={3} py={2} bg="white" borderBottom="1px solid #F4F6F8">
                              <Text fontSize="sm" fontWeight="medium" color="gray.600">
                                {t('pms.independentTasks')} ({independentTasks.length})
                              </Text>
                            </Flex>
                          )}
                          <Box pl={3} pr={3} pb={2} pt={1} bg="gray.50">
                            <Stack spacing={1}>
                              {independentTasks.map((task: any, i: number) => renderTaskRow(task, i))}
                            </Stack>
                          </Box>
                        </Box>
                      )}
                    </Box>
                  )}
                </TabPanel>

                {/* 燃尽图 */}
                <TabPanel p={4}>
                  <BurndownChart data={generateBurndownData()} />
                </TabPanel>

                {/* Sprint 报告：范围变更追踪 + 承诺 vs 实际（对齐 Jira Sprint Report） */}
                <TabPanel p={4}>
                  {/* Sprint 目标：复盘对照锚点，先于报告数据展示 */}
                  {sprint?.goal && (
                    <Box borderWidth="1px" borderRadius="lg" p={4} bg="blue.50" borderColor="blue.200" mb={4}>
                      <HStack spacing={2} mb={2}>
                        <FiTarget size={14} color="#3182ce" />
                        <Text fontSize="sm" fontWeight="600" color="gray.700">{t('pms.sprintGoal')}</Text>
                      </HStack>
                      <Text fontSize="sm" color="gray.700" whiteSpace="pre-wrap">{sprint.goal}</Text>
                    </Box>
                  )}
                  {reportLoading ? (
                    <Flex justify="center" py={8}><Spinner /></Flex>
                  ) : sprintReport ? (
                    <Stack spacing={4}>
                      {/* 承诺 vs 实际统计 */}
                      <Box borderWidth="1px" borderRadius="lg" p={4} bg="gray.50" borderColor="gray.200">
                        <Text fontSize="sm" fontWeight="600" color="gray.700" mb={3}>
                          {t('pms.committedVsCompleted')}
                        </Text>
                        <StatGroup mb={3}>
                          <Stat>
                            <StatLabel color="gray.500" fontSize="xs">{t('pms.committedPoints')}</StatLabel>
                            <StatNumber fontSize="lg" color="gray.700">{sprintReport.committedPoints}</StatNumber>
                          </Stat>
                          <Stat>
                            <StatLabel color="gray.500" fontSize="xs">{t('pms.completedPoints')}</StatLabel>
                            <StatNumber fontSize="lg" color="green.500">{sprintReport.completedPoints}</StatNumber>
                          </Stat>
                        </StatGroup>
                        {sprintReport.committedPoints > 0 && (
                          <Progress value={Math.round((sprintReport.completedPoints / sprintReport.committedPoints) * 100)} size="sm" colorScheme="green" borderRadius="full" />
                        )}
                      </Box>

                      {/* 范围变更时间线 */}
                      <Box>
                        <Text fontSize="sm" fontWeight="600" color="gray.700" mb={3}>
                          {t('pms.scopeChanges')} ({sprintReport.scopeChanges.length})
                        </Text>
                        {sprintReport.scopeChanges.length === 0 ? (
                          <Text color="gray.500" fontSize="sm">{t('pms.noScopeChanges')}</Text>
                        ) : (
                          <Stack spacing={2}>
                            {sprintReport.scopeChanges.map((change, idx) => (
                              <Flex
                                key={idx}
                                align="center"
                                gap={3}
                                p={2}
                                borderRadius="md"
                                bg={change.action === 'sprint_scope_added' ? 'green.50' : 'red.50'}
                              >
                                <Badge
                                  colorScheme={change.action === 'sprint_scope_added' ? 'green' : 'red'}
                                  fontSize="xs"
                                >
                                  {change.action === 'sprint_scope_added' ? t('pms.scopeAdded') : t('pms.scopeRemoved')}
                                </Badge>
                                <Text fontSize="sm" fontWeight="medium">{change.itemTitle}</Text>
                                <Badge variant="subtle" fontSize="xs">
                                  {change.itemType === 'story' ? t('pms.stories') : t('pms.tasks')}
                                </Badge>
                                <Text fontSize="xs" color="gray.500" ml="auto">
                                  {change.userName} · {new Date(change.createdAt).toLocaleString()}
                                </Text>
                              </Flex>
                            ))}
                          </Stack>
                        )}
                      </Box>
                    </Stack>
                  ) : (
                    <Text color="gray.500">{t('common.error')}</Text>
                  )}
                </TabPanel>

                {/* 日历视图 */}
                <TabPanel p={4}>
                  <SprintCalendar sprints={allSprints} />
                </TabPanel>

                {/* 活动时间线 */}
                <TabPanel p={4}>
                  <ActivityTimeline activities={sprintActivities} participants={activityParticipants} />
                </TabPanel>
              </TabPanels>
            </Tabs>
          </CardBody>
        </Card>
      </Stack>

      {/* Sprint 编辑 Drawer：复用共享 SprintDrawer 组件 */}
      <SprintDrawer
        isOpen={isSprintDrawerOpen}
        onClose={() => setIsSprintDrawerOpen(false)}
        mode="edit"
        projectSlug={projectSlug}
        initialData={sprint ? {
          title: sprint.title,
          description: sprint.description || '',
          goal: sprint.goal || '',
          status: sprint.status,
          startDate: sprint.startDate || '',
          endDate: sprint.endDate || '',
        } : undefined}
        onSave={handleSprintSave}
      />

      {/* 新建任务抽屉 - unified TaskDrawer in create mode */}
      <TaskDrawer
        isOpen={isNewTaskDrawerOpen}
        taskId={null}
        mode="create"
        onClose={() => setIsNewTaskDrawerOpen(false)}
        projectSlug={projectSlug}
        taskStatuses={taskStatuses}
        canEditScrum={canEditScrum}
        sprintSlug={sprintSlug}
        onTaskCreated={(task) => { setTasks((prev) => [task, ...prev]) }}
      />

      {/* 任务查看抽屉 - 点击任务以抽屉模式显示详情 */}
      <TaskDrawer
        isOpen={isViewDrawerOpen}
        taskId={viewTaskId}
        mode="view"
        onClose={() => { setIsViewDrawerOpen(false); setViewTaskId(null) }}
        projectSlug={projectSlug}
        taskStatuses={taskStatuses}
        canEditScrum={canEditScrum}
        onTaskUpdated={(updatedTask) => {
          setTasks((prev) => prev.map((tk) => tk.taskId === updatedTask.taskId ? { ...tk, ...updatedTask } : tk))
        }}
      />

      <ConfirmDialog
        isOpen={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={handleConfirm}
        title={t('common.confirm')}
        message={t('pms.deleteSprintConfirm')}
      />

      {/* 关联已存在任务抽屉 */}
      <Drawer isOpen={isLinkTaskDrawerOpen} placement="right" onClose={() => setIsLinkTaskDrawerOpen(false)} size="md" blockScrollOnMount={false}>
        <DrawerOverlay />
        <DrawerContent>
          <DrawerCloseButton />
          <DrawerHeader borderBottomWidth="1px">{t('pms.linkTask')}</DrawerHeader>

          <DrawerBody>
            {linkTaskLoading ? (
              <Flex justify="center" align="center" h="200px">
                <Spinner size="lg" />
              </Flex>
            ) : (
              <VStack spacing={6} align="stretch">
                {/* 搜索选择未关联任务 */}
                <Box>
                  <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={2}>
                    {t('pms.selectTaskToLink')}
                  </Text>
                  <Input
                    placeholder={t('pms.searchTaskPlaceholder')}
                    value={linkSearch}
                    onChange={(e) => setLinkSearch(e.target.value)}
                  />
                  {linkSearch.trim() && (
                    <Box mt={2} maxH="240px" overflowY="auto" borderWidth="1px" borderRadius="md" borderColor="gray.200">
                      {allProjectTasks
                        .filter((task) => {
                          const isLinked = tasks.some((t) => t.taskId === task.taskId)
                          if (isLinked) return false
                          const q = linkSearch.toLowerCase().trim()
                          return task.title.toLowerCase().includes(q) || String(task.taskId).includes(q)
                        })
                        .slice(0, 20)
                        .map((task) => (
                          <Box
                            key={task.taskId}
                            px={3}
                            py={2}
                            borderBottomWidth="1px"
                            borderColor="gray.100"
                            cursor="pointer"
                            _hover={{ bg: 'blue.50' }}
                            onClick={() => {
                              handleLinkTask(task.taskId)
                              setLinkSearch('')
                            }}
                          >
                            <Flex justify="space-between" align="center">
                              <Box>
                                <Text fontSize="sm" fontWeight="medium">TASK-{task.taskId} {task.title}</Text>
                                <Text fontSize="xs" color="gray.500">
                                  {task.priority} · {task.status}
                                </Text>
                              </Box>
                              <FiPlus size={14} color="#3182ce" />
                            </Flex>
                          </Box>
                        ))}
                      {allProjectTasks.filter((task) => {
                        const isLinked = tasks.some((t) => t.taskId === task.taskId)
                        if (isLinked) return false
                        const q = linkSearch.toLowerCase().trim()
                        return task.title.toLowerCase().includes(q) || String(task.taskId).includes(q)
                      }).length === 0 && (
                        <Text color="gray.400" fontSize="sm" textAlign="center" py={4}>
                          {t('pms.noTasksToLink')}
                        </Text>
                      )}
                    </Box>
                  )}
                </Box>

                <Divider />

                {/* 已关联任务列表 */}
                <Box>
                  <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={2}>
                    {t('pms.linkedTasks')} ({tasks.length})
                  </Text>
                  {tasks.length > 0 ? (
                    <Stack spacing={2}>
                      {tasks.map((task: any) => (
                        <Box
                          key={task.taskId}
                          p={3}
                          bg="gray.50"
                          borderRadius="md"
                        >
                          <Flex justify="space-between" align="center">
                            <Box>
                              <Text fontSize="sm" fontWeight="medium">TASK-{task.taskId} {task.title}</Text>
                              <HStack spacing={4} mt={1}>
                                <Text fontSize="xs" color="gray.500">{task.priority}</Text>
                                <Text fontSize="xs" color="gray.500">{task.status}</Text>
                              </HStack>
                            </Box>
                            <IconButton
                              aria-label={t('pms.unlinkTask')}
                              icon={<FiX />}
                              size="xs"
                              variant="ghost"
                              color="gray.400"
                              _hover={{ color: 'red.500' }}
                              onClick={() => handleUnlinkTask(task.taskId)}
                            />
                          </Flex>
                        </Box>
                      ))}
                    </Stack>
                  ) : (
                    <Text color="gray.400" fontSize="sm" textAlign="center" py={4}>
                      {t('pms.noLinkedTasks')}
                    </Text>
                  )}
                </Box>
              </VStack>
            )}
          </DrawerBody>

          <DrawerFooter borderTopWidth="1px">
            <Button variant="ghost" onClick={() => setIsLinkTaskDrawerOpen(false)}>
              {t('common.close')}
            </Button>
          </DrawerFooter>
        </DrawerContent>
      </Drawer>

    </>
  )
}
