'use client'

import {
  Box,
  Heading,
  Text,
  Button,
  VStack,
  HStack,
  Badge,
  Spinner,
  Flex,
  Icon,
  Stack,
  Select,
  IconButton,
  Input,
  InputGroup,
  InputLeftElement,
  useToast,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
} from '@chakra-ui/react'
import { keyframes } from '@emotion/react'
import { FiPlus, FiCheckSquare, FiRepeat, FiEdit, FiTrash2, FiSearch, FiChevronLeft, FiChevronRight } from 'react-icons/fi'
import { useState, useEffect, useRef, Suspense } from 'react'
import { useRouter, useParams, useSearchParams } from 'next/navigation'
import { api } from '@/lib/api'
import { useCurrentUser } from '@/contexts/UserContext'
import { useI18n } from '@/contexts/I18nContext'
import { useWebSocket } from '@/contexts/WebSocketContext'
import TaskDrawer from '@/components/TaskDrawer'
import SprintBreadcrumb from '@/components/SprintBreadcrumb'
import { useProject } from '../ProjectContext'
import { getPriorityBorderColor } from '@/lib/taskPriority'
import { getTaskTypeIcon } from '@/lib/taskType'
import type { TaskStatus } from '@/lib/types'

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
}

// 任务编辑后高亮动画：黄色闪烁后渐变为淡蓝色背景，持续数秒后淡出（与需求池用户故事高亮一致）
const highlightPulse = keyframes`
  0% { background-color: #fef3c7; box-shadow: inset 4px 0 0 #f59e0b; }
  30% { background-color: #fef3c7; box-shadow: inset 4px 0 0 #f59e0b; }
  100% { background-color: #ebf8ff; box-shadow: inset 4px 0 0 #3182ce; }
`

