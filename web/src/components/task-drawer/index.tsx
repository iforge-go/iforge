'use client'

import { useState, useEffect } from 'react'
import {
  Box,
  Flex,
  Text,
  Badge,
  Avatar,
  Heading,
  Button,
  Spinner,
  VStack,
  HStack,
  Checkbox,
  IconButton,
  useToast,
  Drawer,
  DrawerBody,
  DrawerHeader,
  DrawerOverlay,
  DrawerContent,
  DrawerCloseButton,
  Textarea,
  Input,
  Select,
  Icon,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  ButtonGroup,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverHeader,
  PopoverBody,
  Stack,
} from '@chakra-ui/react'
import { FiTag, FiTrash2, FiPlus, FiChevronDown, FiRepeat, FiBook, FiClock, FiActivity, FiGitCommit, FiCpu, FiCalendar, FiX, FiExternalLink } from 'react-icons/fi'
import NextLink from 'next/link'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import { UserSearchResult, BranchCommits, AIOptimizeTaskResult, Participant, Task, TaskStatus, TaskComment, TaskWorkLog, TaskStatusHistory, Sprint, UserStory, ScrumActivity } from '@/lib/types'
import { ApiError } from '@/lib/errorMessages'
import { getPriorityLabelKey, getPriorityColor } from './taskMeta'

// 任务详情响应类型（包含 sideload 数据）
interface TaskDetailResponse {
  task: Task
  workLogs?: TaskWorkLog[]
  actualHours?: number
  assignees?: { userName: string; fullName?: string; image?: string | null }[]
}

// 任务更新响应类型
interface TaskUpdateResponse {
  task: Task
  assignees?: { userName: string; fullName?: string; image?: string | null }[]
}

// 阻塞任务信息
interface BlockedTask {
  taskId: number
  title: string
  status?: string
}
import TaskAssignees from './TaskAssignees'
import TaskComments from './TaskComments'
import TaskTimeTracking from './TaskTimeTracking'
import TaskBranches from './TaskBranches'
import TaskDependencies from './TaskDependencies'
import ActivityTimeline from '../ActivityTimeline'
import AIOptimizeModal from '../AIOptimizeModal'

interface TaskDrawerProps {
  isOpen: boolean
  taskId: number | null
  onClose: () => void
  projectSlug: string
  taskStatuses: TaskStatus[]
  onTaskUpdated?: (updatedTask: Task) => void
  /** 'create' = 新建任务（直接显示编辑表单）；'edit' = 编辑模式（打开即编辑，保存后关闭抽屉）；'view' = 查看模式（默认） */
  mode?: 'view' | 'edit' | 'create'
  /** Called after a task is created (create mode only) */
  onTaskCreated?: (task: Task) => void
  /** Optional context for task creation */
  sprintSlug?: string
  userStorySlug?: string
  /** 当前用户是否有 Scrum 写权限（viewer 为 false）。create 模式始终可编辑。 */
  canEditScrum?: boolean
}

const PRIORITIES = ['urgent', 'high', 'medium', 'low']

