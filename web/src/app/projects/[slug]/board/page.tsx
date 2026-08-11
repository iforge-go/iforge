'use client'

import { useState, useEffect, useMemo } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import {
  Box,
  Button,
  Flex,
  Heading,
  HStack,
  Select,
  Spinner,
  Stack,
  Text,
} from '@chakra-ui/react'
import { FiRepeat, FiPlus } from 'react-icons/fi'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import { useWebSocket } from '@/contexts/WebSocketContext'
import KanbanBoard from './components/KanbanBoard'
import TaskDrawer from '@/components/TaskDrawer'
import SprintBreadcrumb from '@/components/SprintBreadcrumb'
import { useProject } from '../ProjectContext'
import type { TaskStatus } from '@/lib/types'

// "全部活跃 Sprint" 聚合视图的特殊 key（选择器用，非真实 sprint slug）
const ALL_ACTIVE_KEY = '__all_active__'

interface TaskItem {
  taskId: number
  title: string
  description: string | null
  status: string
  priority: string
  taskType: string
  storyPoints: number | null
  assigneeName: string | null
  assignees?: { userName: string; fullName?: string; image?: string | null }[]
  commentCount?: number
  position?: number
  sprintSlug: string | null
  epicSlug?: string | null
  epicTitle?: string | null
  userStorySlug?: string | null
  userStoryTitle?: string | null
  projectSlug?: string
}