function TasksContent() {
  const router = useRouter()
  const params = useParams()
  const searchParams = useSearchParams()
  const projectSlug = params.slug as string
  const { t } = useI18n()
  const toast = useToast()
  const { user, authLoading } = useCurrentUser()
  const { canEditScrum } = useProject()
  // 后端分页：tasks 仅为当前页数据，total 为满足过滤条件的总数（后端返回，前端据此计算总页数）
  const [tasks, setTasks] = useState<TaskItem[]>([])
  const [total, setTotal] = useState(0)
  const [tasksLoading, setTasksLoading] = useState(true)
  const [sprints, setSprints] = useState<any[]>([])
  const [taskStatuses, setTaskStatuses] = useState<TaskStatus[]>([])
  const [sprintInfo, setSprintInfo] = useState<any>(null)

  // Task detail drawer
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null)
  // 抽屉打开模式：'view' 查看（默认） / 'edit' 直接编辑（点击编辑按钮触发，保存后关闭）
  const [drawerMode, setDrawerMode] = useState<'view' | 'edit'>('view')

  // New task drawer
  const [isNewTaskDrawerOpen, setIsNewTaskDrawerOpen] = useState(false)

  // 任务编辑后高亮：记录刚编辑任务的 ID，在列表/看板中高亮显示
  const [highlightedTaskId, setHighlightedTaskId] = useState<number | null>(null)

  // 删除任务确认对话框
  const [taskToDelete, setTaskToDelete] = useState<number | null>(null)
  const [deleting, setDeleting] = useState(false)
  const cancelRef = useRef(null as any)

  // 任务搜索：后端分页后搜索也走后端（避免只搜当前页 20 条）。输入防抖 300ms，
  // 防抖结束时同时应用搜索词和重置到第 1 页（同一批 state 更新，避免重复请求）。
  const [searchQuery, setSearchQuery] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')

  // 分页：每页 20 条（任务列表页后端分页，看板页全部展示不分页）
  const PAGE_SIZE = 20
  const [currentPage, setCurrentPage] = useState(1)

  // 高亮 5 秒后自动清除
  useEffect(() => {
    if (highlightedTaskId === null) return
    const timer = setTimeout(() => setHighlightedTaskId(null), 5000)
    return () => clearTimeout(timer)
  }, [highlightedTaskId])

  // 实时协同：收到 task_updated 事件时同步当前页 state。
  // 不跳过 actor 自身：通过 API/CLI/其他标签页修改任务时，自己的 Web UI 也需要实时同步。
  // 但跳过自身事件的高亮动画（避免 UI 拖拽时闪烁）。
  useWebSocket('task_updated', (payload: any) => {
    if (!payload?.task) return
    const isSelf = payload.actor === user?.userName
    const updatedTask = payload.task
    setTasks((prev) => {
      const exists = prev.some((t) => t.taskId === updatedTask.taskId)
      if (!exists) return prev // 不在当前页则忽略
      return prev.map((t) => (t.taskId === updatedTask.taskId ? { ...t, ...updatedTask } : t))
    })
    if (!isSelf) {
      setHighlightedTaskId(updatedTask.taskId)
    }
  })

  const sprintSlug = searchParams.get('sprint')

  // 搜索防抖：300ms 后应用搜索词并重置到第 1 页
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(searchQuery.trim())
      setCurrentPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [searchQuery])

  // 深链：URL ?task=<id> 时自动打开任务查看抽屉（用于通知 toast 点击跳转）。
  // 后端分页后任务可能不在当前页，抽屉按 ID 自取详情，故不再校验列表成员。
  useEffect(() => {
    const taskParam = searchParams.get('task')
    if (!taskParam) return
    const taskId = parseInt(taskParam, 10)
    if (isNaN(taskId)) return
    setSelectedTaskId(taskId)
    setDrawerMode('view')
  }, [searchParams])

  // Sprint 信息（仅 sprint 上下文用于面包屑；独立 effect，避免翻页时重复请求）
  useEffect(() => {
    if (!sprintSlug) {
      setSprintInfo(null)
      return
    }
    api.getSprint(sprintSlug).then(setSprintInfo).catch(() => {})
  }, [sprintSlug])

  const changeSprint = (newSprintSlug: string) => {
    setCurrentPage(1)
    if (newSprintSlug) {
      router.push(`/projects/${projectSlug}/tasks?sprint=${newSprintSlug}`)
    } else {
      router.push(`/projects/${projectSlug}/tasks`)
    }
  }

  // --- 加载任务状态 + Sprint 列表（与分页无关，仅在 user/project/canEditScrum 变化时跑一次）---
  useEffect(() => {
    if (!user || !projectSlug) return
    let cancelled = false
    const loadStatuses = async () => {
      try {
        const statuses = await api.getTaskStatuses(projectSlug) || []
        if (cancelled) return
        setTaskStatuses(statuses)

        // 如果没有状态，初始化默认状态（仅 member+ 可写；viewer 跳过自动创建，等管理员首次进入时创建）
        if (statuses.length === 0 && canEditScrum) {
          const defaultStatuses = [
            { name: 'To Do', slug: 'todo', color: '#daa724', position: 0, isClosed: false },
            { name: 'In Progress', slug: 'in_progress', color: '#079a0d', position: 1, isClosed: false },
            { name: 'Review', slug: 'review', color: '#9333ea', position: 2, isClosed: false },
            { name: 'Done', slug: 'done', color: '#8c023f', position: 3, isClosed: true },
          ]

          for (const status of defaultStatuses) {
            try {
              await api.createTaskStatus(projectSlug, status)
            } catch (error: any) {
              if (!error.message?.includes('already exists')) {
                throw error
              }
            }
          }

          const newStatuses = await api.getTaskStatuses(projectSlug)
          if (!cancelled) setTaskStatuses(newStatuses || [])
        }
      } catch (error) {
        console.error('Failed to load task statuses:', error)
      }
    }
    loadStatuses()
    api.getSprints(projectSlug).then((data) => { if (!cancelled) setSprints(data || []) }).catch(() => {})
    return () => { cancelled = true }
  }, [user, projectSlug, canEditScrum])

  // --- 加载任务（后端分页）：每次翻页 / 搜索 / Sprint 切换都重新请求后端，数据库只返回当前页 ---
  useEffect(() => {
    if (!user || !projectSlug) return
    let cancelled = false
    const loadTasks = async () => {
      setTasksLoading(true)
      try {
        const offset = (currentPage - 1) * PAGE_SIZE
        const res = await api.getTasksPaged(projectSlug, {
          ...(sprintSlug ? { sprintSlug } : {}),
          ...(debouncedSearch ? { q: debouncedSearch } : {}),
          limit: PAGE_SIZE,
          offset,
        }).catch(() => ({ tasks: [] as TaskItem[], total: 0 }))
        if (cancelled) return
        setTasks(res.tasks as TaskItem[])
        setTotal(res.total ?? 0)
      } catch (error) {
        console.error('Failed to load tasks:', error)
      } finally {
        if (!cancelled) setTasksLoading(false)
      }
    }
    loadTasks()
    return () => { cancelled = true }
  }, [user, projectSlug, sprintSlug, currentPage, debouncedSearch])

  const getPriorityLabelKey = (priority: string): string => {
    switch (priority) {
      case 'urgent': return 'pms.urgent'
      case 'high': return 'pms.high'
      case 'medium': return 'pms.medium'
      case 'low': return 'pms.low'
      default: return priority || 'pms.medium'
    }
  }

  const getStatusName = (slug: string): string => {
    const camelSlug = slug.split('_').map(part => part.charAt(0).toUpperCase() + part.slice(1)).join('')
    const key = `pms.taskStatus${camelSlug}`
    const translated = t(key)
    return translated === key ? slug : translated
  }

  // Task detail handlers
  // 抽屉模式：'view' = 查看模式（点击任务行）；'edit' = 直接编辑模式（点击编辑按钮，保存后关闭）
  function handleTaskClick(taskId: number, mode: 'view' | 'edit' = 'view') {
    setDrawerMode(mode)
    setSelectedTaskId(taskId)
  }

  function handleCloseDrawer() {
    setSelectedTaskId(null)
    setDrawerMode('view')
    // 清除 URL 中的 task 参数，避免翻页/搜索时深链效果重复打开抽屉
    if (searchParams.has('task')) {
      const sp = new URLSearchParams(searchParams.toString())
      sp.delete('task')
      const qs = sp.toString()
      router.replace(`/projects/${projectSlug}/tasks${qs ? `?${qs}` : ''}`, { scroll: false })
    }
  }

  function handleTaskUpdated(updatedTask: any) {
    setTasks((prev) => prev.map((t) => (t.taskId === updatedTask.taskId ? { ...t, ...updatedTask } : t)))
    // 编辑保存后高亮，让用户确认改动已生效
    setHighlightedTaskId(updatedTask.taskId)
  }

  // 删除任务：从当前页移除并 decrement total；若当前页因此变空且非第 1 页，回退到上一页（触发重新请求）
  async function handleDeleteTask() {
    if (taskToDelete === null) return
    setDeleting(true)
    try {
      await api.deleteTask(projectSlug, taskToDelete)
      const wasLastOnPage = tasks.length <= 1
      setTasks((prev) => prev.filter((t) => t.taskId !== taskToDelete))
      setTotal((prev) => Math.max(0, prev - 1))
      setTaskToDelete(null)
      if (wasLastOnPage && currentPage > 1) {
        setCurrentPage((p) => p - 1)
      }
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setDeleting(false)
    }
  }

  if (authLoading) return null

  if (!user) {
    return (
      <Box py={20}>
        <VStack spacing={6} textAlign="center">
          <Heading size="lg">{t('common.pleaseLogin')}</Heading>
          <Button colorScheme="blue" onClick={() => router.push('/login')}>{t('auth.login')}</Button>
        </VStack>
      </Box>
    )
  }

  // 仅首次加载（无数据）时显示全屏 spinner；翻页/搜索时保留旧列表，靠分页按钮 disabled 表示加载中
  if (tasksLoading && tasks.length === 0) {
    return (
      <Flex justify="center" align="center" minH="60vh">
        <Spinner size="xl" color="blue.500" />
      </Flex>
    )
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  // 分页计数联动：当前页显示的条目范围（与分页控件联动，而非显示总数）
  const rangeStart = total === 0 ? 0 : (currentPage - 1) * PAGE_SIZE + 1
  const rangeEnd = Math.min(currentPage * PAGE_SIZE, total)

  return (
    <Stack spacing={6}>
    {/* Header */}
      <Flex justify="space-between" align="center">
        <Box>
          {sprintSlug && sprintInfo ? (
            // 看板页面：Sprint 标题做成链接，点击返回 Sprint 详情页
            <SprintBreadcrumb
              sprint={sprintInfo}
              href={`/projects/${projectSlug}/sprint/${sprintSlug}`}
            />
          ) : (
            <Flex align="center" gap={3}>
              <FiRepeat color="#3182ce" />
              <Heading size="md">{t('pms.allSprints')}</Heading>
            </Flex>
          )}
        </Box>
        <HStack spacing={3}>
          {/* 任务搜索框 - 后端搜索（防抖 300ms，与分页联动） */}
          <InputGroup size="sm" w="200px" h="32px">
            <InputLeftElement pointerEvents="none" h="32px">
              <FiSearch color="#94a3b8" size={14} />
            </InputLeftElement>
            <Input
              placeholder={t('common.search')}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              h="32px"
              borderWidth="1px"
              borderColor="gray.200"
              bg="white"
              borderRadius="6px"
            />
          </InputGroup>
          {/* Sprint 过滤器 */}
          <Select
            size="sm"
            h="32px"
            w="auto"
            minW="140px"
            value={sprintSlug || ''}
            onChange={(e) => changeSprint(e.target.value)}
            borderWidth="1px"
            borderColor="gray.200"
            bg="white"
          >
            <option value="">{t('pms.allSprints')}</option>
            {sprints.map((s) => (
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

      {/* 任务列表（对齐 Jira Issues & filters：全局任务列表 + 筛选；后端分页，每页 20 条） */}
      <Box
        bg="white"
        borderRadius="12px"
        border="1px solid #e2e8f0"
        p="20px"
        boxShadow="0 1px 3px rgba(0,0,0,0.04)"
        w="100%"
      >
          <VStack spacing={4} align="stretch">
            <Flex align="baseline" justify="space-between">
              <Heading size="md">{t('pms.tasks')}</Heading>
              <Text fontSize="xs" color="gray.500">
                {total === 0 ? t('common.noData') : t('common.itemRange', { start: rangeStart, end: rangeEnd, total })}
              </Text>
            </Flex>

            {tasks.length === 0 ? (
              <Text color="gray.500" textAlign="center" py={8}>{t('pms.noTasks')}</Text>
            ) : (
              tasks.map((task) => {
                const assignedSprint = sprints.find((s) => s.slug === task.sprintSlug)
                const isHighlighted = highlightedTaskId === task.taskId
                return (
                  <Box
                    key={task.taskId}
                    position="relative"
                    overflow="hidden"
                    p={4}
                    bg={isHighlighted ? 'blue.50' : 'gray.50'}
                    borderRadius="md"
                    cursor="pointer"
                    _hover={{ bg: 'gray.100' }}
                    onClick={() => handleTaskClick(task.taskId)}
                    transition="background-color 0.6s ease"
                    sx={isHighlighted ? { animation: `${highlightPulse} 1.2s ease-out` } : undefined}
                  >
                    {/* 左侧优先级色条（与看板 TaskCard 一致） */}
                    <Box
                      position="absolute"
                      left={0}
                      top={0}
                      bottom={0}
                      w="3px"
                      bg={getPriorityBorderColor(task.priority)}
                    />
                    <Flex justify="space-between" align="center">
                      <Box flex={1}>
                        <HStack spacing={2} mb={2} align="center">
                          <Text fontSize="md" lineHeight="1">{getTaskTypeIcon(task.taskType).label}</Text>
                          <Text fontWeight="medium" fontSize="md">{task.title}</Text>
                        </HStack>
                        <HStack spacing={4} fontSize="sm" color="gray.500" flexWrap="wrap">
                          <HStack spacing={1}>
                            <Text>{t('common.priority')}:</Text>
                            <Badge colorScheme="gray" size="sm">
                              {t(getPriorityLabelKey(task.priority))}
                            </Badge>
                          </HStack>
                          <HStack spacing={1}>
                            <Text>{t('common.status')}:</Text>
                            <Badge colorScheme="gray" size="sm">
                              {getStatusName(task.status)}
                            </Badge>
                          </HStack>
                          {task.taskType && (
                            <HStack spacing={1}>
                              <Text>{t('pms.taskType')}:</Text>
                              <Badge colorScheme="gray" size="sm">
                                {t(`pms.type${task.taskType.charAt(0).toUpperCase() + task.taskType.slice(1)}`)}
                              </Badge>
                            </HStack>
                          )}
                          {assignedSprint && (
                            <HStack spacing={1} color="blue.600">
                              <Icon as={FiRepeat} />
                              <Text>{assignedSprint.title}</Text>
                            </HStack>
                          )}
                          {(task.storyPoints ?? 0) > 0 && (
                            <HStack spacing={1}>
                              <Icon as={FiCheckSquare} />
                              <Text>{task.storyPoints} {t('pms.storyPoints')}</Text>
                            </HStack>
                          )}
                        </HStack>
                      </Box>
                      {/* 编辑 + 删除按钮：样式与需求池用户故事列表保持一致（xs 尺寸 + FiEdit/FiTrash2） */}
                      {canEditScrum && (
                        <HStack spacing={2}>
                          <IconButton
                            aria-label={t('common.edit')}
                            icon={<FiEdit />}
                            size="xs"
                            variant="ghost"
                            onClick={(e) => {
                              e.stopPropagation()
                              handleTaskClick(task.taskId, 'edit')
                            }}
                          />
                          <IconButton
                            aria-label={t('common.delete')}
                            icon={<FiTrash2 />}
                            size="xs"
                            variant="ghost"
                            color="gray.400"
                            _hover={{ color: 'red.500' }}
                            onClick={(e) => {
                              e.stopPropagation()
                              setTaskToDelete(task.taskId)
                            }}
                          />
                        </HStack>
                      )}
                    </Flex>
                  </Box>
                )
              })
            )}
            {/* 分页控件：仅当超过 1 页时显示；加载中禁用按钮防止重复请求 */}
            {totalPages > 1 && (
              <Flex justify="flex-end" align="center" pt={4} borderTop="1px solid" borderColor="gray.100">
                <HStack spacing={3}>
                  <IconButton
                    aria-label={t('common.prevPage')}
                    icon={<FiChevronLeft />}
                    size="sm"
                    variant="outline"
                    isDisabled={currentPage === 1 || tasksLoading}
                    onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                  />
                  <Text fontSize="sm" color="gray.600" whiteSpace="nowrap">
                    {t('common.pageInfo', { page: currentPage, totalPages })}
                  </Text>
                  <IconButton
                    aria-label={t('common.nextPage')}
                    icon={<FiChevronRight />}
                    size="sm"
                    variant="outline"
                    isDisabled={currentPage === totalPages || tasksLoading}
                    onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                  />
                </HStack>
              </Flex>
            )}
          </VStack>
        </Box>

      {/* Task Detail Drawer */}
      <TaskDrawer
        isOpen={selectedTaskId !== null}
        taskId={selectedTaskId}
        onClose={handleCloseDrawer}
        projectSlug={projectSlug}
        taskStatuses={taskStatuses}
        canEditScrum={canEditScrum}
        onTaskUpdated={handleTaskUpdated}
        // 'edit' 模式（点击编辑按钮）：直接编辑，保存后关闭抽屉；'view' 模式（点击任务行）：查看
        mode={drawerMode}
      />

      {/* New Task Drawer - unified TaskDrawer in create mode */}
      <TaskDrawer
        isOpen={isNewTaskDrawerOpen}
        taskId={null}
        mode="create"
        onClose={() => setIsNewTaskDrawerOpen(false)}
        projectSlug={projectSlug}
        taskStatuses={taskStatuses}
        canEditScrum={canEditScrum}
        sprintSlug={sprintSlug || undefined}
        // 新任务按排序应在第 1 页：当前在第 1 页则直接 prepend 提供即时反馈；否则切到第 1 页由后端返回
        onTaskCreated={(task) => {
          setTotal((prev) => prev + 1)
          if (currentPage === 1) {
            setTasks((prev) => [task, ...prev])
          } else {
            setCurrentPage(1)
          }
        }}
      />

      {/* 删除任务确认 AlertDialog（与需求池/项目删除保持一致的对话框风格） */}
      <AlertDialog
        isOpen={taskToDelete !== null}
        leastDestructiveRef={cancelRef}
        onClose={() => setTaskToDelete(null)}
        blockScrollOnMount={false}
      >
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              {t('common.delete')}
            </AlertDialogHeader>
            <AlertDialogBody>
              {t('pms.deleteTaskConfirm')}
            </AlertDialogBody>
            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={() => setTaskToDelete(null)}>
                {t('common.cancel')}
              </Button>
              <Button colorScheme="red" onClick={handleDeleteTask} ml={3} isLoading={deleting}>
                {t('common.delete')}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>
    </Stack>
  )
}

export default function TaskListPage() {
  return (
    <Suspense fallback={<Flex justify="center" align="center" minH="60vh"><Spinner size="xl" color="blue.500" /></Flex>}>
      <TasksContent />
    </Suspense>
  )
}