export default function TaskDrawer({ isOpen, taskId, onClose, projectSlug, taskStatuses: taskStatusesProp, onTaskUpdated, mode = 'view', onTaskCreated, sprintSlug, userStorySlug, canEditScrum = false }: TaskDrawerProps) {
  const { t } = useI18n()
  const toast = useToast()
  const { user: currentUser } = useCurrentUser()
  const isCreateMode = mode === 'create'
  const isEditMode = mode === 'edit'
  const taskStatuses = taskStatusesProp || []

  const [taskDetail, setTaskDetail] = useState<Task | null>(null)
  const [taskDetailLoading, setTaskDetailLoading] = useState(false)
  const [taskComments, setTaskComments] = useState<TaskComment[]>([])
  const [taskAssignees, setTaskAssignees] = useState<{ userName: string; fullName: string; image: string | null }[]>([])
  const [isEditing, setIsEditing] = useState(false)
  const [editForm, setEditForm] = useState({
    title: '', description: '', status: 'todo', priority: 'medium', taskType: 'task', storyPoints: '', estimatedHours: '', dueDate: '',
  })
  const [savingEdit, setSavingEdit] = useState(false)
  const [newComment, setNewComment] = useState('')
  const [submittingComment, setSubmittingComment] = useState(false)
  const [addingWorkLog, setAddingWorkLog] = useState(false)

  // Subtask state
  const [subtasks, setSubtasks] = useState<Task[]>([])
  const [newSubtaskTitle, setNewSubtaskTitle] = useState('')
  const [addingSubtask, setAddingSubtask] = useState(false)
  const [subtaskLoading, setSubtaskLoading] = useState(false)

  // Activities state
  const [activities, setActivities] = useState<ScrumActivity[]>([])
  // Sideload participants：activity 操作者头像信息，由 getActivities 一次性返回
  const [activityParticipants, setActivityParticipants] = useState<Participant[]>([])

  // Task commits state (commits on associated branches, ahead of default branch)
  const [taskCommits, setTaskCommits] = useState<BranchCommits[]>([])
  const [taskCommitsLoading, setTaskCommitsLoading] = useState(false)

  // Status change history state
  const [statusHistory, setStatusHistory] = useState<TaskStatusHistory[]>([])

  // Breadcrumb context: sprint / story info (for Sprint >> Story >> Task nav)
  const [sprintInfo, setSprintInfo] = useState<Sprint | null>(null)
  const [storyInfo, setStoryInfo] = useState<UserStory | null>(null)

  // Task assignee search state
  const [userSearchKeyword, setUserSearchKeyword] = useState('')
  const [userSearchResults, setUserSearchResults] = useState<UserSearchResult[]>([])
  const [userSearchLoading, setUserSearchLoading] = useState(false)
  const [isAssigneePopoverOpen, setIsAssigneePopoverOpen] = useState(false)

  // Inline update loading flags
  const [updatingStatus, setUpdatingStatus] = useState(false)
  const [updatingPriority, setUpdatingPriority] = useState(false)
  const [updatingAssignees, setUpdatingAssignees] = useState(false)

  // Sprint / User Story 关联：列表 + inline 更新状态
  const [allSprints, setAllSprints] = useState<Sprint[]>([])
  const [allUserStories, setAllUserStories] = useState<UserStory[]>([])
  // create 模式下用户选择的所属故事(初始来自 userStorySlug prop,可覆盖;空串=不关联)
  const [createUserStorySlug, setCreateUserStorySlug] = useState<string>('')
  const [updatingSprint, setUpdatingSprint] = useState(false)
  const [updatingStory, setUpdatingStory] = useState(false)
  const [isSprintPopoverOpen, setIsSprintPopoverOpen] = useState(false)
  const [isStoryPopoverOpen, setIsStoryPopoverOpen] = useState(false)

  // AI 优化任务草稿弹窗（新建/编辑表单中复用）
  const [isAIOptimizeOpen, setIsAIOptimizeOpen] = useState(false)

  // Search users with debounce
  useEffect(() => {
    if (!userSearchKeyword.trim()) {
      setUserSearchResults([])
      return
    }
    setUserSearchLoading(true)
    const timer = setTimeout(async () => {
      try {
        const result = await api.searchUsers(userSearchKeyword, 20)
        setUserSearchResults(result.users || [])
      } catch {
        setUserSearchResults([])
      } finally {
        setUserSearchLoading(false)
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [userSearchKeyword])

  // Reset form when drawer opens in create mode
  useEffect(() => {
    if (isOpen && isCreateMode) {
      setTaskAssignees([])
      setEditForm({ title: '', description: '', status: 'todo', priority: 'medium', taskType: 'task', storyPoints: '', estimatedHours: '', dueDate: '' })
      setTaskDetail(null)
      setTaskComments([])
    }
  }, [isOpen, isCreateMode])

  // Load task data when taskId changes (view/edit mode only)
  useEffect(() => {
    if (isCreateMode) return
    if (!taskId) {
      setTaskDetail(null)
      setTaskComments([])
      setTaskAssignees([])
      setSubtasks([])
      setIsEditing(false)
      setNewComment('')
      setTaskCommits([])
      return
    }

    setTaskDetailLoading(true)
    setTaskDetail(null)
    setTaskComments([])
    setTaskAssignees([])
    setSubtasks([])
    // edit 模式且有编辑权限时，打开即进入编辑状态
    setIsEditing(isEditMode && canEditScrum)
    setNewComment('')
    setSprintInfo(null)
    setStoryInfo(null)
    setStatusHistory([])
    setTaskCommits([])
    setTaskCommitsLoading(true)

    Promise.all([
      api.getTask(projectSlug, taskId).then((res: TaskDetailResponse) => {
        const task = res.task || res
        // 合并 sideload: workLogs + actualHours（后端在 fiber.Map 顶层返回）
        setTaskDetail({ ...task, workLogs: res.workLogs || [], actualHours: res.actualHours || 0 })
        // edit 模式且有编辑权限：数据加载完成后回填表单并进入编辑
        if (isEditMode && canEditScrum) {
          setEditForm({
            title: task.title || '',
            description: task.description || '',
            status: task.status || '',
            priority: task.priority || 'medium',
            taskType: task.taskType || 'task',
            storyPoints: task.storyPoints?.toString() || '',
            estimatedHours: task.estimatedHours != null ? String(task.estimatedHours) : '',
            dueDate: task.dueDate ? task.dueDate.split('T')[0] : '',
          })
          setIsEditing(true)
        }
        // Handle assignees
        if (res.assignees && res.assignees.length > 0) {
          setTaskAssignees(res.assignees.map((a) => ({
            userName: a.userName,
            fullName: a.fullName || a.userName,
            image: a.image || null,
          })))
        } else if (task.assigneeName) {
          setTaskAssignees([{ userName: task.assigneeName, fullName: task.assigneeName, image: null }])
        }
        // Load breadcrumb context: sprint + story
        if (task.sprintSlug) {
          api.getSprint(task.sprintSlug).then(setSprintInfo).catch(() => setSprintInfo(null))
        }
        if (task.userStorySlug) {
          api.getUserStory(task.userStorySlug).then((story) => {
            setStoryInfo(story)
            // If task has no sprint but story does, use story's sprint for breadcrumb
            if (!task.sprintSlug && story.sprintSlug) {
              api.getSprint(story.sprintSlug).then(setSprintInfo).catch(() => {})
            }
          }).catch(() => setStoryInfo(null))
        }
      }),
      api.getTaskComments(projectSlug, taskId).then((data) => setTaskComments(data || [])).catch(() => {}),
      api.getSubtasks(projectSlug, taskId).then((data) => setSubtasks(data || [])).catch(() => {}),
      api.getActivities(projectSlug, 'task', taskId).then((data) => {
        setActivities(data?.activities || [])
        setActivityParticipants(data?.participants || [])
      }).catch(() => {
        setActivities([])
        setActivityParticipants([])
      }),
      api.getTaskStatusHistory(projectSlug, taskId).then((data) => setStatusHistory(data || [])).catch(() => {}),
      api.getTaskCommits(projectSlug, taskId).then((res) => setTaskCommits(res?.branches || [])).catch(() => {}).finally(() => setTaskCommitsLoading(false)),
    ]).finally(() => setTaskDetailLoading(false))
  }, [taskId, projectSlug, isCreateMode, canEditScrum, isEditMode])

  // Sync editing state when edit-mode / permission changes (without reloading task data)
  useEffect(() => {
    if (taskDetail && taskDetail.taskId) {
      setIsEditing(canEditScrum || isEditMode)
    }
  }, [taskDetail, canEditScrum, isEditMode])

  // Load breadcrumb context in create mode (from props)
  useEffect(() => {
    if (!isCreateMode) return
    setSprintInfo(null)
    setStoryInfo(null)
    setCreateUserStorySlug(userStorySlug || '')
    if (sprintSlug) {
      api.getSprint(sprintSlug).then(setSprintInfo).catch(() => setSprintInfo(null))
    }
    if (userStorySlug) {
      api.getUserStory(userStorySlug).then((story) => {
        setStoryInfo(story)
        if (!sprintSlug && story.sprintSlug) {
          api.getSprint(story.sprintSlug).then(setSprintInfo).catch(() => {})
        }
      }).catch(() => setStoryInfo(null))
    }
  }, [isCreateMode, sprintSlug, userStorySlug])

  // 加载 sprint / user story 列表供关联选择
  useEffect(() => {
    if (!projectSlug) return
    api.getSprints(projectSlug).then((data) => setAllSprints(data || [])).catch(() => {})
    api.getUserStories(projectSlug).then((data) => setAllUserStories(data || [])).catch(() => {})
  }, [projectSlug])

  function getStatusName(slug: string): string {
    if (!slug) return ''
    const camelSlug = slug.split('_').map(part => part.charAt(0).toUpperCase() + part.slice(1)).join('')
    const key = `pms.taskStatus${camelSlug}`
    const translated = t(key)
    return translated === key ? slug : translated
  }

  function getStatusHex(slug: string): string | undefined {
    return taskStatuses.find(s => s.slug === slug)?.color
  }

  function formatDate(dateStr: string): string {
    const d = new Date(dateStr)
    if (isNaN(d.getTime())) return dateStr
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  }

  function formatDateTime(dateStr: string): string {
    const d = new Date(dateStr)
    if (isNaN(d.getTime())) return dateStr
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
  }

  function isOverdue(dueDate: string): boolean {
    const d = new Date(dueDate)
    if (isNaN(d.getTime())) return false
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    d.setHours(0, 0, 0, 0)
    return d < today
  }

  // --- Inline quick update handlers ---
  async function handleQuickStatusUpdate(newStatus: string) {
    if (!taskId || !taskDetail || newStatus === taskDetail.status) return
    const oldStatus = taskDetail.status
    setTaskDetail({ ...taskDetail, status: newStatus })
    setUpdatingStatus(true)
    try {
      const res = await api.updateTask(projectSlug, taskId, { status: newStatus }) as TaskUpdateResponse
      const updated = res.task || res
      setTaskDetail(updated)
      onTaskUpdated?.(updated)
      // Refresh status history and activities after status change
      api.getTaskStatusHistory(projectSlug, taskId).then((data) => setStatusHistory(data || [])).catch(() => {})
      api.getActivities(projectSlug, 'task', taskId).then((data) => {
        setActivities(data?.activities || [])
        setActivityParticipants(data?.participants || [])
      }).catch(() => {})
    } catch (error: unknown) {
      setTaskDetail({ ...taskDetail, status: oldStatus })
      const err = error as ApiError
      if (err.reason === 'task_blocked' && err.blockedTasks) {
        const taskList = err.blockedTasks.map((bt: BlockedTask) => `#${bt.taskId} ${bt.title}`).join(', ')
        toast({ title: t('pms.taskBlockedTitle'), description: t('pms.taskBlockedDesc', { tasks: taskList }), status: 'warning', duration: 5000 })
      } else {
        toast({ title: t('pms.updateFailed'), status: 'error', duration: 3000 })
      }
    } finally {
      setUpdatingStatus(false)
    }
  }

  async function handleQuickPriorityUpdate(newPriority: string) {
    if (!taskId || !taskDetail || newPriority === taskDetail.priority) return
    const oldPriority = taskDetail.priority
    setTaskDetail({ ...taskDetail, priority: newPriority })
    setUpdatingPriority(true)
    try {
      const res = await api.updateTask(projectSlug, taskId, { priority: newPriority }) as TaskUpdateResponse
      const updated = res.task || res
      setTaskDetail(updated)
      onTaskUpdated?.(updated)
    } catch {
      setTaskDetail({ ...taskDetail, priority: oldPriority })
      toast({ title: t('pms.updateFailed'), status: 'error', duration: 3000 })
    } finally {
      setUpdatingPriority(false)
    }
  }

  async function handleQuickAssigneesUpdate(newAssigneesOrUpdater: { userName: string; fullName: string; image: string | null }[] | ((prev: { userName: string; fullName: string; image: string | null }[]) => { userName: string; fullName: string; image: string | null }[])) {
    if (!taskId) return
    // 兼容 TaskAssignees 组件内的函数式调用 setAssignees((prev) => ...) 与直接传数组的调用
    const newAssignees = typeof newAssigneesOrUpdater === 'function'
      ? newAssigneesOrUpdater(taskAssignees)
      : newAssigneesOrUpdater
    const oldAssignees = taskAssignees
    setTaskAssignees(newAssignees)
    setUpdatingAssignees(true)
    try {
      const res = await api.updateTask(projectSlug, taskId, {
        assignees: newAssignees.map(a => a.userName),
      }) as TaskUpdateResponse
      if (res.assignees) {
        setTaskAssignees(res.assignees.map(a => ({
          userName: a.userName,
          fullName: a.fullName || a.userName,
          image: a.image || null,
        })))
      }
      const updated = res.task || res
      onTaskUpdated?.({ ...updated, assignees: res.assignees || newAssignees })
    } catch {
      setTaskAssignees(oldAssignees)
      toast({ title: t('pms.updateFailed'), status: 'error', duration: 3000 })
    } finally {
      setUpdatingAssignees(false)
    }
  }

  async function handleQuickSprintUpdate(newSprintSlug: string) {
    if (!taskId || !taskDetail || newSprintSlug === (taskDetail.sprintSlug || '')) return
    const oldSprintSlug = taskDetail.sprintSlug
    setTaskDetail({ ...taskDetail, sprintSlug: newSprintSlug })
    setUpdatingSprint(true)
    try {
      const res = await api.updateTask(projectSlug, taskId, { sprintSlug: newSprintSlug }) as TaskUpdateResponse
      const updated = res.task || res
      setTaskDetail(updated)
      onTaskUpdated?.(updated)
      // 同步面包屑上下文
      if (newSprintSlug) {
        api.getSprint(newSprintSlug).then(setSprintInfo).catch(() => setSprintInfo(null))
      } else {
        setSprintInfo(null)
      }
      // 刷新动态（关联操作会产生 activity 记录）
      api.getActivities(projectSlug, 'task', taskId).then((data) => {
        setActivities(data?.activities || [])
        setActivityParticipants(data?.participants || [])
      }).catch(() => {})
    } catch {
      setTaskDetail({ ...taskDetail, sprintSlug: oldSprintSlug })
      toast({ title: t('pms.updateFailed'), status: 'error', duration: 3000 })
    } finally {
      setUpdatingSprint(false)
    }
  }

  async function handleQuickUserStoryUpdate(newUserStorySlug: string) {
    if (!taskId || !taskDetail || newUserStorySlug === (taskDetail.userStorySlug || '')) return
    const oldUserStorySlug = taskDetail.userStorySlug
    setTaskDetail({ ...taskDetail, userStorySlug: newUserStorySlug })
    setUpdatingStory(true)
    try {
      const res = await api.updateTask(projectSlug, taskId, { userStorySlug: newUserStorySlug }) as TaskUpdateResponse
      const updated = res.task || res
      setTaskDetail(updated)
      onTaskUpdated?.(updated)
      // 同步面包屑上下文
      if (newUserStorySlug) {
        api.getUserStory(newUserStorySlug).then((story) => {
          setStoryInfo(story)
          // 如果任务本身没有 sprint 但 story 有，则同步 story 的 sprint 到面包屑
          if (!taskDetail.sprintSlug && story.sprintSlug) {
            api.getSprint(story.sprintSlug).then(setSprintInfo).catch(() => {})
          }
        }).catch(() => setStoryInfo(null))
      } else {
        setStoryInfo(null)
      }
      api.getActivities(projectSlug, 'task', taskId).then((data) => {
        setActivities(data?.activities || [])
        setActivityParticipants(data?.participants || [])
      }).catch(() => {})
    } catch {
      setTaskDetail({ ...taskDetail, userStorySlug: oldUserStorySlug })
      toast({ title: t('pms.updateFailed'), status: 'error', duration: 3000 })
    } finally {
      setUpdatingStory(false)
    }
  }

  async function handleSubmitComment() {
    if (!newComment.trim() || !taskId) return
    setSubmittingComment(true)
    try {
      const comment = await api.createTaskComment(projectSlug, taskId, newComment)
      setTaskComments([...taskComments, comment])
      setNewComment('')
      toast({ title: t('pms.commentAddedSuccess'), status: 'success', duration: 2000 })
    } catch (error: unknown) {
      const err = error as ApiError
      toast({ title: t('pms.addCommentFailed'), description: err.message, status: 'error', duration: 3000 })
    } finally {
      setSubmittingComment(false)
    }
  }

  function handleStartEdit() {
    if (!taskDetail) return
    setEditForm({
      title: taskDetail.title || '',
      description: taskDetail.description || '',
      status: taskDetail.status || '',
      priority: taskDetail.priority || 'medium',
      taskType: taskDetail.taskType || 'task',
      storyPoints: taskDetail.storyPoints?.toString() || '',
      estimatedHours: taskDetail.estimatedHours != null ? String(taskDetail.estimatedHours) : '',
      dueDate: taskDetail.dueDate ? taskDetail.dueDate.split('T')[0] : '',
    })
    setIsEditing(true)
  }

  // 应用 AI 优化任务结果到当前 editForm（新建/编辑共用）
  function handleAIOptimizeApply(result: AIOptimizeTaskResult) {
    setEditForm(prev => ({
      ...prev,
      title: result.title,
      description: result.description,
      priority: result.priority,
      taskType: result.taskType,
      storyPoints: result.storyPoints ? String(result.storyPoints) : '',
    }))
  }

  async function handleSaveEdit() {
    if (isCreateMode) {
      if (!editForm.title.trim()) {
        toast({ title: t('pms.inputTitle'), status: 'error', duration: 2000 })
        return
      }
      setSavingEdit(true)
      try {
        // Sprint 继承:未显式指定 Sprint 时,继承所选故事的 Sprint
        const selectedStory = allUserStories.find(s => s.slug === createUserStorySlug)
        const effectiveSprintSlug = sprintSlug || selectedStory?.sprintSlug || undefined
        const task = await api.createTask(projectSlug, {
          title: editForm.title.trim(),
          description: editForm.description.trim() || '',
          status: editForm.status,
          priority: editForm.priority,
          taskType: editForm.taskType,
          storyPoints: editForm.storyPoints ? parseInt(editForm.storyPoints) : 0,
          estimatedHours: editForm.estimatedHours ? parseFloat(editForm.estimatedHours) : undefined,
          assigneeName: taskAssignees.length > 0 ? taskAssignees[0].userName : undefined,
          sprintSlug: effectiveSprintSlug,
          userStorySlug: createUserStorySlug || undefined,
          dueDate: editForm.dueDate || undefined,
        })
        onTaskCreated?.(task)
        onClose()
        toast({ title: t('pms.taskCreatedSuccess'), status: 'success', duration: 2000 })
      } catch (error: unknown) {
        const err = error as ApiError
        toast({ title: t('pms.createFailed'), description: err.message, status: 'error', duration: 3000 })
      } finally {
        setSavingEdit(false)
      }
      return
    }

    if (!taskId || !editForm.title.trim()) {
      toast({ title: t('pms.inputTitle'), status: 'error', duration: 2000 })
      return
    }
    setSavingEdit(true)
    try {
      const res = await api.updateTask(projectSlug, taskId, {
        title: editForm.title.trim(),
        description: editForm.description.trim() || null,
        status: editForm.status,
        priority: editForm.priority,
        taskType: editForm.taskType,
        assignees: taskAssignees.map(a => a.userName),
        storyPoints: editForm.storyPoints ? parseInt(editForm.storyPoints) : null,
        estimatedHours: editForm.estimatedHours ? parseFloat(editForm.estimatedHours) : null,
        dueDate: editForm.dueDate || '',
      }) as TaskUpdateResponse
      const task = res.task || res
      // 保留 workLogs/actualHours（updateTask 不返回这些 sideload）
      setTaskDetail({
        ...task,
        workLogs: taskDetail?.workLogs || [],
        actualHours: taskDetail?.actualHours || 0
      } as Task)
      if (res.assignees) {
        setTaskAssignees(res.assignees.map(a => ({
          userName: a.userName,
          fullName: a.fullName || a.userName,
          image: a.image || null,
        })))
      }
      onTaskUpdated?.({ ...task, assignees: res.assignees || [] })
      setIsEditing(false)
      // edit 模式（列表页直接编辑）：保存成功后关闭抽屉
      if (isEditMode) {
        onClose()
        return
      }
      // Refresh status history and activities after edit
      api.getTaskStatusHistory(projectSlug, taskId).then((data) => setStatusHistory(data || [])).catch(() => {})
      api.getActivities(projectSlug, 'task', taskId).then((data) => {
        setActivities(data?.activities || [])
        setActivityParticipants(data?.participants || [])
      }).catch(() => [])
    } catch (error: unknown) {
      const err = error as ApiError
      if (err.reason === 'task_blocked' && err.blockedTasks) {
        const taskList = err.blockedTasks.map((bt: BlockedTask) => `#${bt.taskId} ${bt.title}`).join(', ')
        toast({ title: t('pms.taskBlockedTitle'), description: t('pms.taskBlockedDesc', { tasks: taskList }), status: 'warning', duration: 5000 })
      } else {
        toast({ title: t('pms.updateFailed'), description: err.message, status: 'error', duration: 3000 })
      }
    } finally {
      setSavingEdit(false)
    }
  }

  function handleCancelEdit() {
    setIsEditing(false)
  }

  // --- Work Log handlers (Time Tracking) ---
  async function handleAddWorkLog(hours: number, description: string) {
    if (!taskId) return
    setAddingWorkLog(true)
    try {
      await api.createWorkLog(projectSlug, taskId, { hours, description })
      // 刷新 taskDetail 以获取最新 workLogs + actualHours
      const res = await api.getTask(projectSlug, taskId) as TaskDetailResponse
      const task = res.task || res
      setTaskDetail({ ...task, workLogs: res.workLogs || [], actualHours: res.actualHours || 0 })
    } catch (error: unknown) {
      const err = error as ApiError
      toast({ title: t('pms.addWorkLogFailed'), description: err.message, status: 'error', duration: 3000 })
    } finally {
      setAddingWorkLog(false)
    }
  }

  async function handleDeleteWorkLog(logId: number) {
    if (!taskId) return
    try {
      await api.deleteWorkLog(logId)
      const res = await api.getTask(projectSlug, taskId) as TaskDetailResponse
      const task = res.task || res
      setTaskDetail({ ...task, workLogs: res.workLogs || [], actualHours: res.actualHours || 0 })
    } catch (error: unknown) {
      const err = error as ApiError
      toast({ title: t('pms.deleteWorkLogFailed'), description: err.message, status: 'error', duration: 3000 })
    }
  }

  // --- Subtask handlers ---
  async function handleCreateSubtask() {
    if (!newSubtaskTitle.trim() || !taskId) return
    setSubtaskLoading(true)
    try {
      const subtask = await api.createSubtask(projectSlug, taskId, { title: newSubtaskTitle.trim() })
      setSubtasks((prev) => [subtask, ...prev])
      setNewSubtaskTitle('')
      setAddingSubtask(false)
    } catch (error: unknown) {
      const err = error as ApiError
      toast({ title: t('pms.createFailed'), description: err.message, status: 'error', duration: 3000 })
    } finally {
      setSubtaskLoading(false)
    }
  }

  async function handleToggleSubtask(subtask: Task) {
    if (!taskId) return
    const isDone = subtask.status === 'done' || subtask.status === 'completed' || subtask.status === 'closed' || subtask.status === 'archived'
    const newStatus = isDone ? 'todo' : 'done'
    setSubtasks((prev) => prev.map((s) => s.taskId === subtask.taskId ? { ...s, status: newStatus } : s))
    try {
      await api.updateTask(projectSlug, subtask.taskId, { status: newStatus })
    } catch {
      setSubtasks((prev) => prev.map((s) => s.taskId === subtask.taskId ? { ...s, status: subtask.status } : s))
      toast({ title: t('pms.updateFailed'), status: 'error', duration: 3000 })
    }
  }

  async function handleDeleteSubtask(subtaskId: number) {
    if (!taskId) return
    setSubtasks((prev) => prev.filter((s) => s.taskId !== subtaskId))
    try {
      await api.deleteTask(projectSlug, subtaskId)
    } catch {
      api.getSubtasks(projectSlug, taskId).then((data) => setSubtasks(data || [])).catch(() => {})
      toast({ title: t('pms.updateFailed'), status: 'error', duration: 3000 })
    }
  }

  return (
    <>
    <Drawer
      isOpen={isOpen}
      placement="right"
      onClose={onClose}
      size="md"
      blockScrollOnMount={false}
    >
      <DrawerOverlay />
      <DrawerContent>
        <DrawerCloseButton />
        <DrawerHeader borderBottomWidth="1px">
          {isCreateMode ? (
            <Heading size="md">{t('pms.newTask')}</Heading>
          ) : taskDetailLoading ? (
            <Spinner size="sm" />
          ) : taskDetail ? (
            <Flex justify="space-between" align="center">
              <VStack align="start" spacing={2} w="100%">
                <HStack spacing={2} align="baseline">
                  <Text fontSize="sm" color="gray.500" fontWeight="normal">#{taskId}</Text>
                  <Heading size="md">{taskDetail.title}</Heading>
                </HStack>
                <HStack spacing={2}>
                  {/* Priority: clickable Menu for inline edit (member+ only) */}
                  {canEditScrum ? (
                    <Menu>
                      <MenuButton
                        as={Badge}
                        colorScheme={getPriorityColor(taskDetail.priority)}
                        size="sm"
                        cursor="pointer"
                        _hover={{ opacity: 0.8 }}
                      >
                        {taskDetail.priority ? t(getPriorityLabelKey(taskDetail.priority)) : '-'} <Icon as={FiChevronDown} />
                      </MenuButton>
                      <MenuList>
                        {PRIORITIES.map((p) => (
                          <MenuItem
                            key={p}
                            onClick={() => handleQuickPriorityUpdate(p)}
                            isDisabled={updatingPriority}
                          >
                            <HStack spacing={2}>
                              <Badge colorScheme={getPriorityColor(p)} size="sm">{t(getPriorityLabelKey(p))}</Badge>
                              {p === taskDetail.priority && <Text color="green.500">✓</Text>}
                            </HStack>
                          </MenuItem>
                        ))}
                      </MenuList>
                    </Menu>
                  ) : (
                    <Badge colorScheme={getPriorityColor(taskDetail.priority)} size="sm">
                      {taskDetail.priority ? t(getPriorityLabelKey(taskDetail.priority)) : '-'}
                    </Badge>
                  )}
                  <Badge colorScheme="gray" size="sm">
                    {taskDetail.taskType ? t(`pms.type${taskDetail.taskType.charAt(0).toUpperCase() + taskDetail.taskType.slice(1)}`) : '-'}
                  </Badge>
                </HStack>
              </VStack>
              {!isEditing && canEditScrum && (
                <Button size="sm" variant="outline" onClick={handleStartEdit} mr="32px">
                  {t('common.edit')}
                </Button>
              )}
            </Flex>
          ) : null}
        </DrawerHeader>

        <DrawerBody>
          {isCreateMode ? (
            <VStack align="stretch" spacing={4} pt={4}>
              {/* 上下文：显示任务将关联到的 Sprint / Story */}
              {(sprintInfo || storyInfo) && (
                <Box bg="gray.50" borderWidth="1px" borderColor="gray.200" borderRadius="md" px={3} py={2}>
                  <Text fontSize="xs" color="gray.500" mb={1}>{t('pms.associateTo')}</Text>
                  <HStack spacing={2} wrap="wrap">
                    {sprintInfo && (
                      <Badge colorScheme="blue" variant="subtle">
                        <Flex as="span" align="center" gap={1}>
                          <FiRepeat size={10} />
                          {sprintInfo.title}
                        </Flex>
                      </Badge>
                    )}
                    {storyInfo && (
                      <Badge colorScheme="purple" variant="subtle">
                        <Flex as="span" align="center" gap={1}>
                          <FiBook size={10} />
                          {storyInfo.title}
                        </Flex>
                      </Badge>
                    )}
                  </HStack>
                </Box>
              )}
              <Box>
                <Flex justify="space-between" align="center" mb={1}>
                  <Text fontSize="sm" fontWeight="medium">{t('pms.taskTitle')}</Text>
                  <Button
                    size="xs"
                    leftIcon={<FiCpu />}
                    colorScheme="purple"
                    variant="outline"
                    onClick={() => setIsAIOptimizeOpen(true)}
                    isDisabled={!editForm.title.trim()}
                  >
                    {t('pms.aiOptimize')}
                  </Button>
                </Flex>
                <Input
                  value={editForm.title}
                  onChange={(e) => setEditForm({ ...editForm, title: e.target.value })}
                  size="sm"
                  placeholder={t('pms.taskTitlePlaceholder')}
                />
              </Box>
              <Box>
                <Text fontSize="sm" fontWeight="medium" mb={1}>{t('common.description')}</Text>
                <Textarea
                  value={editForm.description}
                  onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
                  rows={4}
                  size="sm"
                  placeholder={t('pms.taskDescriptionPlaceholder')}
                />
              </Box>
              <HStack spacing={4}>
                <Box flex={1}>
                  <Text fontSize="sm" fontWeight="medium" mb={1}>{t('common.status')}</Text>
                  <Select
                    value={editForm.status}
                    onChange={(e) => setEditForm({ ...editForm, status: e.target.value })}
                    size="sm"
                  >
                    {taskStatuses.length > 0 ? (
                      taskStatuses.map((status) => (
                        <option key={status.id} value={status.slug}>
                          {getStatusName(status.slug)}
                        </option>
                      ))
                    ) : (
                      <>
                        <option value="todo">{t('pms.taskStatusTodo')}</option>
                        <option value="in_progress">{t('pms.taskStatusInProgress')}</option>
                        <option value="review">{t('pms.taskStatusReview')}</option>
                        <option value="done">{t('pms.taskStatusDone')}</option>
                      </>
                    )}
                  </Select>
                </Box>
                <Box flex={1}>
                  <Text fontSize="sm" fontWeight="medium" mb={1}>{t('common.priority')}</Text>
                  <Select
                    value={editForm.priority}
                    onChange={(e) => setEditForm({ ...editForm, priority: e.target.value })}
                    size="sm"
                  >
                    <option value="urgent">{t('pms.urgent')}</option>
                    <option value="high">{t('pms.high')}</option>
                    <option value="medium">{t('pms.medium')}</option>
                    <option value="low">{t('pms.low')}</option>
                  </Select>
                </Box>
              </HStack>
              <HStack spacing={4}>
                <Box flex={1}>
                  <Text fontSize="sm" fontWeight="medium" mb={1}>{t('pms.taskType')}</Text>
                  <Select
                    value={editForm.taskType}
                    onChange={(e) => setEditForm({ ...editForm, taskType: e.target.value })}
                    size="sm"
                  >
                    <option value="task">{t('pms.typeTask')}</option>
                    <option value="bug">{t('pms.typeBug')}</option>
                    <option value="feature">{t('pms.typeFeature')}</option>
                    <option value="improvement">{t('pms.typeImprovement')}</option>
                  </Select>
                </Box>
                <Box flex={1}>
                  <Text fontSize="sm" fontWeight="medium" mb={1}>{t('pms.assignees')}</Text>
                  <TaskAssignees
                    assignees={taskAssignees}
                    setAssignees={setTaskAssignees}
                    searchKeyword={userSearchKeyword}
                    setSearchKeyword={setUserSearchKeyword}
                    searchResults={userSearchResults}
                    setSearchResults={setUserSearchResults}
                    searchLoading={userSearchLoading}
                    setSearchLoading={setUserSearchLoading}
                    isPopoverOpen={isAssigneePopoverOpen}
                    setIsPopoverOpen={setIsAssigneePopoverOpen}
                  />
                </Box>
              </HStack>
              <Box>
                <Text fontSize="sm" fontWeight="medium" mb={1}>{t('pms.userStory')}</Text>
                <Select
                  value={createUserStorySlug}
                  onChange={(e) => {
                    const newSlug = e.target.value
                    setCreateUserStorySlug(newSlug)
                    // 选中故事后同步上下文区故事徽标;并实现 Sprint 继承
                    // (选中带 Sprint 的故事且未显式指定 Sprint 时,任务继承故事 Sprint)
                    const selected = allUserStories.find(s => s.slug === newSlug)
                    if (selected) {
                      setStoryInfo(selected)
                      if (!sprintSlug && selected.sprintSlug) {
                        api.getSprint(selected.sprintSlug).then(setSprintInfo).catch(() => {})
                      }
                    } else {
                      setStoryInfo(null)
                    }
                  }}
                  size="sm"
                >
                  <option value="">{t('pms.noUserStory')}</option>
                  {allUserStories.map(story => (
                    <option key={story.slug} value={story.slug}>{story.title}</option>
                  ))}
                </Select>
              </Box>
              <Box>
                <Text fontSize="sm" fontWeight="medium" mb={1}>{t('pms.storyPoints')}</Text>
                <Input
                  type="number"
                  value={editForm.storyPoints}
                  onChange={(e) => setEditForm({ ...editForm, storyPoints: e.target.value })}
                  placeholder="0"
                  size="sm"
                  w="100px"
                />
              </Box>
              <Box>
                <Text fontSize="sm" fontWeight="medium" mb={1}>{t('pms.estimatedHours')}</Text>
                <Input
                  type="number"
                  step="0.5"
                  value={editForm.estimatedHours}
                  onChange={(e) => setEditForm({ ...editForm, estimatedHours: e.target.value })}
                  placeholder="0"
                  size="sm"
                  w="100px"
                />
              </Box>
              <Box>
                <Text fontSize="sm" fontWeight="medium" mb={1}>{t('pms.dueDate')}</Text>
                <Input
                  type="date"
                  value={editForm.dueDate}
                  onChange={(e) => setEditForm({ ...editForm, dueDate: e.target.value })}
                  size="sm"
                />
              </Box>
              <HStack spacing={3} pt={2} justify="flex-end">
                <Button size="sm" variant="outline" onClick={onClose}>
                  {t('common.cancel')}
                </Button>
                <Button colorScheme="blue" size="sm" onClick={handleSaveEdit} isLoading={savingEdit}>
                  {t('common.create')}
                </Button>
              </HStack>
            </VStack>
          ) : taskDetailLoading ? (
            <Flex justify="center" align="center" h="200px">
              <Spinner size="xl" />
            </Flex>
          ) : taskDetail ? (
            isEditing ? (
              <VStack align="stretch" spacing={4} pt={4}>
                <Box>
                  <Flex justify="space-between" align="center" mb={1}>
                    <Text fontSize="sm" fontWeight="medium">{t('pms.taskTitle')}</Text>
                    <Button
                      size="xs"
                      leftIcon={<FiCpu />}
                      colorScheme="purple"
                      variant="outline"
                      onClick={() => setIsAIOptimizeOpen(true)}
                      isDisabled={!editForm.title.trim()}
                    >
                      {t('pms.aiOptimize')}
                    </Button>
                  </Flex>
                  <Input
                    value={editForm.title}
                    onChange={(e) => setEditForm({ ...editForm, title: e.target.value })}
                    size="sm"
                  />
                </Box>
                <Box>
                  <Text fontSize="sm" fontWeight="medium" mb={1}>{t('common.description')}</Text>
                  <Textarea
                    value={editForm.description}
                    onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
                    rows={4}
                    size="sm"
                  />
                </Box>
                <HStack spacing={4}>
                  <Box flex={1}>
                    <Text fontSize="sm" fontWeight="medium" mb={1}>{t('common.status')}</Text>
                    <Select
                      value={editForm.status}
                      onChange={(e) => setEditForm({ ...editForm, status: e.target.value })}
                      size="sm"
                    >
                      {taskStatuses.map((status) => (
                        <option key={status.id} value={status.slug}>
                          {getStatusName(status.slug)}
                        </option>
                      ))}
                    </Select>
                  </Box>
                  <Box flex={1}>
                    <Text fontSize="sm" fontWeight="medium" mb={1}>{t('common.priority')}</Text>
                    <Select
                      value={editForm.priority}
                      onChange={(e) => setEditForm({ ...editForm, priority: e.target.value })}
                      size="sm"
                    >
                      <option value="urgent">{t('pms.urgent')}</option>
                      <option value="high">{t('pms.high')}</option>
                      <option value="medium">{t('pms.medium')}</option>
                      <option value="low">{t('pms.low')}</option>
                    </Select>
                  </Box>
                </HStack>
                <HStack spacing={4}>
                  <Box flex={1}>
                    <Text fontSize="sm" fontWeight="medium" mb={1}>{t('pms.taskType')}</Text>
                    <Select
                      value={editForm.taskType}
                      onChange={(e) => setEditForm({ ...editForm, taskType: e.target.value })}
                      size="sm"
                    >
                      <option value="task">{t('pms.typeTask')}</option>
                      <option value="bug">{t('pms.typeBug')}</option>
                      <option value="feature">{t('pms.typeFeature')}</option>
                      <option value="improvement">{t('pms.typeImprovement')}</option>
                    </Select>
                  </Box>
                  <Box flex={1}>
                    <Text fontSize="sm" fontWeight="medium" mb={1}>{t('pms.assignees')}</Text>
                    <TaskAssignees
                      assignees={taskAssignees}
                      setAssignees={setTaskAssignees}
                      searchKeyword={userSearchKeyword}
                      setSearchKeyword={setUserSearchKeyword}
                      searchResults={userSearchResults}
                      setSearchResults={setUserSearchResults}
                      searchLoading={userSearchLoading}
                      setSearchLoading={setUserSearchLoading}
                      isPopoverOpen={isAssigneePopoverOpen}
                      setIsPopoverOpen={setIsAssigneePopoverOpen}
                    />
                  </Box>
                </HStack>
                <Box>
                  <Text fontSize="sm" fontWeight="medium" mb={1}>{t('pms.storyPoints')}</Text>
                  <Input
                    type="number"
                    value={editForm.storyPoints}
                    onChange={(e) => setEditForm({ ...editForm, storyPoints: e.target.value })}
                    placeholder="0"
                    size="sm"
                    w="100px"
                  />
                </Box>
                <Box>
                  <Text fontSize="sm" fontWeight="medium" mb={1}>{t('pms.estimatedHours')}</Text>
                  <Input
                    type="number"
                    step="0.5"
                    value={editForm.estimatedHours}
                    onChange={(e) => setEditForm({ ...editForm, estimatedHours: e.target.value })}
                    placeholder="0"
                    size="sm"
                    w="100px"
                  />
                </Box>
                <Box>
                  <Text fontSize="sm" fontWeight="medium" mb={1}>{t('pms.dueDate')}</Text>
                  <Input
                    type="date"
                    value={editForm.dueDate}
                    onChange={(e) => setEditForm({ ...editForm, dueDate: e.target.value })}
                    size="sm"
                  />
                </Box>
                <HStack spacing={3} pt={2} justify="flex-end">
                  <Button size="sm" variant="outline" onClick={handleCancelEdit}>
                    {t('common.cancel')}
                  </Button>
                  <Button colorScheme="blue" size="sm" onClick={handleSaveEdit} isLoading={savingEdit}>
                    {t('common.save')}
                  </Button>
                </HStack>
              </VStack>
            ) : (
              <VStack align="stretch" spacing={6} pt={4}>
                {/* Status: inline button group for quick switch (member+ only) */}
                <Box>
                  <Text fontSize="xs" fontWeight="medium" color="gray.500" mb={2}>{t('common.status')}</Text>
                  {canEditScrum ? (
                    <ButtonGroup size="sm" spacing={1} flexWrap="wrap">
                      {(taskStatuses.length > 0 ? taskStatuses : [
                        { slug: 'todo', color: '#a0aec0' }, 
                        { slug: 'in_progress', color: '#4299e1' }, 
                        { slug: 'review', color: '#ed8936' }, 
                        { slug: 'done', color: '#48bb7b' }
                      ]).map((status) => {
                        const isActive = taskDetail?.status === status.slug
                        const dotColor = status.color || '#a0aec0'
                        return (
                          <Button
                            key={status.slug}
                            size="sm"
                            variant={isActive ? 'solid' : 'outline'}
                            bg={isActive ? dotColor : 'transparent'}
                            color={isActive ? 'white' : 'gray.600'}
                            borderColor={isActive ? dotColor : 'gray.200'}
                            opacity={updatingStatus ? 0.6 : 1}
                            _hover={isActive ? { bg: dotColor, opacity: updatingStatus ? 0.6 : 0.9 } : { bg: 'gray.50' }}
                            _active={isActive ? { bg: dotColor } : {}}
                            onClick={() => handleQuickStatusUpdate(status.slug)}
                            isDisabled={updatingStatus}
                          >
                            <Box w="6px" h="6px" borderRadius="full" bg={isActive ? 'white' : dotColor} mr="6px" />
                            {getStatusName(status.slug)}
                          </Button>
                        )
                      })}
                    </ButtonGroup>
                  ) : (
                    <Badge colorScheme="gray" size="sm">{getStatusName(taskDetail.status)}</Badge>
                  )}
                </Box>

                {/* Assignees: inline edit with search (member+ only) */}
                <Box>
                  <Text fontSize="xs" fontWeight="medium" color="gray.500" mb={2}>{t('pms.assignees')}</Text>
                  {updatingAssignees ? (
                    <Spinner size="xs" />
                  ) : canEditScrum ? (
                    <>
                      <TaskAssignees
                        assignees={taskAssignees}
                        setAssignees={handleQuickAssigneesUpdate}
                        searchKeyword={userSearchKeyword}
                        setSearchKeyword={setUserSearchKeyword}
                        searchResults={userSearchResults}
                        setSearchResults={setUserSearchResults}
                        searchLoading={userSearchLoading}
                        setSearchLoading={setUserSearchLoading}
                        isPopoverOpen={isAssigneePopoverOpen}
                        setIsPopoverOpen={setIsAssigneePopoverOpen}
                      />
                      {currentUser && !taskAssignees.some(a => a.userName === currentUser.userName) && (
                        <Button
                          size="xs"
                          variant="link"
                          colorScheme="blue"
                          onClick={() => handleQuickAssigneesUpdate([...taskAssignees, {
                            userName: currentUser.userName,
                            fullName: currentUser.fullName || currentUser.userName,
                            image: currentUser.image || null,
                          }])}
                        >
                          {t('pms.assignMe')}
                        </Button>
                      )}
                    </>
                  ) : (
                    <HStack spacing={1}>
                      {taskAssignees.length === 0 ? (
                        <Text fontSize="sm" color="gray.400">-</Text>
                      ) : taskAssignees.map((a) => (
                        <Avatar key={a.userName} size="xs" name={a.fullName || a.userName} src={a.image || undefined} />
                      ))}
                    </HStack>
                  )}
                </Box>

                {/* Sprint: inline 关联（member+ only） */}
                <Box>
                  <Text fontSize="xs" fontWeight="medium" color="gray.500" mb={2}>{t('pms.sprint')}</Text>
                  {updatingSprint ? (
                    <Spinner size="xs" />
                  ) : canEditScrum ? (
                    <Popover isOpen={isSprintPopoverOpen} onClose={() => setIsSprintPopoverOpen(false)} placement="bottom">
                      <PopoverTrigger>
                        {taskDetail.sprintSlug ? (
                          <Button size="sm" variant="subtle" colorScheme="blue" leftIcon={<FiCalendar size={12} />}
                            onClick={() => setIsSprintPopoverOpen(true)}>
                            {allSprints.find(s => s.slug === taskDetail.sprintSlug)?.title || taskDetail.sprintSlug}
                          </Button>
                        ) : (
                          <Button size="sm" variant="outline" colorScheme="gray" leftIcon={<FiCalendar size={12} />}
                            onClick={() => setIsSprintPopoverOpen(true)}>
                            {t('pms.assignToSprint')}
                          </Button>
                        )}
                      </PopoverTrigger>
                      <PopoverContent w="240px">
                        <PopoverHeader fontWeight="semibold" fontSize="sm">{t('pms.assignToSprint')}</PopoverHeader>
                        <PopoverBody py={2}>
                          <Stack spacing={1}>
                            {/* 已关联时提供跳转到 Sprint 详情入口 */}
                            {taskDetail.sprintSlug && (
                              <NextLink href={`/projects/${projectSlug}/sprint/${taskDetail.sprintSlug}`} style={{ textDecoration: 'none' }}>
                                <HStack spacing={2} fontSize="xs" color="blue.600" px={2} py={1} cursor="pointer" borderRadius="sm" _hover={{ bg: 'blue.50' }}>
                                  <FiExternalLink size={11} />
                                  <Text>{t('pms.viewSprint')}</Text>
                                </HStack>
                              </NextLink>
                            )}
                            {taskDetail.sprintSlug && (
                              <Box borderTop="1px solid" borderColor="gray.100" my={1} />
                            )}
                            {allSprints.filter(s => s.status === 'active' || s.status === 'open').map(sprint => (
                              <Button key={sprint.slug} size="xs" variant="ghost" justifyContent="flex-start"
                                onClick={() => { handleQuickSprintUpdate(sprint.slug); setIsSprintPopoverOpen(false) }}>
                                {sprint.title}
                              </Button>
                            ))}
                            {allSprints.filter(s => s.status === 'active' || s.status === 'open').length === 0 && (
                              <Text fontSize="xs" color="gray.500" px={2}>{t('pms.noActiveSprint')}</Text>
                            )}
                            {taskDetail.sprintSlug && (
                              <>
                                <Box borderTop="1px solid" borderColor="gray.100" my={1} />
                                <Button size="xs" variant="ghost" colorScheme="red" justifyContent="flex-start"
                                  leftIcon={<FiX size={12} />}
                                  onClick={() => { handleQuickSprintUpdate(''); setIsSprintPopoverOpen(false) }}>
                                  {t('pms.moveToBacklog')}
                                </Button>
                              </>
                            )}
                          </Stack>
                        </PopoverBody>
                      </PopoverContent>
                    </Popover>
                  ) : (
                    <Text fontSize="sm" color="gray.600">
                      {taskDetail.sprintSlug
                        ? (allSprints.find(s => s.slug === taskDetail.sprintSlug)?.title || t('pms.noSprint'))
                        : t('pms.noSprint')}
                    </Text>
                  )}
                </Box>

                {/* User Story: inline 关联（member+ only） */}
                <Box>
                  <Text fontSize="xs" fontWeight="medium" color="gray.500" mb={2}>{t('pms.userStory')}</Text>
                  {updatingStory ? (
                    <Spinner size="xs" />
                  ) : canEditScrum ? (
                    <Popover isOpen={isStoryPopoverOpen} onClose={() => setIsStoryPopoverOpen(false)} placement="bottom">
                      <PopoverTrigger>
                        {taskDetail.userStorySlug ? (
                          <Button size="sm" variant="subtle" colorScheme="blue" leftIcon={<FiBook size={12} />}
                            onClick={() => setIsStoryPopoverOpen(true)}>
                            {allUserStories.find(s => s.slug === taskDetail.userStorySlug)?.title || taskDetail.userStorySlug}
                          </Button>
                        ) : (
                          <Button size="sm" variant="outline" colorScheme="gray" leftIcon={<FiBook size={12} />}
                            onClick={() => setIsStoryPopoverOpen(true)}>
                            {t('pms.assignToUserStory')}
                          </Button>
                        )}
                      </PopoverTrigger>
                      <PopoverContent w="280px">
                        <PopoverHeader fontWeight="semibold" fontSize="sm">{t('pms.selectUserStory')}</PopoverHeader>
                        <PopoverBody py={2} maxH="300px" overflowY="auto">
                          <Stack spacing={1}>
                            {/* 已关联时提供跳转到 User Story 详情入口 */}
                            {taskDetail.userStorySlug && (
                              <NextLink href={`/projects/${projectSlug}/story/${taskDetail.userStorySlug}`} style={{ textDecoration: 'none' }}>
                                <HStack spacing={2} fontSize="xs" color="blue.600" px={2} py={1} cursor="pointer" borderRadius="sm" _hover={{ bg: 'blue.50' }}>
                                  <FiExternalLink size={11} />
                                  <Text>{t('pms.viewStory')}</Text>
                                </HStack>
                              </NextLink>
                            )}
                            {taskDetail.userStorySlug && (
                              <Box borderTop="1px solid" borderColor="gray.100" my={1} />
                            )}
                            {allUserStories.map(story => (
                              <Button key={story.slug} size="xs" variant="ghost" justifyContent="flex-start"
                                onClick={() => { handleQuickUserStoryUpdate(story.slug); setIsStoryPopoverOpen(false) }}>
                                {story.title}
                              </Button>
                            ))}
                            {allUserStories.length === 0 && (
                              <Text fontSize="xs" color="gray.500" px={2}>{t('pms.noUserStories')}</Text>
                            )}
                            {taskDetail.userStorySlug && (
                              <>
                                <Box borderTop="1px solid" borderColor="gray.100" my={1} />
                                <Button size="xs" variant="ghost" colorScheme="red" justifyContent="flex-start"
                                  leftIcon={<FiX size={12} />}
                                  onClick={() => { handleQuickUserStoryUpdate(''); setIsStoryPopoverOpen(false) }}>
                                  {t('pms.unlinkUserStory')}
                                </Button>
                              </>
                            )}
                          </Stack>
                        </PopoverBody>
                      </PopoverContent>
                    </Popover>
                  ) : (
                    <Text fontSize="sm" color="gray.600">
                      {taskDetail.userStorySlug
                        ? (allUserStories.find(s => s.slug === taskDetail.userStorySlug)?.title || t('pms.noUserStory'))
                        : t('pms.noUserStory')}
                    </Text>
                  )}
                </Box>

                {taskDetail.storyPoints && (
                  <HStack spacing={1} fontSize="sm" color="gray.600">
                    <Icon as={FiTag} />
                    <Text>{taskDetail.storyPoints} {t('pms.storyPoints')}</Text>
                  </HStack>
                )}

                {(taskDetail.estimatedHours != null || (taskDetail.actualHours ?? 0) > 0) && (
                  <HStack spacing={1} fontSize="sm" color="gray.600">
                    <Icon as={FiClock} />
                    <Text>{taskDetail.actualHours ?? 0}h / {taskDetail.estimatedHours ?? 0}h</Text>
                  </HStack>
                )}

                {taskDetail.dueDate && (
                  <HStack spacing={1} fontSize="sm" color={isOverdue(taskDetail.dueDate) ? 'red.500' : 'gray.600'}>
                    <Icon as={FiClock} />
                    <Text>{t('pms.dueDate')}: {formatDate(taskDetail.dueDate)}</Text>
                    {isOverdue(taskDetail.dueDate) && (
                      <Badge colorScheme="red" size="sm" ml={1}>{t('pms.overdue')}</Badge>
                    )}
                  </HStack>
                )}

                {taskDetail.description && (
                  <Box>
                    <Heading size="sm" mb={2}>{t('common.description')}</Heading>
                    <Text whiteSpace="pre-wrap" fontSize="sm">{taskDetail.description}</Text>
                  </Box>
                )}

                {/* Subtasks section */}
                <Box>
                  <HStack justify="space-between" mb={2}>
                    <Heading size="sm">
                      {t('pms.subtasks')}
                      {subtasks.length > 0 && (
                        <Text as="span" fontSize="xs" color="gray.500" ml={2}>
                          {subtasks.filter(s => s.status === 'done' || s.status === 'completed' || s.status === 'closed' || s.status === 'archived').length}/{subtasks.length}
                        </Text>
                      )}
                    </Heading>
                  </HStack>

                  <VStack align="stretch" spacing={1}>
                    {subtasks.map((subtask) => {
                      const isDone = subtask.status === 'done' || subtask.status === 'completed' || subtask.status === 'closed' || subtask.status === 'archived'
                      return (
                        <HStack key={subtask.taskId} spacing={2}>
                          <Checkbox
                            isChecked={isDone}
                            onChange={() => handleToggleSubtask(subtask)}
                            isDisabled={!canEditScrum}
                          />
                          <Text
                            fontSize="sm"
                            flex="1"
                            textDecoration={isDone ? 'line-through' : 'none'}
                            color={isDone ? 'gray.400' : 'inherit'}
                          >
                            {subtask.title}
                          </Text>
                          {subtask.priority && (
                            <Badge
                              size="sm"
                              colorScheme={getPriorityColor(subtask.priority)}
                            >
                              {subtask.priority}
                            </Badge>
                          )}
                          {canEditScrum && (
                            <IconButton
                              aria-label="Delete subtask"
                              icon={<FiTrash2 />}
                              size="xs"
                              variant="ghost"
                              onClick={() => handleDeleteSubtask(subtask.taskId)}
                            />
                          )}
                        </HStack>
                      )
                    })}

                    {canEditScrum && (addingSubtask ? (
                      <HStack spacing={2}>
                        <Input
                          size="sm"
                          placeholder={t('pms.subtaskTitlePlaceholder')}
                          value={newSubtaskTitle}
                          onChange={(e) => setNewSubtaskTitle(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') handleCreateSubtask()
                            if (e.key === 'Escape') { setAddingSubtask(false); setNewSubtaskTitle('') }
                          }}
                          autoFocus
                        />
                        <Button size="sm" colorScheme="blue" onClick={handleCreateSubtask} isLoading={subtaskLoading}>
                          {t('common.add')}
                        </Button>
                        <Button size="sm" variant="outline" onClick={() => { setAddingSubtask(false); setNewSubtaskTitle('') }}>
                          {t('common.cancel')}
                        </Button>
                      </HStack>
                    ) : (
                      <Button
                        size="sm"
                        variant="ghost"
                        colorScheme="blue"
                        leftIcon={<FiPlus />}
                        onClick={() => setAddingSubtask(true)}
                      >
                        {t('pms.addSubtask')}
                      </Button>
                    ))}
                  </VStack>
                </Box>

                {/* Branches: 松耦合关联 Git 分支（关联已有 / 创建新分支） */}
                {taskDetail.taskId && (
                  <TaskBranches
                    projectSlug={projectSlug}
                    taskId={taskDetail.taskId}
                    canEdit={!!canEditScrum}
                  />
                )}

                {/* Dependencies: 有向依赖 + 硬阻塞（前置未完成时拒绝推进到进行中） */}
                {taskDetail.taskId && (
                  <TaskDependencies
                    projectSlug={projectSlug}
                    taskId={taskDetail.taskId}
                    canEdit={!!canEditScrum}
                  />
                )}

                {/* Commits: 关联分支上独有的提交（相对默认分支的 ahead commits） */}
                {taskDetail.taskId && (taskCommits.length > 0 || taskCommitsLoading) && (
                  <Box>
                    <HStack spacing={2} mb={2}>
                      <Icon as={FiGitCommit} color="gray.500" />
                      <Text fontSize="sm" fontWeight="medium" color="gray.700">
                        {t('pms.commits')}
                      </Text>
                      {!taskCommitsLoading && (
                        <Text fontSize="xs" color="gray.500">
                          ({taskCommits.reduce((sum, b) => sum + (b.commits?.length || 0), 0)})
                        </Text>
                      )}
                    </HStack>
                    {taskCommitsLoading ? (
                      <Spinner size="xs" />
                    ) : (
                      <VStack align="stretch" spacing={3}>
                        {taskCommits.map((bc) => {
                          const [owner, repo] = bc.repoFullName.split('/')
                          // 分支名含 "/" 时（如 feature/task-15），需保留 "/" 作为路径分隔符
                          const commitsHref = bc.branchName.split('/').map(encodeURIComponent).join('/')
                          const branchHref = `/${owner}/${repo}/commits/${commitsHref}`
                          if (!bc.commits || bc.commits.length === 0) {
                            return (
                              <Box key={`${bc.repoFullName}:${bc.branchName}`} fontSize="xs" color="gray.400">
                                {bc.repoFullName}:{bc.branchName} · {t('pms.noCommits')}
                              </Box>
                            )
                          }
                          return (
                            <Box key={`${bc.repoFullName}:${bc.branchName}`}>
                              <NextLink href={branchHref} target="_blank">
                                <HStack spacing={1} fontSize="xs" mb={1} _hover={{ color: 'blue.500' }}>
                                  <Text color="gray.500" flexShrink={0}>{bc.repoFullName}</Text>
                                  <Text color="gray.400">:</Text>
                                  <Text fontWeight="medium" color="gray.700" _hover={{ color: 'blue.500' }}>
                                    {bc.branchName}
                                  </Text>
                                </HStack>
                              </NextLink>
                              <VStack align="stretch" spacing={1}>
                                {bc.commits.map((commit) => {
                                  const shortSha = (commit.id || '').slice(0, 7)
                                  const firstLine = (commit.message || '').split('\n')[0] || '(no message)'
                                  return (
                                    <HStack
                                      key={commit.id}
                                      spacing={2}
                                      bg="gray.50"
                                      borderRadius="md"
                                      px={2}
                                      py={1.5}
                                      fontSize="xs"
                                    >
                                      <Text fontFamily="mono" color="blue.600" flexShrink={0} w="60px">
                                        {shortSha}
                                      </Text>
                                      <Text color="gray.700" flex={1} isTruncated>{firstLine}</Text>
                                      <Text color="gray.500" flexShrink={0}>{commit.author}</Text>
                                      <Text color="gray.400" flexShrink={0}>{formatDateTime(commit.time)}</Text>
                                    </HStack>
                                  )
                                })}
                              </VStack>
                            </Box>
                          )
                        })}
                      </VStack>
                    )}
                  </Box>
                )}

                {/* Activity Timeline */}
                <Box>
                  <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={3}>
                    {t('pms.activities')}
                  </Text>
                  <ActivityTimeline activities={activities} taskStatuses={taskStatuses} participants={activityParticipants} />
                </Box>

                {/* Status Change History */}
                {statusHistory.length > 0 && (
                  <Box>
                    <HStack spacing={2} mb={3}>
                      <Icon as={FiActivity} color="gray.500" />
                      <Text fontSize="sm" fontWeight="medium" color="gray.700">
                        {t('pms.statusHistory')}
                      </Text>
                    </HStack>
                    <VStack align="stretch" spacing={2}>
                      {statusHistory.map((item, idx) => (
                        <Box
                          key={item.id || idx}
                          fontSize="sm"
                          bg="gray.50"
                          borderRadius="md"
                          px={3}
                          py={2}
                          borderLeft="3px solid"
                          borderLeftColor={getStatusHex(item.newStatus) || '#a0aec0'}
                        >
                          <HStack spacing={2} flexWrap="wrap">
                            <Badge bg={`${getStatusHex(item.oldStatus) || '#a0aec0'}22`} color={getStatusHex(item.oldStatus) || '#718096'} fontSize="2xs">
                              {getStatusName(item.oldStatus)}
                            </Badge>
                            <Text color="gray.400" fontSize="xs">→</Text>
                            <Badge bg={`${getStatusHex(item.newStatus) || '#a0aec0'}22`} color={getStatusHex(item.newStatus) || '#718096'} fontSize="2xs">
                              {getStatusName(item.newStatus)}
                            </Badge>
                            <Text color="gray.500" fontSize="xs" ml="auto">
                              {item.changedByName} · {formatDateTime(item.createdAt)}
                            </Text>
                          </HStack>
                          {item.comment && (
                            <Text mt={1} color="gray.600" fontSize="xs">{item.comment}</Text>
                          )}
                        </Box>
                      ))}
                    </VStack>
                  </Box>
                )}

                {taskDetail.taskId && (
                  <TaskTimeTracking
                    estimatedHours={taskDetail.estimatedHours ?? null}
                    actualHours={taskDetail.actualHours ?? 0}
                    workLogs={taskDetail.workLogs ?? []}
                    canEdit={!!canEditScrum}
                    onAddLog={handleAddWorkLog}
                    onDeleteLog={handleDeleteWorkLog}
                    addingLog={addingWorkLog}
                  />
                )}

                <TaskComments
                  comments={taskComments}
                  newComment={newComment}
                  onNewCommentChange={setNewComment}
                  onSubmit={handleSubmitComment}
                  submitting={submittingComment}
                  canComment={canEditScrum}
                />
              </VStack>
            )
          ) : null}
        </DrawerBody>
      </DrawerContent>
    </Drawer>
    {/* AI 优化任务草稿弹窗（新建/编辑表单共用） */}
    <AIOptimizeModal
      isOpen={isAIOptimizeOpen}
      onClose={() => setIsAIOptimizeOpen(false)}
      projectSlug={projectSlug}
      type="task"
      draft={{
        title: editForm.title,
        description: editForm.description,
      }}
      onApply={handleAIOptimizeApply}
    />
    </>
  )
}
