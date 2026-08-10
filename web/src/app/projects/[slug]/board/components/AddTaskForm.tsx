'use client'

import { useState } from 'react'
import {
  Box,
  Button,
  HStack,
  Input,
  Select,
  VStack,
  useToast,
} from '@chakra-ui/react'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'

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
}

interface AddTaskFormProps {
  status: string
  projectSlug: string
  sprintSlug?: string
  onTaskCreated: (task: TaskItem) => void
  onCancel: () => void
}

// --- Add Task Form ---
export default function AddTaskForm({
  status,
  projectSlug,
  sprintSlug,
  onTaskCreated,
  onCancel,
}: AddTaskFormProps) {
  const { t } = useI18n()
  const toast = useToast()
  const [title, setTitle] = useState('')
  const [priority, setPriority] = useState('medium')
  const [taskType, setTaskType] = useState('task')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleSubmit = async () => {
    if (!title.trim()) {
      toast({
        title: t('pms.inputTitle'),
        status: 'error',
        duration: 2000,
      })
      return
    }

    setIsSubmitting(true)
    try {
      const task = await api.createTask(projectSlug, {
        title: title.trim(),
        status,
        priority,
        taskType,
        sprintSlug,
      })
      onTaskCreated(task)
      setTitle('')
      toast({
        title: t('pms.taskCreatedSuccess'),
        status: 'success',
        duration: 2000,
      })
    } catch (error: any) {
      toast({
        title: t('pms.createFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Box bg="white" borderRadius="8px" borderWidth="1px" borderColor="#e2e8f0" p="10px">
      <VStack spacing="8px" align="stretch">
        <Input
          placeholder={t('pms.taskTitle')}
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          size="sm"
          autoFocus
        />
        <HStack spacing="8px">
          <Select
            size="sm"
            value={priority}
            onChange={(e) => setPriority(e.target.value)}
            flex="1"
          >
            <option value="urgent">{t('pms.urgent')}</option>
            <option value="high">{t('pms.high')}</option>
            <option value="medium">{t('pms.medium')}</option>
            <option value="low">{t('pms.low')}</option>
          </Select>
          <Select
            size="sm"
            value={taskType}
            onChange={(e) => setTaskType(e.target.value)}
            flex="1"
          >
            <option value="task">{t('pms.typeTask')}</option>
            <option value="bug">{t('pms.typeBug')}</option>
            <option value="feature">{t('pms.typeFeature')}</option>
            <option value="improvement">{t('pms.typeImprovement')}</option>
          </Select>
        </HStack>
        <HStack spacing="8px" justify="flex-end">
          <Button size="sm" variant="ghost" onClick={onCancel} isDisabled={isSubmitting}>
            {t('common.cancel')}
          </Button>
          <Button
            size="sm"
            colorScheme="blue"
            onClick={handleSubmit}
            isLoading={isSubmitting}
          >
            {t('common.add')}
          </Button>
        </HStack>
      </VStack>
    </Box>
  )
}
