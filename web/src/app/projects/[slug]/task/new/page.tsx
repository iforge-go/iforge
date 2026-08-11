'use client'

import {
  Container,
  Heading,
  Text,
  Button,
  VStack,
  HStack,
  FormControl,
  FormLabel,
  Input,
  Textarea,
  Select,
  NumberInput,
  NumberInputField,
  NumberInputStepper,
  NumberIncrementStepper,
  NumberDecrementStepper,
  useToast,
  Card,
  CardBody,
} from '@chakra-ui/react'
import { Suspense, useState, useEffect } from 'react'
import { useRouter, useParams, useSearchParams } from 'next/navigation'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'

function NewTaskContent() {
  const router = useRouter()
  const params = useParams()
  const projectSlug = params.slug as string
  const searchParams = useSearchParams()
  const { t } = useI18n()
  const toast = useToast()
  const { user, authLoading } = useCurrentUser()
  const [sprints, setSprints] = useState<any[]>([])
  const [userStories, setUserStories] = useState<any[]>([])

  const initialUserStorySlug = searchParams.get('userStorySlug') || ''
  const [formData, setFormData] = useState({
    title: '',
    description: '',
    status: 'todo',
    priority: 'medium',
    taskType: 'task',
    storyPoints: 0,
    assigneeName: '',
    sprintSlug: '',
    userStorySlug: initialUserStorySlug,
  })
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    if (!user || !projectSlug) return
    Promise.all([
      api.getSprints(projectSlug).then((data) => setSprints(data || [])).catch(() => {}),
      api.getUserStories(projectSlug).then((data) => setUserStories(data || [])).catch(() => {}),
    ])
  }, [user, projectSlug])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!formData.title) {
      toast({
        title: t('common.error'),
        description: t('pms.titleAndSlugRequired'),
        status: 'error',
        duration: 3000,
      })
      return
    }

    setIsSubmitting(true)
    try {
      const task = await api.createTask(projectSlug, {
        title: formData.title,
        description: formData.description || undefined,
        status: formData.status,
        priority: formData.priority,
        taskType: formData.taskType,
        storyPoints: formData.storyPoints || undefined,
        assigneeName: formData.assigneeName || undefined,
        sprintSlug: formData.sprintSlug || undefined,
        userStorySlug: formData.userStorySlug || undefined,
      })
      
      toast({
        title: t('pms.taskCreated'),
        description: t('pms.taskCreatedDesc'),
        status: 'success',
        duration: 3000,
      })
      
      router.push(`/projects/${projectSlug}/task/${task.taskId}`)
    } catch (error: any) {
      toast({
        title: t('common.error'),
        description: error.message || t('pms.createFailed'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  if (authLoading) return null

  if (!user) {
    return (
      <Container maxW="container.md" py={20}>
        <VStack spacing={6} textAlign="center">
          <Heading size="lg">{t('common.pleaseLogin')}</Heading>
          <Button colorScheme="blue" onClick={() => router.push('/login')}>
            {t('auth.login')}
          </Button>
        </VStack>
      </Container>
    )
  }

  return (
    <Container maxW="container.md" py={8}>
      <VStack spacing={8} align="stretch">
        <VStack align="start" spacing={2}>
          <Heading size="lg">{t('pms.newTask')}</Heading>
          <Text color="gray.600">{t('pms.newTaskDesc')}</Text>
        </VStack>

        <Card>
          <CardBody>
            <form onSubmit={handleSubmit}>
              <VStack spacing={6}>
                <FormControl isRequired>
                  <FormLabel>{t('pms.taskTitle')}</FormLabel>
                  <Input
                    value={formData.title}
                    onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                    placeholder={t('pms.taskTitlePlaceholder')}
                  />
                </FormControl>

                <FormControl>
                  <FormLabel>{t('pms.taskDescription')}</FormLabel>
                  <Textarea
                    value={formData.description}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    placeholder={t('pms.taskDescriptionPlaceholder')}
                    rows={4}
                  />
                </FormControl>

                <HStack spacing={4} w="100%">
                  <FormControl>
                    <FormLabel>{t('pms.taskStatus')}</FormLabel>
                    <Select
                      value={formData.status}
                      onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                    >
                      <option value="todo">{t('pms.taskStatusTodo')}</option>
                      <option value="in_progress">{t('pms.taskStatusInProgress')}</option>
                      <option value="review">{t('pms.taskStatusReview')}</option>
                      <option value="done">{t('pms.taskStatusDone')}</option>
                    </Select>
                  </FormControl>

                  <FormControl>
                    <FormLabel>{t('pms.taskPriority')}</FormLabel>
                    <Select
                      value={formData.priority}
                      onChange={(e) => setFormData({ ...formData, priority: e.target.value })}
                    >
                      <option value="urgent">{t('pms.priorityUrgent')}</option>
                      <option value="high">{t('pms.priorityHigh')}</option>
                      <option value="medium">{t('pms.priorityMedium')}</option>
                      <option value="low">{t('pms.priorityLow')}</option>
                    </Select>
                  </FormControl>
                </HStack>

                <HStack spacing={4} w="100%">
                  <FormControl>
                    <FormLabel>{t('pms.taskType')}</FormLabel>
                    <Select
                      value={formData.taskType}
                      onChange={(e) => setFormData({ ...formData, taskType: e.target.value })}
                    >
                      <option value="task">{t('pms.typeTask')}</option>
                      <option value="bug">{t('pms.typeBug')}</option>
                      <option value="feature">{t('pms.typeFeature')}</option>
                      <option value="improvement">{t('pms.typeImprovement')}</option>
                    </Select>
                  </FormControl>

                  <FormControl>
                    <FormLabel>{t('pms.storyPoints')}</FormLabel>
                    <NumberInput
                      value={formData.storyPoints}
                      onChange={(valueString) => setFormData({ ...formData, storyPoints: parseInt(valueString) || 0 })}
                      min={0}
                    >
                      <NumberInputField />
                      <NumberInputStepper>
                        <NumberIncrementStepper />
                        <NumberDecrementStepper />
                      </NumberInputStepper>
                    </NumberInput>
                  </FormControl>
                </HStack>

                <FormControl>
                  <FormLabel>{t('pms.assignee')}</FormLabel>
                  <Input
                    value={formData.assigneeName}
                    onChange={(e) => setFormData({ ...formData, assigneeName: e.target.value })}
                    placeholder={t('pms.assigneePlaceholder')}
                  />
                </FormControl>

                <HStack spacing={4} w="100%">
                  <FormControl>
                    <FormLabel>{t('pms.sprint')}</FormLabel>
                    <Select
                      value={formData.sprintSlug}
                      onChange={(e) => setFormData({ ...formData, sprintSlug: e.target.value })}
                      placeholder={t('pms.selectSprint')}
                    >
                      {sprints.map((sprint) => (
                        <option key={sprint.slug} value={sprint.slug}>
                          {sprint.title}
                        </option>
                      ))}
                    </Select>
                  </FormControl>

                  <FormControl>
                    <FormLabel>{t('pms.userStory')}</FormLabel>
                    <Select
                      value={formData.userStorySlug}
                      onChange={(e) => setFormData({ ...formData, userStorySlug: e.target.value })}
                      placeholder={t('pms.selectUserStory')}
                    >
                      {userStories.map((story) => (
                        <option key={story.slug} value={story.slug}>
                          {story.title}
                        </option>
                      ))}
                    </Select>
                  </FormControl>
                </HStack>

                <HStack spacing={4} w="100%">
                  <Button
                    variant="outline"
                    onClick={() => router.push(`/projects/${projectSlug}`)}
                    flex={1}
                  >
                    {t('common.cancel')}
                  </Button>
                  <Button
                    type="submit"
                    colorScheme="blue"
                    isLoading={isSubmitting}
                    flex={1}
                  >
                    {t('pms.createTask')}
                  </Button>
                </HStack>
              </VStack>
            </form>
          </CardBody>
        </Card>
      </VStack>
    </Container>
  )
}

export default function NewTaskPage() {
  return (
    <Suspense fallback={null}>
      <NewTaskContent />
    </Suspense>
  )
}
