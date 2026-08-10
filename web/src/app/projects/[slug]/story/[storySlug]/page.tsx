'use client'

import { useState, useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import {
  Box,
  Button,
  Flex,
  Heading,
  HStack,
  Stack,
  Text,
  Badge,
  Divider,
  Spinner,
  useToast,
  Drawer,
  DrawerOverlay,
  DrawerContent,
  DrawerCloseButton,
  DrawerHeader,
  DrawerBody,
  DrawerFooter,
  Input,
  IconButton,
  VStack,
} from '@chakra-ui/react'
import { FiPlus, FiLink, FiX, FiActivity, FiCpu } from 'react-icons/fi'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import TaskDrawer from '@/components/TaskDrawer'
import UserStoryDrawer, { UserStoryFormData } from '@/components/UserStoryDrawer'
import ActivityTimeline from '@/components/ActivityTimeline'
import TaskProgressCard from '@/components/TaskProgressCard'
import StoryHeader from './components/StoryHeader'
import AIDecomposeModal from './components/AIDecomposeModal'
import { WorkflowProgress } from './components/WorkflowProgress'
import { useProject } from '../../ProjectContext'
import { getStatusHexColor } from '@/lib/taskStatus'
import { ConfirmDialog } from '@/components/ConfirmDialog'

const getPriorityLabelKey = (priority: string): string => {
  switch (priority) {
    case 'urgent': return 'pms.urgent'
    case 'high': return 'pms.high'
    case 'medium': return 'pms.medium'
    case 'low': return 'pms.low'
    default: return priority || 'pms.medium'
  }
}

const getPriorityColor = (priority: string) => {
  switch (priority) {
    case 'urgent': return 'red'
    case 'high': return 'orange'
    case 'medium': return 'yellow'
    case 'low': return 'green'
    default: return 'gray'
  }
}

// 将 task status slug 翻译为本地化名称（与 BacklogTab/tasks 页面的 getStatusName 逻辑一致）
const getTaskStatusName = (t: (k: string) => string, slug: string): string => {
  const camelSlug = slug.split('_').map(part => part.charAt(0).toUpperCase() + part.slice(1)).join('')
  const key = `pms.taskStatus${camelSlug}`
  const translated = t(key)
  return translated === key ? slug : translated
}

export default function UserStoryDetailPage() {
  const router = useRouter()
  const params = useParams()
  const projectSlug = params.slug as string
  const storySlug = params.storySlug as string
  const { t } = useI18n()
  const toast = useToast()
  const { user, authLoading } = useCurrentUser()
  const { canEditScrum } = useProject()
  const [story, setStory] = useState<any>(null)
  const [storyLoading, setStoryLoading] = useState(true)
  const [tasks, setTasks] = useState<any[]>([])
  const [userStories, setUserStories] = useState<any[]>([])
  // 用户故事编辑 Drawer：复用共享 UserStoryDrawer 组件
  const [isStoryDrawerOpen, setIsStoryDrawerOpen] = useState(false)

  // 新建任务抽屉
  const [isNewTaskDrawerOpen, setIsNewTaskDrawerOpen] = useState(false)
  const [taskStatuses, setTaskStatuses] = useState<any[]>([])

  // AI 拆解 Modal
  const [isAIDecomposeOpen, setIsAIDecomposeOpen] = useState(false)

  // 查看任务抽屉
  const [viewTaskId, setViewTaskId] = useState<number | null>(null)

  // 关联已存在任务抽屉
  const [isLinkTaskDrawerOpen, setIsLinkTaskDrawerOpen] = useState(false)
  const [allProjectTasks, setAllProjectTasks] = useState<any[]>([])
  const [linkSearch, setLinkSearch] = useState('')
  const [linkTaskLoading, setLinkTaskLoading] = useState(false)
  const [storyActivities, setStoryActivities] = useState<any[]>([])
  // 后端 sideload participants：批量预取操作者头像信息
  const [activityParticipants, setActivityParticipants] = useState<any[]>([])

  const [confirmOpen, setConfirmOpen] = useState(false)
  const [confirmAction, setConfirmAction] = useState<(() => void) | null>(null)

  useEffect(() => {
    if (!user || !storySlug) return
    Promise.all([
      api.getUserStory(storySlug).then(setStory).finally(() => setStoryLoading(false)),
      api.getUserStoryTasks(storySlug).then((data) => setTasks(data || [])).catch(() => {}),
      api.getUserStories(projectSlug).then((data) => setUserStories(data || [])).catch(() => {}),
      api.getTaskStatuses(projectSlug).then((data) => setTaskStatuses(data || [])).catch(() => {}),
      api.getActivities(projectSlug, 'story', storySlug).then((data) => {
        setStoryActivities(data?.activities || [])
        setActivityParticipants(data?.participants || [])
      }).catch(() => {
        setStoryActivities([])
        setActivityParticipants([])
      }),
    ])
  }, [user, storySlug, projectSlug])

  const handleEditClick = () => {
    if (story) setIsStoryDrawerOpen(true)
  }

  // 保存编辑：作为 UserStoryDrawer 的 onSave 回调，由共享组件在关闭抽屉前调用
  const handleStorySave = async (data: UserStoryFormData) => {
    if (!story) return
    try {
      const updated = await api.updateUserStory(storySlug, {
        title: data.title,
        description: data.description || null,
        status: data.status,
        priority: data.priority,
        storyPoints: data.storyPoints || 0,
        acceptanceCriteria: data.acceptanceCriteria || null,
        epicSlug: data.epicSlug || '',
      })
      setStory(updated)
      toast({
        title: t('common.success'),
        description: t('pms.userStoryUpdated'),
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
    if (!story) return
    setConfirmAction(() => async () => {
      try {
        await api.deleteUserStory(storySlug)
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

  // Inline 更新经办人：乐观更新无需 toast（参考用户偏好：高频字段编辑成功不弹通知）
  // 注意：清空经办人时传空字符串而非 null（Go json 把 null 解析为 nil pointer，无法区分"未传"和"清空"）
  const handleUpdateAssignee = async (assigneeName: string | null) => {
    if (!story) return
    try {
      const updated = await api.updateUserStory(storySlug, { assigneeName: assigneeName || '' })
      setStory(updated)
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

  // 打开关联任务抽屉，加载项目中所有 task
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
      const res: any = await api.updateTask(projectSlug, taskId, { userStorySlug: storySlug })
      const updated = res.task || res
      setTasks((prev) => [...prev, updated])
      toast({ title: t('common.success'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    }
  }

  // 取消关联任务
  async function handleUnlinkTask(taskId: number) {
    try {
      await api.updateTask(projectSlug, taskId, { userStorySlug: '' })
      setTasks((prev) => prev.filter((t) => t.taskId !== taskId))
      toast({ title: t('common.success'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    }
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

  if (storyLoading) {
    return (
      <Flex justify="center" align="center" minH="60vh">
        <Spinner size="xl" color="blue.500" />
      </Flex>
    )
  }

  if (!story) {
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
        {/* Epic 面包屑:Story 关联到 Epic 时显示,点击跳转 Epic 详情 */}
        {story?.epicSlug && (
          <HStack fontSize="sm" color="gray.600" spacing={1}>
            <Text>Epic:</Text>
            <Link href={`/projects/${projectSlug}/epics/${story.epicSlug}`} style={{ textDecoration: 'none' }}>
              <Text as="span" color="blue.500" _hover={{ textDecoration: 'underline' }}>
                {story.epicTitle}
              </Text>
            </Link>
          </HStack>
        )}
        {/* 顶部操作栏 + 用户故事信息卡片 */}
        <StoryHeader
          story={story}
          tasks={tasks}
          onEditClick={handleEditClick}
          onDelete={handleDelete}
          onBack={() => router.back()}
          canEditScrum={canEditScrum}
          onUpdateAssignee={handleUpdateAssignee}
        />

        {/* 研发流程进度条：根据 story + tasks 数据自动推断当前节点 */}
        <WorkflowProgress story={story} tasks={tasks} />

        {/* 故事进度概览：环形图 + 任务状态分布 + 工作量统计 */}
        <TaskProgressCard tasks={tasks} title={t('pms.storyProgress')} />

        {/* 关联任务列表 */}
        <Box bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" overflow="hidden">
          <Box px={6} py={3} borderBottom="1px solid #F4F6F8" bg="gray.50">
            <Flex justify="space-between" align="center">
              <Heading as="h3" size="md" color="gray.900">
                {t('pms.tasks')} ({tasks.length})
              </Heading>
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
                      variant="outline"
                      colorScheme="purple"
                      leftIcon={<FiCpu />}
                      onClick={() => setIsAIDecomposeOpen(true)}
                    >
                      {t('pms.aiDecompose')}
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
          </Box>
          <Box p={4}>
            {tasks.length > 0 ? (
              <Stack spacing={2}>
                {tasks.map((task: any, index: number) => (
                  <Box
                    key={task.taskId || `task-${index}`}
                    p={3}
                    bg="gray.50"
                    borderRadius="md"
                    _hover={{ bg: 'gray.100' }}
                  >
                    <Flex justify="space-between" align="center">
                      <Box cursor="pointer" flex={1} onClick={() => setViewTaskId(task.taskId)}>
                        <Text fontWeight="medium">{task.title}</Text>
                        <HStack spacing={4} mt={1}>
                          <Text fontSize="sm" color="gray.500">
                            {t('common.priority')}: {t(getPriorityLabelKey(task.priority))}
                          </Text>
                          <Text fontSize="sm" color="gray.500">
                            {t('common.status')}: {getTaskStatusName(t, task.status)}
                          </Text>
                          {task.storyPoints != null && task.storyPoints > 0 && (
                            <Text fontSize="sm" color="blue.600">
                              {task.storyPoints} {t('pms.storyPoints')}
                            </Text>
                          )}
                        </HStack>
                      </Box>
                      <HStack spacing={2}>
                        <Badge bg={getStatusHexColor(task.status, taskStatuses)} color="white">
                          {getTaskStatusName(t, task.status)}
                        </Badge>
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
                  </Box>
                ))}
              </Stack>
            ) : (
              <Text color="gray.500">{t('pms.noTasks')}</Text>
            )}
          </Box>
        </Box>

        {/* 活动时间线 */}
        <Box bg="white" borderRadius="12px" border="1px solid #e2e8f0" p="20px" boxShadow="0 1px 3px rgba(0,0,0,0.04)" w="100%">
          <Flex align="center" gap={2} mb={4}>
            <FiActivity color="#3182ce" />
            <Heading size="sm">{t('pms.activities')}</Heading>
          </Flex>
          <ActivityTimeline activities={storyActivities} participants={activityParticipants} />
        </Box>
      </Stack>

      {/* 用户故事编辑 Drawer：复用共享 UserStoryDrawer 组件 */}
      <UserStoryDrawer
        isOpen={isStoryDrawerOpen}
        onClose={() => setIsStoryDrawerOpen(false)}
        mode="edit"
        projectSlug={projectSlug}
        initialData={story ? {
          title: story.title,
          description: story.description || '',
          status: story.status,
          priority: story.priority,
          storyPoints: story.storyPoints || 0,
          acceptanceCriteria: story.acceptanceCriteria || '',
          epicSlug: story.epicSlug || null,
        } : undefined}
        onSave={handleStorySave}
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
        userStorySlug={storySlug}
        onTaskCreated={(task) => { setTasks((prev) => [task, ...prev]) }}
      />

      {/* 查看任务抽屉 */}
      <TaskDrawer
        isOpen={viewTaskId !== null}
        taskId={viewTaskId}
        mode="view"
        onClose={() => setViewTaskId(null)}
        projectSlug={projectSlug}
        taskStatuses={taskStatuses}
        canEditScrum={canEditScrum}
        onTaskUpdated={(updatedTask: any) => {
          setTasks((prev) => prev.map((t) => t.taskId === updatedTask.taskId ? updatedTask : t))
        }}
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
                                  {t(getPriorityLabelKey(task.priority))} · {task.status}
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
                      {tasks.map((task: any, index: number) => (
                        <Box
                          key={task.taskId || `task-${index}`}
                          p={3}
                          bg="gray.50"
                          borderRadius="md"
                        >
                          <Flex justify="space-between" align="center">
                            <Box>
                              <Text fontSize="sm" fontWeight="medium">TASK-{task.taskId} {task.title}</Text>
                              <HStack spacing={4} mt={1}>
                                <Text fontSize="xs" color="gray.500">
                                  {t(getPriorityLabelKey(task.priority))}
                                </Text>
                                <Text fontSize="xs" color="gray.500">
                                  {task.status}
                                </Text>
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

      {/* AI 拆解 Modal */}
      <AIDecomposeModal
        isOpen={isAIDecomposeOpen}
        onClose={() => setIsAIDecomposeOpen(false)}
        storySlug={storySlug}
        projectSlug={projectSlug}
        onTasksCreated={(newTasks) => {
          setTasks((prev) => [...newTasks, ...prev])
        }}
      />

      <ConfirmDialog
        isOpen={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={handleConfirm}
        title={t('common.confirm')}
        message={t('pms.deleteUserStoryConfirm')}
      />
    </>
  )
}
