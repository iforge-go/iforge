'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  Button,
  VStack,
  HStack,
  Card,
  CardBody,
  Badge,
  Icon,
  Spinner,
  Textarea,
  useToast,
  Divider,
} from '@chakra-ui/react'
import { FiArrowLeft, FiUser, FiTag, FiMessageSquare } from 'react-icons/fi'
import { useState, useEffect } from 'react'
import { api } from '@/lib/api'
import { useCurrentUser } from '@/contexts/UserContext'
import { useRouter, useParams } from 'next/navigation'
import { useI18n } from '@/contexts/I18nContext'
import { useProject } from '../../ProjectContext'
import { getStatusHexColor } from '@/lib/taskStatus'

export default function TaskDetailPage() {
  const router = useRouter()
  const params = useParams()
  const projectSlug = params.slug as string
  const taskId = Number(params.taskId)
  const { t } = useI18n()
  const toast = useToast()
  const { user, authLoading } = useCurrentUser()
  const { canEditScrum } = useProject()
  const [task, setTask] = useState<any>(null)
  const [taskLoading, setTaskLoading] = useState(true)
  const [comments, setComments] = useState<any[]>([])
  const [newComment, setNewComment] = useState('')
  const [submittingComment, setSubmittingComment] = useState(false)
  const [taskStatuses, setTaskStatuses] = useState<any[]>([])

  useEffect(() => {
    if (!user || !taskId) return
    Promise.all([
      api.getTask(projectSlug, taskId).then((res: any) => setTask(res.task || res)).finally(() => setTaskLoading(false)),
      api.getTaskComments(projectSlug, taskId).then((data) => setComments(data || [])).catch(() => {}),
      api.getTaskStatuses(projectSlug).then((data) => setTaskStatuses(data || [])).catch(() => {}),
    ])
  }, [user, taskId, projectSlug])

  const handleSubmitComment = async () => {
    if (!newComment.trim()) return

    setSubmittingComment(true)
    try {
      const comment = await api.createTaskComment(projectSlug, taskId, newComment)
      setComments([...comments, comment])
      setNewComment('')
      toast({
        title: t('pms.commentAdded'),
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
    } finally {
      setSubmittingComment(false)
    }
  }

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'urgent':
        return 'red'
      case 'high':
        return 'orange'
      case 'medium':
        return 'yellow'
      case 'low':
        return 'green'
      default:
        return 'gray'
    }
  }

  if (authLoading) return null

  if (!user) {
    return (
      <Container maxW="container.lg" py={20}>
        <VStack spacing={6} textAlign="center">
          <Heading size="lg">{t('common.pleaseLogin')}</Heading>
          <Button colorScheme="blue" onClick={() => router.push('/login')}>
            {t('auth.login')}
          </Button>
        </VStack>
      </Container>
    )
  }

  if (taskLoading) {
    return (
      <Container maxW="container.lg" py={20}>
        <VStack spacing={4}>
          <Spinner size="xl" color="blue.500" />
          <Text color="gray.600">{t('common.loading')}</Text>
        </VStack>
      </Container>
    )
  }

  if (!task) {
    return (
      <Container maxW="container.lg" py={20}>
        <VStack spacing={6} textAlign="center">
          <Heading size="lg">{t('common.noData')}</Heading>
          <Button colorScheme="blue" onClick={() => router.push(`/projects/${projectSlug}/tasks?view=board`)}>
            {t('common.back')}
          </Button>
        </VStack>
      </Container>
    )
  }

  return (
    <Container maxW="container.lg" py={8}>
      <VStack spacing={6} align="stretch">
        {/* 返回按钮 */}
        <Button
          leftIcon={<FiArrowLeft />}
          variant="ghost"
          onClick={() => router.push(`/projects/${projectSlug}/tasks?view=board`)}
        >
          {t('common.back')}
        </Button>

        {/* 任务头部 */}
        <Card>
          <CardBody>
            <VStack align="stretch" spacing={4}>
              <HStack justify="space-between" align="start">
                <VStack align="start" spacing={2} flex={1}>
                  <Heading size="lg">{task.title}</Heading>
                  <HStack spacing={2}>
                    <Badge bg={getStatusHexColor(task.status, taskStatuses)} color="white" size="lg">
                      {task.status}
                    </Badge>
                    <Badge colorScheme={getPriorityColor(task.priority)} size="lg">
                      {task.priority}
                    </Badge>
                    <Badge colorScheme="gray" size="lg">
                      {task.taskType}
                    </Badge>
                  </HStack>
                </VStack>
              </HStack>

              <HStack spacing={6} fontSize="sm" color="gray.600">
                <HStack spacing={1}>
                  <Icon as={FiUser} />
                  <Text>{t('pms.assignee')}: {task.assigneeName || '-'}</Text>
                </HStack>
                {task.storyPoints && (
                  <HStack spacing={1}>
                    <Icon as={FiTag} />
                    <Text>{task.storyPoints} {t('pms.storyPoints')}</Text>
                  </HStack>
                )}
              </HStack>
            </VStack>
          </CardBody>
        </Card>

        {/* 任务描述 */}
        {task.description && (
          <Card>
            <CardBody>
              <VStack align="stretch" spacing={3}>
                <Heading size="sm">{t('pms.taskDescription')}</Heading>
                <Text whiteSpace="pre-wrap">{task.description}</Text>
              </VStack>
            </CardBody>
          </Card>
        )}

        {/* 评论区 */}
        <Card>
          <CardBody>
            <VStack align="stretch" spacing={4}>
              <HStack>
                <Icon as={FiMessageSquare} />
                <Heading size="sm">{t('pms.comments')} ({comments.length})</Heading>
              </HStack>

              <Divider />

              {/* 评论列表 */}
              {comments.length === 0 ? (
                <Text color="gray.500" textAlign="center" py={4}>
                  {t('common.noData')}
                </Text>
              ) : (
                <VStack align="stretch" spacing={4}>
                  {comments.map((comment) => (
                    <Box key={comment.commentId} p={4} bg="gray.50" borderRadius="md">
                      <HStack justify="space-between" mb={2}>
                        <HStack spacing={2}>
                          <Box
                            w={8}
                            h={8}
                            borderRadius="full"
                            bg="blue.100"
                            display="flex"
                            alignItems="center"
                            justifyContent="center"
                          >
                            <Text fontSize="sm" fontWeight="bold" color="blue.600">
                              {comment.authorName.charAt(0).toUpperCase()}
                            </Text>
                          </Box>
                          <Text fontWeight="medium">{comment.authorName}</Text>
                        </HStack>
                        <Text fontSize="xs" color="gray.500">
                          {new Date(comment.createdAt).toLocaleString()}
                        </Text>
                      </HStack>
                      <Text whiteSpace="pre-wrap">{comment.content}</Text>
                    </Box>
                  ))}
                </VStack>
              )}

              <Divider />

              {/* 添加评论（仅 member+ 可写） */}
              {canEditScrum && (
                <VStack align="stretch" spacing={3}>
                  <Text fontWeight="medium">{t('pms.addComment')}</Text>
                  <Textarea
                    value={newComment}
                    onChange={(e) => setNewComment(e.target.value)}
                    placeholder={t('pms.commentPlaceholder')}
                    rows={4}
                  />
                  <Button
                    colorScheme="blue"
                    onClick={handleSubmitComment}
                    isLoading={submittingComment}
                    isDisabled={!newComment.trim()}
                    alignSelf="flex-end"
                  >
                    {t('pms.submitComment')}
                  </Button>
                </VStack>
              )}
            </VStack>
          </CardBody>
        </Card>
      </VStack>
    </Container>
  )
}
