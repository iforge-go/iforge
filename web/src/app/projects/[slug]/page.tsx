'use client'

import { useState, useEffect, useRef } from 'react'
import { useParams, useRouter } from 'next/navigation'
import {
  Box,
  Button,
  Card,
  CardBody,
  Flex,
  Heading,
  Stack,
  Text,
  HStack,
  Badge,
  Icon,
  Drawer,
  DrawerOverlay,
  DrawerContent,
  DrawerCloseButton,
  DrawerHeader,
  DrawerBody,
  DrawerFooter,
  Input,
  Textarea,
  Switch,
  FormControl,
  FormLabel,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
  useToast,
} from '@chakra-ui/react'
import { FiCheckSquare, FiRepeat, FiInbox, FiArrowRight, FiTarget } from 'react-icons/fi'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import ProjectHeader from './components/ProjectHeader'
import { useProject } from './ProjectContext'

export default function ProjectDetailPage() {
  const router = useRouter()
  const params = useParams()
  const projectSlug = params.slug as string
  const { t } = useI18n()
  const toast = useToast()
  const { user, authLoading } = useCurrentUser()
  const { project, loading: projectLoading, setProject } = useProject()
  const [sprints, setSprints] = useState<any[]>([])
  const [tasks, setTasks] = useState<any[]>([])
  const [userStories, setUserStories] = useState<any[]>([])
  const [epics, setEpics] = useState<any[]>([])
  const [isEditing, setIsEditing] = useState(false)
  const [editData, setEditData] = useState<Partial<any>>({})
  const [isDeleteAlertOpen, setIsDeleteAlertOpen] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const cancelRef = useRef(null as any)

  useEffect(() => {
    if (!user || !projectSlug) return
    Promise.all([
      api.getSprints(projectSlug).then((data) => setSprints(data || [])).catch(() => {}),
      api.getTasks(projectSlug).then((data) => setTasks(data || [])).catch(() => {}),
      api.getUserStories(projectSlug).then((data) => setUserStories(data || [])).catch(() => {}),
      api.getEpics(projectSlug).then((data) => setEpics(data || [])).catch(() => {}),
    ]).catch(() => {})
  }, [user, projectSlug])

  const handleEditClick = () => {
    if (project) {
      setEditData({
        name: project.name,
        description: project.description,
        isPrivate: project.isPrivate,
      })
      setIsEditing(true)
    }
  }

  const handleEditSubmit = async () => {
    if (!project) return
    try {
      const updated = await api.updateProject(projectSlug, editData)
      setProject(updated)
      setIsEditing(false)
      toast({
        title: t('common.success'),
        description: t('pms.projectUpdated'),
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
    setIsDeleteAlertOpen(true)
  }

  const confirmDelete = async () => {
    if (!project) return
    setDeleting(true)
    try {
      await api.deleteProject(projectSlug)
      router.push('/pms')
    } catch (error: any) {
      toast({
        title: t('common.error'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setDeleting(false)
      setIsDeleteAlertOpen(false)
    }
  }

  if (authLoading) return null

  if (!user) {
    return (
      <Box py={20}>
        <Stack spacing={6} textAlign="center">
          <Heading size="lg">{t('common.pleaseLogin')}</Heading>
          <Button colorScheme="blue" onClick={() => router.push('/login')}>
            {t('auth.login')}
          </Button>
        </Stack>
      </Box>
    )
  }

  if (projectLoading) {
    return (
      <Box py={20} textAlign="center">
        <Text color="gray.600">{t('common.loading')}</Text>
      </Box>
    )
  }

  if (!project) {
    return (
      <Box py={20}>
        <Stack spacing={6} textAlign="center">
          <Heading size="lg">{t('common.noData')}</Heading>
          <Button colorScheme="blue" onClick={() => router.push('/pms')}>
            {t('common.back')}
          </Button>
        </Stack>
      </Box>
    )
  }

  const taskList = tasks || []
  const sprintList = sprints || []
  const storyList = userStories || []

  // Scrum 工作流数据
  // 所有故事都在 Backlog 中（故事不再分配给 Sprint）
  const backlogStories = storyList
  const backlogPoints = backlogStories.reduce((sum: number, s: any) => sum + (s.storyPoints || 0), 0)
  const activeSprint = sprintList.find((s: any) => s.status === 'active') || sprintList.find((s: any) => s.status !== 'closed')
  // 统计口径由后端 ListSprints 返回(taskCount/totalStoryPoints),兼容 Story 级 Sprint 规划:
  // taskCount 含父 Story 在该 Sprint 的继承 task;totalStoryPoints 基于 story.sprint_id 直接关联。
  const activeSprintPoints = activeSprint?.totalStoryPoints ?? 0
  const activeSprintTaskCount = activeSprint?.taskCount ?? 0
  const totalTasks = taskList.length
  const completedTasks = taskList.filter((t: any) => t.status === 'completed' || t.status === 'done' || t.status === 'closed' || t.status === 'archived').length

  // 活跃 Epic:未完成(open/in_progress)的 Epic,按更新时间倒序,取前 5 个用于概览页展示
  const activeEpics = (epics || [])
    .filter((e: any) => e.status !== 'done' && e.status !== 'closed')
    .sort((a: any, b: any) => new Date(b.updatedAt || 0).getTime() - new Date(a.updatedAt || 0).getTime())
    .slice(0, 5)

  return (
    <Stack spacing={6}>
      {/* 项目头部 */}
      <ProjectHeader project={project} projectSlug={projectSlug} onEditClick={handleEditClick} onDelete={handleDelete} />

      {/* Scrum 工作流引导 */}
      <Card bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" overflow="hidden">
        <CardBody p={5}>
          <Flex justify="space-between" align="center" mb={4}>
            <Heading size="sm">{t('pms.scrumWorkflow')}</Heading>
          </Flex>
          <Flex direction={{ base: 'column', md: 'row' }} align="stretch" gap={2}>
            {/* 步骤 1: Epic 规划 */}
            <Box
              flex={1}
              p={4}
              bg="purple.50"
              borderRadius="md"
              cursor="pointer"
              onClick={() => router.push(`/projects/${projectSlug}/epics`)}
              _hover={{ bg: 'purple.100' }}
              transition="background 0.2s"
            >
              <HStack mb={2}>
                <Box w={6} h={6} bg="purple.500" color="white" borderRadius="full" display="flex" alignItems="center" justifyContent="center" fontSize="xs" fontWeight="bold">1</Box>
                <Icon as={FiTarget} color="purple.500" />
                <Text fontWeight="medium">{t('pms.flowStepEpic')}</Text>
              </HStack>
              <Text fontSize="xs" color="gray.500" mb={2}>{t('pms.flowStepEpicDesc')}</Text>
              {activeEpics.length > 0 ? (
                <HStack spacing={3} fontSize="sm">
                  <Text color="gray.700">{activeEpics.length} {t('pms.epics')}</Text>
                </HStack>
              ) : (
                <Text fontSize="xs" color="purple.600">{t('pms.createEpicFirst')} →</Text>
              )}
            </Box>

            {/* 箭头 1 */}
            <Flex align="center" justify="center" px={{ base: 0, md: 2 }} py={{ base: 1, md: 0 }}>
              <Icon as={FiArrowRight} color="gray.400" display={{ base: 'none', md: 'block' }} />
            </Flex>

            {/* 步骤 2: 需求池 */}
            <Box
              flex={1}
              p={4}
              bg="orange.50"
              borderRadius="md"
              cursor="pointer"
              onClick={() => router.push(`/projects/${projectSlug}/backlog`)}
              _hover={{ bg: 'orange.100' }}
              transition="background 0.2s"
            >
              <HStack mb={2}>
                <Box w={6} h={6} bg="orange.500" color="white" borderRadius="full" display="flex" alignItems="center" justifyContent="center" fontSize="xs" fontWeight="bold">2</Box>
                <Icon as={FiInbox} color="orange.500" />
                <Text fontWeight="medium">{t('pms.flowStep1')}</Text>
              </HStack>
              <Text fontSize="xs" color="gray.500" mb={2}>{t('pms.flowStep1Desc')}</Text>
              {backlogStories.length > 0 ? (
                <HStack spacing={3} fontSize="sm">
                  <Text color="gray.700">{backlogStories.length} {t('pms.stories')}</Text>
                  <Text color="purple.500" fontWeight="medium">{backlogPoints} {t('pms.storyPoints')}</Text>
                </HStack>
              ) : (
                <Text fontSize="xs" color="orange.600">{t('pms.createStoryFirst')} →</Text>
              )}
            </Box>

            {/* 箭头 2 */}
            <Flex align="center" justify="center" px={{ base: 0, md: 2 }} py={{ base: 1, md: 0 }}>
              <Icon as={FiArrowRight} color="gray.400" display={{ base: 'none', md: 'block' }} />
            </Flex>

            {/* 步骤 3: Sprint 规划 */}
            <Box
              flex={1}
              p={4}
              bg="blue.50"
              borderRadius="md"
              cursor="pointer"
              onClick={() => router.push(`/projects/${projectSlug}/sprint`)}
              _hover={{ bg: 'blue.100' }}
              transition="background 0.2s"
            >
              <HStack mb={2}>
                <Box w={6} h={6} bg="blue.500" color="white" borderRadius="full" display="flex" alignItems="center" justifyContent="center" fontSize="xs" fontWeight="bold">3</Box>
                <Icon as={FiRepeat} color="blue.500" />
                <Text fontWeight="medium">{t('pms.flowStep2')}</Text>
              </HStack>
              <Text fontSize="xs" color="gray.500" mb={2}>{t('pms.flowStep2Desc')}</Text>
              {activeSprint ? (
                <HStack spacing={3} fontSize="sm">
                  <Text color="gray.700" fontWeight="medium" noOfLines={1}>{activeSprint.title}</Text>
                  <Badge borderWidth="1px" borderStyle="solid" bg="transparent" color={activeSprint.status === 'active' ? 'green.500' : activeSprint.status === 'closed' ? 'red.500' : 'gray.500'} borderColor={activeSprint.status === 'active' ? 'green.500' : activeSprint.status === 'closed' ? 'red.500' : 'gray.500'} fontSize="2xs">
                    {activeSprint.status === 'open' ? t('pms.statusOpen') : activeSprint.status === 'active' ? t('pms.statusActive') : activeSprint.status === 'closed' ? t('pms.statusClosed') : activeSprint.status}
                  </Badge>
                  <Text color="gray.600" fontSize="xs">{activeSprintTaskCount} {t('pms.tasks')}</Text>
                  <Text color="purple.500" fontSize="xs">{activeSprintPoints} {t('pms.storyPoints')}</Text>
                </HStack>
              ) : (
                <Text fontSize="xs" color="blue.600">{t('pms.createSprintFirst')} →</Text>
              )}
            </Box>

            {/* 箭头 3 */}
            <Flex align="center" justify="center" px={{ base: 0, md: 2 }} py={{ base: 1, md: 0 }}>
              <Icon as={FiArrowRight} color="gray.400" display={{ base: 'none', md: 'block' }} />
            </Flex>

            {/* 步骤 4: 任务执行 */}
            <Box
              flex={1}
              p={4}
              bg="green.50"
              borderRadius="md"
              cursor="pointer"
              onClick={() => router.push(`/projects/${projectSlug}/tasks?view=board`)}
              _hover={{ bg: 'green.100' }}
              transition="background 0.2s"
            >
              <HStack mb={2}>
                <Box w={6} h={6} bg="green.500" color="white" borderRadius="full" display="flex" alignItems="center" justifyContent="center" fontSize="xs" fontWeight="bold">4</Box>
                <Icon as={FiCheckSquare} color="green.500" />
                <Text fontWeight="medium">{t('pms.flowStep3')}</Text>
              </HStack>
              <Text fontSize="xs" color="gray.500" mb={2}>{t('pms.flowStep3Desc')}</Text>
              {totalTasks > 0 ? (
                <HStack spacing={3} fontSize="sm">
                  <Text color="gray.700">{completedTasks}/{totalTasks} {t('pms.tasks')}</Text>
                  <Box w="60px" h="4px" bg="green.200" borderRadius="full" overflow="hidden">
                    <Box w={`${totalTasks > 0 ? Math.round((completedTasks / totalTasks) * 100) : 0}%`} h="100%" bg="green.500" borderRadius="full" />
                  </Box>
                </HStack>
              ) : (
                <Text fontSize="xs" color="green.600">{t('pms.goToExecute')} →</Text>
              )}
            </Box>
          </Flex>
        </CardBody>
      </Card>

      {/* 活跃 Epic 区块:概览页展示前 5 个未完成 Epic,点击跳转 Epic 列表页 */}
      {activeEpics.length > 0 && (
        <Card bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" overflow="hidden">
          <CardBody p={5}>
            <Flex justify="space-between" align="center" mb={4}>
              <HStack spacing={2}>
                <Box color="purple.500"><FiTarget size={16} /></Box>
                <Heading size="sm">{t('pms.activeEpics')}</Heading>
              </HStack>
              <Button
                size="sm"
                variant="ghost"
                colorScheme="purple"
                rightIcon={<FiArrowRight size={14} />}
                onClick={() => router.push(`/projects/${projectSlug}/epics`)}
              >
                {t('pms.viewAllEpics')}
              </Button>
            </Flex>
            <Stack spacing={2}>
              {activeEpics.map((epic: any) => (
                <Flex
                  key={epic.slug}
                  align="center"
                  px={3}
                  py={2}
                  borderRadius="md"
                  bg="gray.50"
                  _hover={{ bg: 'gray.100', cursor: 'pointer' }}
                  onClick={() => router.push(`/projects/${projectSlug}/epics/${epic.slug}`)}
                  transition="background 0.2s"
                >
                  <HStack flex={1} align="center" spacing={3} minW={0}>
                    <Box color="purple.500" flexShrink={0}><FiTarget size={14} /></Box>
                    <Text fontWeight="medium" color="gray.800" noOfLines={1} flex={1}>{epic.title}</Text>
                    {/* Epic 进度条(后端聚合字段) */}
                    {epic.progressPercent != null && (
                      <HStack spacing={2} flexShrink={0}>
                        <Box w="80px" h="6px" bg="gray.200" borderRadius="full" overflow="hidden">
                          <Box w={`${epic.progressPercent}%`} h="100%" bg="purple.500" borderRadius="full" />
                        </Box>
                        <Text fontSize="xs" color="gray.500" minW="32px" textAlign="right">{epic.progressPercent}%</Text>
                      </HStack>
                    )}
                  </HStack>
                  <HStack spacing={3} ml={4} flexShrink={0} fontSize="xs" color="gray.500">
                    <Text>{epic.totalStories || 0} {t('pms.stories')}</Text>
                    {epic.targetDate && (
                      <Text>📅 {String(epic.targetDate).split('T')[0]}</Text>
                    )}
                    {epic.ownerName && (
                      <Text>👤 {epic.ownerName}</Text>
                    )}
                  </HStack>
                </Flex>
              ))}
            </Stack>
          </CardBody>
        </Card>
      )}

      {/* 编辑项目 Drawer */}
      <Drawer isOpen={isEditing} placement="right" onClose={() => setIsEditing(false)} size="md" blockScrollOnMount={false}>
        <DrawerOverlay />
        <DrawerContent>
          <DrawerCloseButton />
          <DrawerHeader borderBottomWidth="1px">{t('pms.editProject')}</DrawerHeader>

          <DrawerBody>
            <Stack spacing={4}>
              <Box>
                <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.projectName')}</Text>
                <Input
                  value={editData.name || ''}
                  onChange={(e) => setEditData({ ...editData, name: e.target.value })}
                  placeholder={t('pms.projectNamePlaceholder')}
                />
              </Box>
              <Box>
                <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.projectDescription')}</Text>
                <Textarea
                  value={editData.description || ''}
                  onChange={(e) => setEditData({ ...editData, description: e.target.value })}
                  placeholder={t('pms.projectDescriptionPlaceholder')}
                  resize="none"
                  rows={4}
                />
              </Box>
              <FormControl display="flex" alignItems="center">
                <FormLabel htmlFor="isPrivate" mb={0}>
                  {t('pms.privateProject')}
                </FormLabel>
                <Switch
                  id="isPrivate"
                  isChecked={editData.isPrivate || false}
                  onChange={(e) => setEditData({ ...editData, isPrivate: e.target.checked })}
                />
              </FormControl>
            </Stack>
          </DrawerBody>

          <DrawerFooter borderTopWidth="1px">
            <Button variant="ghost" onClick={() => setIsEditing(false)} mr={3}>
              {t('common.cancel')}
            </Button>
            <Button colorScheme="blue" onClick={handleEditSubmit}>
              {t('common.save')}
            </Button>
          </DrawerFooter>
        </DrawerContent>
      </Drawer>

      {/* 删除项目确认 AlertDialog */}
      <AlertDialog
        isOpen={isDeleteAlertOpen}
        leastDestructiveRef={cancelRef}
        onClose={() => setIsDeleteAlertOpen(false)}
        blockScrollOnMount={false}
      >
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              {t('pms.deleteProjectTitle')}
            </AlertDialogHeader>
            <AlertDialogBody>
              {t('pms.deleteConfirm')}
            </AlertDialogBody>
            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={() => setIsDeleteAlertOpen(false)}>
                {t('common.cancel')}
              </Button>
              <Button colorScheme="red" onClick={confirmDelete} ml={3} isLoading={deleting}>
                {t('common.delete')}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>
    </Stack>
  )
}
