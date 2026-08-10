'use client'

import Link from 'next/link'
import {
  Box,
  Button,
  Flex,
  Stack,
  Text,
  Badge,
} from '@chakra-ui/react'
import { FiPlus, FiCalendar } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

interface TasksTabProps {
  projectSlug: string
  tasks: any[]
  sprints: any[]
  onTaskClick: (taskId: number) => void
  onNewTaskClick: () => void
}

export default function TasksTab({ projectSlug, tasks, sprints, onTaskClick, onNewTaskClick }: TasksTabProps) {
  const { t } = useI18n()

  const taskList = tasks || []
  const sprintList = sprints || []

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'todo':
      case 'open':
      case 'to_do':
        return 'gray'
      case 'in_progress':
        return 'blue'
      case 'review':
        return 'purple'
      case 'done':
      case 'completed':
        return 'green'
      default:
        return 'gray'
    }
  }

  const getStatusName = (slug: string): string => {
    const camelSlug = slug.split('_').map(part => part.charAt(0).toUpperCase() + part.slice(1)).join('')
    const key = `pms.taskStatus${camelSlug}`
    const translated = t(key)
    return translated === key ? slug : translated
  }

  const getPriorityLabelKey = (priority: string): string => {
    switch (priority) {
      case 'urgent': return 'pms.urgent'
      case 'high': return 'pms.high'
      case 'medium': return 'pms.medium'
      case 'low': return 'pms.low'
      default: return priority || 'pms.medium'
    }
  }

  return (
    <>
      <Flex justify="space-between" align="center" mb={4}>
        <Text fontSize="sm" color="gray.500">{t('pms.tasks')} ({taskList.length})</Text>
        <Flex gap={3} align="center">
          <Link href={`/projects/${projectSlug}/tasks`} style={{ fontSize: 'sm', color: '#3182ce' }}>
            {t('common.viewAll')}
          </Link>
          <Button size="sm" colorScheme="blue" leftIcon={<FiPlus />} onClick={onNewTaskClick}>
            {t('pms.newTask')}
          </Button>
        </Flex>
      </Flex>
      {taskList && taskList.length > 0 ? (
        <Stack spacing={2}>
          {taskList.map((task: any) => {
            const assignedSprint = sprintList.find((s: any) => s.slug === task.sprintSlug)
            return (
              <Box
                key={task.taskId}
                p={3}
                bg="gray.50"
                borderRadius="md"
                cursor="pointer"
                _hover={{ bg: 'gray.100' }}
                onClick={() => onTaskClick(task.taskId)}
              >
                <Flex justify="space-between" align="center">
                  <Box textAlign="left">
                    <Text fontWeight="medium">{task.title}</Text>
                    <Flex gap={4} mt={1} flexWrap="wrap">
                      <Text fontSize="sm" color="gray.500">{t('common.priority')}: {t(getPriorityLabelKey(task.priority))}</Text>
                      <Text fontSize="sm" color="gray.500">{t('common.status')}: {getStatusName(task.status)}</Text>
                      {assignedSprint && (
                        <Text fontSize="sm" color="blue.600">
                          <FiCalendar size={12} style={{ marginRight: 4 }} /> {assignedSprint.title}
                        </Text>
                      )}
                    </Flex>
                  </Box>
                  <Badge variant="outline" colorScheme={getStatusColor(task.status)} flexShrink={0}>
                    {getStatusName(task.status)}
                  </Badge>
                </Flex>
              </Box>
            )
          })}
        </Stack>
      ) : (
        <Text color="gray.500">{t('pms.noTasks')}</Text>
      )}
    </>
  )
}
