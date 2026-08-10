'use client'

import { useState, useRef, useMemo } from 'react'
import NextLink from 'next/link'
import {
  Box,
  Flex,
  Text,
  Avatar,
  HStack,
  VStack,
  Button,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  useToast,
} from '@chakra-ui/react'
import { FiChevronDown, FiLayers, FiTarget, FiUser, FiFlag, FiSettings, FiBook } from 'react-icons/fi'
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useSensor,
  useSensors,
  closestCorners,
  type DragStartEvent,
  type DragOverEvent,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { getEpicColor } from '@/lib/epicColor'
import type { TaskStatus } from '@/lib/types'
import KanbanColumn from './KanbanColumn'
import SortableTaskCard from './TaskCard'
import AddTaskForm from './AddTaskForm'

// --- Types ---

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
  // Epic 关联信息(后端 sideload 填充),透传给 TaskCard 显示彩色标签
  epicSlug?: string | null
  epicTitle?: string | null
  // Sprint 归属(聚合视图下用于在卡片显示 Sprint Badge)
  sprintSlug?: string | null
  // UserStory 关联信息(后端 sideload 填充),透传给 TaskCard 显示所属故事徽标
  userStorySlug?: string | null
  userStoryTitle?: string | null
  // 未完成前置依赖数(后端 sideload 填充),>0 时卡片显示阻塞标记
  blockedByCount?: number
  projectSlug?: string
}

interface ColumnDef {
  id: number
  status: string
  name: string
  color: string
  // 看板列在制品上限(对齐 Jira Kanban WIP);null/undefined=不限制,>0 时列头显示 N/Limit 超限标红
  wipLimit?: number | null
}

// Swimlane 分组维度(参考 Jira:支持按 Epic/经办人/优先级分泳道,或平铺)
type SwimlaneDimension = 'none' | 'epic' | 'assignee' | 'priority' | 'story'

// Swimlane 分组结果
interface SwimlaneGroup {
  key: string          // 分组唯一 key(用于 React key + DnD)
  title: string        // 分组标题(Epic 名/经办人名/优先级名)
  color?: string       // 分组颜色(Epic 用彩色标识,其他维度用状态色)
  tasks: TaskItem[]    // 该组内的所有任务
}

// --- Priority left-border color ---
function getPriorityBorderColor(priority: string): string {
  switch (priority) {
    case 'urgent': return '#ef4444'
    case 'high': return '#f97316'
    case 'medium': return '#eab308'
    case 'low': return '#22c55e'
    default: return '#e2e8f0'
  }
}

// 状态别名映射：将历史/非标准状态值规范化到项目配置的 slug
// 防止任务因 status 不在 taskStatuses 配置中而无法在看板显示
// 新 4 状态模型：todo / in_progress / review / done
const STATUS_ALIASES: Record<string, string> = {
  open: 'todo',
  completed: 'done',
  closed: 'done',
  archived: 'done',
  active: 'in_progress',
  doing: 'in_progress',
}

function normalizeStatus(status: string): string {
  return STATUS_ALIASES[status] || status
}

function getTaskTypeIcon(taskType: string): { label: string; color: string } {
  switch (taskType) {
    case 'bug': return { label: '🐛', color: 'red.500' }
    case 'feature': return { label: '🚀', color: 'purple.500' }
    case 'improvement': return { label: '🔧', color: 'yellow.600' }
    default: return { label: '📋', color: 'blue.500' }
  }
}

interface KanbanBoardProps {
  tasks: TaskItem[]
  taskStatuses: TaskStatus[]
  projectSlug: string
  setTasks: React.Dispatch<React.SetStateAction<TaskItem[]>>
  onTaskClick: (taskId: number) => void
  sprintSlug?: string
  canEditScrum: boolean
  highlightedTaskId?: number | null
  sprintSlugToTitle?: Record<string, string>
  showSprintBadge?: boolean
}