// 活跃 Sprint 看板（对齐 Jira Board：显示活跃 Sprint 的任务）
// 多个活跃 Sprint 时支持"全部活跃 Sprint"聚合视图，合并展示所有任务并用 Badge 标注归属
export default function BoardPage() {
  const params = useParams()
  const projectSlug = params.slug as string
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()
  const { canEditScrum } = useProject()

  const [sprints, setSprints] = useState<any[]>([])
  const [activeSprintSlug, setActiveSprintSlug] = useState<string>('')
  const [tasks, setTasks] = useState<TaskItem[]>([])
  const [taskStatuses, setTaskStatuses] = useState<TaskStatus[]>([])
  const [loading, setLoading] = useState(true)

  // 任务详情抽屉
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null)
  const [drawerMode, setDrawerMode] = useState<'view' | 'edit'>('view')
  const [isNewTaskDrawerOpen, setIsNewTaskDrawerOpen] = useState(false)
  const [highlightedTaskId, setHighlightedTaskId] = useState<number | null>(null)

  const activeSprints = useMemo(
    () => sprints.filter((s) => s.status === 'active'),
    [sprints]
  )
  // 聚合视图下用：sprint slug -> title 映射，供任务卡片显示 Sprint Badge
  const sprintSlugToTitle = useMemo(() => {
    const map: Record<string, string> = {}
    for (const s of activeSprints) map[s.slug] = s.title
    return map
  }, [activeSprints])
  const isAllActiveView = activeSprintSlug === ALL_ACTIVE_KEY
  // 聚合视图下新建任务默认关联第一个活跃 Sprint（单 sprint 视图则用所选 sprint）
  const effectiveSprintSlug = isAllActiveView
    ? (activeSprints[0]?.slug || undefined)
    : (activeSprintSlug || undefined)

  // 实时协同：task_updated 事件同步
  useWebSocket('task_updated', (payload: any) => {
    if (!payload?.task) return
    const isSelf = payload.actor === user?.userName
    const updatedTask = payload.task
    setTasks((prev) => {
      const exists = prev.some((tk) => tk.taskId === updatedTask.taskId)
      if (!exists) return prev
      return prev.map((tk) => (tk.taskId === updatedTask.taskId ? { ...tk, ...updatedTask } : tk))
    })
    if (!isSelf) setHighlightedTaskId(updatedTask.taskId)
  })

  useEffect(() => {
    if (!user || !projectSlug) return
    const loadData = async () => {
      try {
        const [sprintsData, statuses] = await Promise.all([
          api.getSprints(projectSlug).catch(() => []),
          api.getTaskStatuses(projectSlug).catch(() => []),
        ])
        setSprints(sprintsData || [])
        setTaskStatuses(statuses || [])

        // 多个活跃 Sprint 默认进入聚合视图；单个则选中该 Sprint
        const active = (sprintsData || []).filter((s: any) => s.status === 'active')
        if (active.length >= 2) {
          setActiveSprintSlug(ALL_ACTIVE_KEY)
        } else if (active.length === 1) {
          setActiveSprintSlug(active[0].slug)
        }
      } catch (error) {
        console.error('Failed to load board data:', error)
      } finally {
        setLoading(false)
      }
    }
    loadData()
  }, [user, projectSlug])

  // 加载选中 Sprint 的任务（聚合视图并行请求所有活跃 Sprint 并合并）
  useEffect(() => {
    if (!user || !projectSlug || !activeSprintSlug) return
    if (isAllActiveView) {
      Promise.all(
        activeSprints.map((s) => api.getTasks(projectSlug, { sprintSlug: s.slug }).catch(() => [] as TaskItem[]))
      )
        .then((results) => setTasks(results.flat()))
        .catch(() => setTasks([]))
    } else {
      api.getTasks(projectSlug, { sprintSlug: activeSprintSlug })
        .then((data) => setTasks(data || []))
        .catch(() => setTasks([]))
    }
  }, [user, projectSlug, activeSprintSlug, isAllActiveView, activeSprints])

  // 高亮 5 秒后清除
  useEffect(() => {
    if (highlightedTaskId === null) return
    const timer = setTimeout(() => setHighlightedTaskId(null), 5000)
    return () => clearTimeout(timer)
  }, [highlightedTaskId])

  const currentSprint = isAllActiveView ? undefined : sprints.find((s) => s.slug === activeSprintSlug)

  function handleTaskClick(taskId: number, mode: 'view' | 'edit' = 'view') {
    setDrawerMode(mode)
    setSelectedTaskId(taskId)
  }

  function handleCloseDrawer() {
    setSelectedTaskId(null)
    setDrawerMode('view')
  }

  function handleTaskUpdated(updatedTask: any) {
    setTasks((prev) => prev.map((tk) => (tk.taskId === updatedTask.taskId ? { ...tk, ...updatedTask } : tk)))
    setHighlightedTaskId(updatedTask.taskId)
  }

  if (authLoading) return null

  if (!user) {
    return (
      <Box py={20}>
        <Text textAlign="center">{t('common.pleaseLogin')}</Text>
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

  // 无活跃 Sprint：空状态提示
  if (activeSprints.length === 0) {
    return (
      <Flex direction="column" align="center" justify="center" py={20} gap={4}>
        <FiRepeat size={48} color="#94a3b8" />
        <Text color="gray.500" fontSize="lg">{t('pms.noActiveSprintHint')}</Text>
        <Button colorScheme="blue" leftIcon={<FiRepeat />} as={Link} href={`/projects/${projectSlug}/backlog`}>
          {t('pms.backlog')}
        </Button>
      </Flex>
    )
  }

  return (
    <Stack spacing={4}>
      {/* Header：Sprint 标题 + Sprint 过滤器（样式对齐任务页） */}
      <Flex justify="space-between" align="center">
        <Box>
          {currentSprint ? (
            <SprintBreadcrumb
              sprint={currentSprint}
              href={`/projects/${projectSlug}/sprint/${currentSprint.slug}`}
            />
          ) : (
            <Flex align="center" gap={3}>
              <FiRepeat color="#3182ce" />
              <Heading size="md">{t('pms.allActiveSprints')}</Heading>
            </Flex>
          )}
        </Box>
        <HStack spacing={3}>
          {/* Sprint 过滤器 */}
          <Select
            size="sm"
            h="32px"
            w="auto"
            minW="140px"
            value={activeSprintSlug}
            onChange={(e) => setActiveSprintSlug(e.target.value)}
            borderWidth="1px"
            borderColor="gray.200"
            bg="white"
          >
            <option value={ALL_ACTIVE_KEY}>{t('pms.allActiveSprints')}</option>
            {activeSprints.map((s) => (
              <option key={s.slug} value={s.slug}>{s.title}</option>
            ))}
          </Select>
          {canEditScrum && (
            <Button
              size="sm"
              leftIcon={<FiPlus />}
              colorScheme="blue"
              onClick={() => setIsNewTaskDrawerOpen(true)}
            >
              {t('pms.newTask')}
            </Button>
          )}
        </HStack>
      </Flex>

      {/* KanbanBoard */}
      <KanbanBoard
        tasks={tasks}
        taskStatuses={taskStatuses}
        projectSlug={projectSlug}
        setTasks={setTasks as any}
        onTaskClick={handleTaskClick}
        sprintSlug={effectiveSprintSlug}
        canEditScrum={canEditScrum}
        highlightedTaskId={highlightedTaskId}
        sprintSlugToTitle={sprintSlugToTitle}
        showSprintBadge={isAllActiveView}
      />

      {/* 任务详情抽屉 */}
      <TaskDrawer
        isOpen={selectedTaskId !== null}
        taskId={selectedTaskId}
        onClose={handleCloseDrawer}
        projectSlug={projectSlug}
        taskStatuses={taskStatuses}
        canEditScrum={canEditScrum}
        onTaskUpdated={handleTaskUpdated}
        mode={drawerMode}
      />

      {/* 新建任务抽屉 */}
      <TaskDrawer
        isOpen={isNewTaskDrawerOpen}
        taskId={null}
        mode="create"
        onClose={() => setIsNewTaskDrawerOpen(false)}
        projectSlug={projectSlug}
        taskStatuses={taskStatuses}
        canEditScrum={canEditScrum}
        sprintSlug={effectiveSprintSlug}
        onTaskCreated={(task) => { setTasks((prev) => [task, ...prev]) }}
      />
    </Stack>
  )
}