// --- Kanban Board ---
export default function KanbanBoard({
  tasks,
  taskStatuses,
  projectSlug,
  setTasks,
  onTaskClick,
  sprintSlug,
  canEditScrum,
  highlightedTaskId,
  sprintSlugToTitle,
  showSprintBadge,
}: KanbanBoardProps) {
  const { t } = useI18n()
  const toast = useToast()

  // DnD state
  const [activeId, setActiveId] = useState<number | null>(null)
  const [overStatus, setOverStatus] = useState<string | null>(null)
  const activeTaskRef = useRef<TaskItem | null>(null)
  const originalStatusRef = useRef<string | null>(null)
  const dragDirectionRef = useRef<'up' | 'down'>('down')

  // Inline add-task form state
  const [addingTaskStatus, setAddingTaskStatus] = useState<string | null>(null)

  // Swimlane 分组维度(默认平铺,可选 Epic/经办人/优先级分泳道)
  const [swimlane, setSwimlane] = useState<SwimlaneDimension>('none')

  // viewer 禁用拖拽（对齐后端 UpdateTask 要求 member+）
  const sensors = useSensors(
    ...(canEditScrum
      ? [useSensor(PointerSensor, { activationConstraint: { distance: 8 } })]
      : [])
  )

  // --- Drag handlers ---
  function handleDragStart(event: DragStartEvent) {
    const id = event.active.id as number
    setActiveId(id)
    const task = tasks.find((t) => t.taskId === id) || null
    activeTaskRef.current = task
    originalStatusRef.current = task?.status || null
    dragDirectionRef.current = 'down'
  }

  function handleDragOver(event: DragOverEvent) {
    const { active, over } = event
    if (!over) return

    const activeId = active.id as number
    const activeStatus = active.data.current?.status as string | undefined
    const overStatus = (over.data.current?.status as string | undefined) ?? null

    setOverStatus(overStatus)

    // Track drag direction based on vertical movement
    const activeRect = active.rect.current.translated
    const overRect = over.rect
    if (activeRect && overRect) {
      const activeY = activeRect.top
      const overY = overRect.top
      if (activeY < overY) {
        dragDirectionRef.current = 'down'
      } else if (activeY > overY) {
        dragDirectionRef.current = 'up'
      }
    }

    const overTaskId = typeof over.id === 'number' ? over.id : null

    // 同列内拖拽排序 - 根据拖拽方向决定插入位置
    if (overStatus && activeStatus === overStatus && overTaskId) {
      setTasks((prev) => {
        const activeIndex = prev.findIndex((t) => t.taskId === activeId)
        const overIndex = prev.findIndex((t) => t.taskId === overTaskId)
        if (activeIndex !== -1 && overIndex !== -1 && activeIndex !== overIndex) {
          // 根据拖拽方向调整插入位置
          let targetIndex = overIndex
          if (dragDirectionRef.current === 'down') {
            // 向下拖：插入到目标后面
            targetIndex = overIndex + 1
          } else {
            // 向上拖：插入到目标前面（即目标位置）
            targetIndex = overIndex
          }
          // arrayMove 会先移除再插入，需要调整索引
          const removed = [...prev]
          removed.splice(activeIndex, 1)
          const adjustedIndex = activeIndex < targetIndex ? targetIndex - 1 : targetIndex
          removed.splice(adjustedIndex, 0, prev[activeIndex])
          return removed
        }
        return prev
      })
      return
    }

    // 跨列拖拽
    if (overStatus && activeStatus !== overStatus) {
      setTasks((prev) => {
        const activeTask = prev.find((t) => t.taskId === activeId)
        if (!activeTask) return prev

        const otherTasks = prev.filter((t) => t.taskId !== activeId)
        const overTask = overTaskId ? otherTasks.find((t) => t.taskId === overTaskId) : null

        let insertIndex: number
        if (overTask) {
          const baseIndex = otherTasks.indexOf(overTask)
          // 根据拖拽方向决定插入到目标前还是后
          if (dragDirectionRef.current === 'down') {
            insertIndex = baseIndex + 1
          } else {
            insertIndex = baseIndex
          }
        } else {
          // 拖到空白列，插入到该列末尾
          const columnTasks = otherTasks.filter((t) => t.status === overStatus)
          insertIndex = columnTasks.length > 0
            ? otherTasks.indexOf(columnTasks[columnTasks.length - 1]) + 1
            : otherTasks.length
        }

        const updated = [...otherTasks]
        updated.splice(insertIndex, 0, { ...activeTask, status: overStatus })
        return updated
      })
    }
  }

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    const overColStatus = (over?.data.current?.status as string | undefined) ?? null
    const originalStatus = originalStatusRef.current

    setActiveId(null)
    setOverStatus(null)
    activeTaskRef.current = null
    originalStatusRef.current = null

    let finalStatus: string | null = null
    if (overColStatus) {
      finalStatus = overColStatus
    } else if (over) {
      const overTask = tasks.find((t) => t.taskId === over.id)
      if (overTask) finalStatus = overTask.status
    }

    if (!finalStatus || !originalStatus) return

    const taskId = active.id as number
    const isCrossColumn = originalStatus !== finalStatus

    // 持久化：更新该列所有任务的 position
    const affectedColumnTasks = tasks.filter((t) => t.status === finalStatus)
    const updatePromises = affectedColumnTasks.map((task, index) =>
      api.updateTaskPosition(projectSlug, task.taskId, index)
    )

    Promise.all(updatePromises).catch((error: any) => {
      toast({
        title: t('pms.updateFailed'),
        description: error.message || t('pms.updateFailed'),
        status: 'error',
        duration: 3000,
      })
    })

    // 跨列拖拽还需要更新状态
    if (isCrossColumn) {
      api.updateTask(projectSlug, taskId, { status: finalStatus }).catch((error: any) => {
        // 回滚状态
        setTasks((prev) =>
          prev.map((t) =>
            t.taskId === taskId ? { ...t, status: originalStatus } : t
          )
        )
        if (error.reason === 'task_blocked' && error.blockedTasks) {
          const taskList = error.blockedTasks.map((bt: any) => `#${bt.taskId} ${bt.title}`).join(', ')
          toast({
            title: t('pms.taskBlockedTitle'),
            description: t('pms.taskBlockedDesc', { tasks: taskList }),
            status: 'warning',
            duration: 5000,
          })
        } else {
          toast({
            title: t('pms.updateFailed'),
            description: error.message || t('pms.updateFailed'),
            status: 'error',
            duration: 3000,
          })
        }
      })
    }
  }

  // --- Add task handler ---
  function handleTaskCreated(newTask: TaskItem) {
    setTasks((prev) => [newTask, ...prev])
    setAddingTaskStatus(null)
  }

  // 根据 slug 获取翻译后的列名
  function getStatusName(slug: string): string {
    if (!slug) return ''
    // 将 slug 转换为驼峰命名：in_progress -> InProgress
    const camelSlug = slug.split('_').map(part => part.charAt(0).toUpperCase() + part.slice(1)).join('')
    const key = `pms.taskStatus${camelSlug}`
    const translated = t(key)
    // 如果翻译结果和 key 相同（未找到翻译），使用 slug 作为 fallback
    return translated === key ? slug : translated
  }

  // 优先级显示配置(与 Backlog 优先级分组一致)
  const PRIORITY_DISPLAY: Record<string, { label: string; color: string; order: number }> = {
    urgent: { label: t('pms.urgent'), color: '#ef4444', order: 0 },
    high: { label: t('pms.high'), color: '#f97316', order: 1 },
    medium: { label: t('pms.medium'), color: '#eab308', order: 2 },
    low: { label: t('pms.low'), color: '#22c55e', order: 3 },
  }

  // 按 swimlane 维度分组任务(参考 Jira Swimlane:每组一行,组内按状态分列)
  // "未分配"组(Epic 无关联/经办人未指派/优先级缺失)放最后
  const swimlaneGroups = useMemo((): SwimlaneGroup[] => {
    if (swimlane === 'none') return []
    const groups = new Map<string, SwimlaneGroup>()
    const unassignedKey = '__unassigned__'

    for (const task of tasks) {
      let key: string
      let title: string
      let color: string | undefined

      switch (swimlane) {
        case 'epic': {
          if (task.epicSlug && task.epicTitle) {
            key = task.epicSlug
            title = task.epicTitle
            color = getEpicColor(task.epicSlug)
          } else {
            key = unassignedKey
            title = t('pms.noEpic')
            color = '#94a3b8'
          }
          break
        }
        case 'assignee': {
          // 后端列表 API 隐藏了 assigneeName(json:"-"),经办人通过 assignees 数组获取
          // 取第一个 assignee 作为主经办人(与 TaskCard 显示逻辑一致)
          const primaryAssignee = task.assignees?.[0]?.userName || task.assigneeName
          if (primaryAssignee) {
            key = `assignee_${primaryAssignee}`
            title = task.assignees?.[0]?.fullName || primaryAssignee
            color = undefined
          } else {
            key = unassignedKey
            title = t('pms.unassigned')
            color = '#94a3b8'
          }
          break
        }
        case 'priority': {
          const p = task.priority || 'medium'
          const display = PRIORITY_DISPLAY[p] || PRIORITY_DISPLAY.medium
          key = `priority_${p}`
          title = display.label
          color = display.color
          break
        }
        case 'story': {
          if (task.userStorySlug && task.userStoryTitle) {
            key = task.userStorySlug
            title = task.userStoryTitle
            color = '#8b5cf6'
          } else {
            key = unassignedKey
            title = t('pms.noUserStory')
            color = '#94a3b8'
          }
          break
        }
        default:
          continue
      }

      if (!groups.has(key)) {
        groups.set(key, { key, title, color, tasks: [] })
      }
      groups.get(key)!.tasks.push(task)
    }

    // 转为数组并排序:"未分配"组永远放最后,其他按出现顺序(优先级维度按紧急度排序)
    const result = Array.from(groups.values())
    if (swimlane === 'priority') {
      result.sort((a, b) => {
        if (a.key === unassignedKey) return 1
        if (b.key === unassignedKey) return -1
        const orderA = PRIORITY_DISPLAY[a.key.replace('priority_', '')]?.order ?? 99
        const orderB = PRIORITY_DISPLAY[b.key.replace('priority_', '')]?.order ?? 99
        return orderA - orderB
      })
    } else {
      result.sort((a, b) => {
        if (a.key === unassignedKey) return 1
        if (b.key === unassignedKey) return -1
        return 0
      })
    }
    return result
  }, [tasks, swimlane, t])

  // 动态生成列（去重，避免重复的 slug）
  const uniqueStatuses = taskStatuses.filter((status, index, self) =>
    index === self.findIndex((s) => s.slug === status.slug)
  )
  const columns: ColumnDef[] = uniqueStatuses.map(status => ({
    id: status.id,
    status: status.slug,
    name: getStatusName(status.slug),
    color: status.color,
    wipLimit: status.wipLimit ?? null,
  }))

  const activeTask = activeTaskRef.current

  // Swimlane 切换器配置
  const swimlaneOptions: { value: SwimlaneDimension; label: string; icon: any }[] = [
    { value: 'none', label: t('pms.swimlaneNone'), icon: FiLayers },
    { value: 'epic', label: t('pms.swimlaneEpic'), icon: FiTarget },
    { value: 'story', label: t('pms.swimlaneStory'), icon: FiBook },
    { value: 'assignee', label: t('pms.swimlaneAssignee'), icon: FiUser },
    { value: 'priority', label: t('pms.swimlanePriority'), icon: FiFlag },
  ]
  const currentSwimlaneOption = swimlaneOptions.find(o => o.value === swimlane) || swimlaneOptions[0]

  // 渲染单组 swimlane 的列布局(与平铺模式列结构一致,只是 tasks 限定在该组内)
  const renderSwimlaneColumns = (groupTasks: TaskItem[]) => (
    <Flex
      gap="12px"
      align="flex-start"
      w="100%"
      overflowX="auto"
    >
      {columns.map((column) => {
        const columnTasks = groupTasks.filter((t) => normalizeStatus(t.status) === column.status)
        const isOver = overStatus === column.status
        const isAdding = addingTaskStatus === column.status

        return (
          <KanbanColumn
            key={column.status}
            column={column}
            tasks={columnTasks}
            isOver={isOver}
            onAddTask={canEditScrum ? () => setAddingTaskStatus(column.status) : undefined}
          >
            <SortableContext
              items={columnTasks.map((t) => t.taskId)}
              strategy={verticalListSortingStrategy}
            >
              {columnTasks.map((task) => (
                <SortableTaskCard
                  key={task.taskId}
                  task={task}
                  sprintTitle={showSprintBadge && task.sprintSlug ? sprintSlugToTitle?.[task.sprintSlug] : undefined}
                  onClick={() => onTaskClick(task.taskId)}
                  isHighlighted={highlightedTaskId === task.taskId}
                />
              ))}
            </SortableContext>

            {isAdding && (
              <AddTaskForm
                status={column.status}
                projectSlug={projectSlug}
                sprintSlug={sprintSlug}
                onTaskCreated={handleTaskCreated}
                onCancel={() => setAddingTaskStatus(null)}
              />
            )}
          </KanbanColumn>
        )
      })}
    </Flex>
  )

  return (
    <VStack spacing={3} align="stretch" w="100%" overflow="hidden">
      {/* Swimlane 切换器(参考 Jira:看板顶部可切换分组维度) */}
      <Flex justify="flex-end" align="center" gap={2}>
        {canEditScrum && (
          <Button
            as={NextLink}
            href={`/projects/${projectSlug}/statuses`}
            size="sm"
            variant="outline"
            colorScheme="gray"
            leftIcon={<FiSettings size={14} />}
          >
            {t('statusConfig.configureStatuses')}
          </Button>
        )}
        <Menu>
          <MenuButton
            as={Button}
            size="sm"
            variant="outline"
            colorScheme="gray"
            rightIcon={<FiChevronDown />}
            leftIcon={<currentSwimlaneOption.icon size={14} />}
          >
            {t('pms.swimlane')}: {currentSwimlaneOption.label}
          </MenuButton>
          <MenuList minW="180px">
            {swimlaneOptions.map(opt => (
              <MenuItem
                key={opt.value}
                icon={<opt.icon size={14} />}
                onClick={() => setSwimlane(opt.value)}
                bg={swimlane === opt.value ? 'blue.50' : undefined}
                fontWeight={swimlane === opt.value ? 'semibold' : 'normal'}
              >
                {opt.label}
              </MenuItem>
            ))}
          </MenuList>
        </Menu>
      </Flex>

      <Flex justify="center" pb="4" w="100%">
        <DndContext
          sensors={sensors}
          onDragStart={handleDragStart}
          onDragOver={handleDragOver}
          onDragEnd={handleDragEnd}
          collisionDetection={closestCorners}
        >
          {swimlane === 'none' ? (
            /* 平铺模式:所有列并排,与原看板一致 */
            <Flex
              bg="white"
              borderRadius="12px"
              border="1px solid #e2e8f0"
              p="20px"
              boxShadow="0 1px 3px rgba(0,0,0,0.04)"
              w="100%"
              minW={0}
              gap="16px"
              align="flex-start"
              overflowX="auto"
            >
              {columns.map((column) => {
                const columnTasks = tasks.filter((t) => normalizeStatus(t.status) === column.status)
                const isOver = overStatus === column.status
                const isAdding = addingTaskStatus === column.status

                return (
                  <KanbanColumn
                    key={column.status}
                    column={column}
                    tasks={columnTasks}
                    isOver={isOver}
                    onAddTask={canEditScrum ? () => setAddingTaskStatus(column.status) : undefined}
                  >
                    <SortableContext
                      items={columnTasks.map((t) => t.taskId)}
                      strategy={verticalListSortingStrategy}
                    >
                      {columnTasks.map((task) => (
                        <SortableTaskCard
                          key={task.taskId}
                          task={task}
                          onClick={() => onTaskClick(task.taskId)}
                          isHighlighted={highlightedTaskId === task.taskId}
                        />
                      ))}
                    </SortableContext>

                    {isAdding && (
                      <AddTaskForm
                        status={column.status}
                        projectSlug={projectSlug}
                        sprintSlug={sprintSlug}
                        onTaskCreated={handleTaskCreated}
                        onCancel={() => setAddingTaskStatus(null)}
                      />
                    )}
                  </KanbanColumn>
                )
              })}
            </Flex>
          ) : (
            /* Swimlane 模式:按维度分组,每组一行(标题 + 列布局) */
            <VStack spacing={3} align="stretch" w="100%">
              {swimlaneGroups.length === 0 ? (
                <Box p={8} textAlign="center" color="gray.400" fontSize="sm">
                  {t('pms.swimlaneNoData')}
                </Box>
              ) : (
                swimlaneGroups.map((group) => (
                  <Box
                    key={group.key}
                    bg="white"
                    borderRadius="12px"
                    border="1px solid #e2e8f0"
                    boxShadow="0 1px 3px rgba(0,0,0,0.04)"
                    overflow="hidden"
                  >
                    {/* Swimlane 分组标题行 */}
                    <Flex
                      align="center"
                      px="16px"
                      py="8px"
                      bg="gray.50"
                      borderBottom="1px solid #e2e8f0"
                      position="relative"
                    >
                      {/* 左侧彩色色条(与 Backlog Epic 分组标题一致) */}
                      {group.color && (
                        <Box position="absolute" left={0} top={0} bottom={0} w="4px" bg={group.color} />
                      )}
                      <HStack spacing={2} ml={group.color ? 2 : 0}>
                        {swimlane === 'epic' && <FiTarget size={14} color={group.color || '#94a3b8'} />}
                        {swimlane === 'assignee' && (
                          group.key !== '__unassigned__' ? (
                            <Avatar size="2xs" name={group.title} fontSize="9px" />
                          ) : (
                            <FiUser size={14} color="#94a3b8" />
                          )
                        )}
                        {swimlane === 'priority' && (
                          <Box w="10px" h="10px" borderRadius="full" bg={group.color || '#94a3b8'} />
                        )}
                        {swimlane === 'story' && <FiBook size={14} color={group.color || '#94a3b8'} />}
                        <Text fontSize="sm" fontWeight="semibold" color="gray.800">
                          {group.title}
                        </Text>
                        <Text fontSize="xs" color="gray.400">
                          {group.tasks.length} {t('pms.tasks')}
                        </Text>
                      </HStack>
                    </Flex>
                    {/* 该组的列布局 */}
                    <Box p="12px">
                      {renderSwimlaneColumns(group.tasks)}
                    </Box>
                  </Box>
                ))
              )}
            </VStack>
          )}

          {/* Drag Overlay */}
          <DragOverlay dropAnimation={{
            duration: 200,
            easing: 'ease',
          }}>
            {activeTask ? (
              <Box
                bg="white"
                borderRadius="8px"
                borderWidth="1px"
                borderColor="#3b82f6"
                boxShadow="0 8px 25px rgba(0,0,0,0.15), 0 0 0 1px rgba(59,130,246,0.1)"
                w="300px"
                cursor="grabbing"
                overflow="hidden"
                position="relative"
              >
                <Box
                  position="absolute"
                  left={0}
                  top={0}
                  bottom={0}
                  w="3px"
                  bg={getPriorityBorderColor(activeTask.priority)}
                />
                <Box pl="14px" pr="12px" py="10px">
                  <Flex align="center" justify="space-between" mb="6px">
                    <HStack spacing="6px">
                      <Text fontSize="13px">{getTaskTypeIcon(activeTask.taskType).label}</Text>
                      <Text fontSize="11px" color="#94a3b8" fontWeight="500">
                        TASK-{activeTask.taskId}
                      </Text>
                    </HStack>
                    {activeTask.storyPoints != null && activeTask.storyPoints > 0 && (
                      <Box
                        bg="#f1f5f9"
                        color="#475569"
                        fontSize="10px"
                        fontWeight="600"
                        px="5px"
                        py="1px"
                        borderRadius="4px"
                      >
                        {activeTask.storyPoints}
                      </Box>
                    )}
                  </Flex>
                  <Text fontSize="13px" fontWeight="500" color="#1e293b" lineHeight="1.4" noOfLines={2}>
                    {activeTask.title}
                  </Text>
                  {activeTask.assigneeName && (
                    <Flex align="center" gap="6px" mt="8px">
                      <Avatar
                        size="xs"
                        name={activeTask.assigneeName}
                        bg="#e2e8f0"
                        color="#475569"
                        fontSize="9px"
                      />
                      <Text fontSize="11px" color="#64748b">
                        {activeTask.assigneeName}
                      </Text>
                    </Flex>
                  )}
                </Box>
              </Box>
            ) : null}
          </DragOverlay>
        </DndContext>
      </Flex>
    </VStack>
  )
}
