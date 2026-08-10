'use client'

import { useState, useRef } from 'react'
import {
  Box,
  Button,
  Card,
  CardBody,
  Flex,
  Grid,
  Stack,
  Text,
  Divider,
  Badge,
  HStack,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
} from '@chakra-ui/react'
import { FiEdit, FiTrash2, FiColumns, FiCheckCircle, FiPlay, FiTarget } from 'react-icons/fi'
import NextLink from 'next/link'
import { useI18n } from '@/contexts/I18nContext'
import SprintBreadcrumb from '@/components/SprintBreadcrumb'

interface SprintHeaderProps {
  sprint: any
  tasks: any[]
  projectSlug: string
  taskStatuses: any[]
  onEditClick: () => void
  onDelete: () => void
  onStart: () => Promise<void>
  onComplete: () => Promise<void>
  canEditScrum: boolean
}

export default function SprintHeader({
  sprint,
  tasks,
  projectSlug,
  taskStatuses,
  onEditClick,
  onDelete,
  onStart,
  onComplete,
  canEditScrum,
}: SprintHeaderProps) {
  const { t } = useI18n()
  const [isCompleteDialogOpen, setIsCompleteDialogOpen] = useState(false)
  const [completing, setCompleting] = useState(false)
  const [starting, setStarting] = useState(false)
  const cancelRef = useRef(null as any)

  // 任务完成统计：用 taskStatuses 的 isClosed 字段判断，兜底常见关闭状态值
  const completedCount = tasks.filter((task) => {
    const statusConfig = taskStatuses.find((s) => s.slug === task.status)
    return statusConfig?.isClosed || ['done', 'closed', 'completed', 'archived'].includes(task.status)
  }).length
  const incompleteCount = tasks.length - completedCount

  // 状态 Badge：待开始(gray) / 进行中(green) / 已关闭(red)
  const getStatusBadge = () => {
    const status = sprint.status
    const colorScheme = status === 'active' ? 'green' : status === 'closed' ? 'red' : 'gray'
    const labelKey = status === 'active' ? 'pms.statusActive' : status === 'closed' ? 'pms.statusClosed' : 'pms.statusOpen'
    return <Badge colorScheme={colorScheme} fontSize="xs">{t(labelKey)}</Badge>
  }

  // 开始迭代：直接执行，无需确认对话框（开始操作无破坏性，可随时通过编辑改回）
  const handleStart = async () => {
    setStarting(true)
    try {
      await onStart()
    } finally {
      setStarting(false)
    }
  }

  const handleComplete = async () => {
    setCompleting(true)
    try {
      await onComplete()
      setIsCompleteDialogOpen(false)
    } finally {
      setCompleting(false)
    }
  }

  return (
    <>
      {/* 顶部：面包屑 + 状态 + 操作按钮 */}
      <Flex justify="space-between" align="center">
        <HStack spacing={3}>
          <SprintBreadcrumb sprint={sprint} />
          {getStatusBadge()}
        </HStack>
        <Flex gap={2}>
          {/* 待开始的迭代：PO 可一键开始 */}
          {canEditScrum && sprint.status === 'open' && (
            <Button colorScheme="blue" variant="solid" isLoading={starting} onClick={handleStart} size="sm">
              <FiPlay style={{ marginRight: 8 }} />
              {t('pms.startSprint')}
            </Button>
          )}
          {/* 进行中的迭代：PO 可一键完成 */}
          {canEditScrum && sprint.status === 'active' && (
            <Button colorScheme="green" variant="solid" onClick={() => setIsCompleteDialogOpen(true)} size="sm">
              <FiCheckCircle style={{ marginRight: 8 }} />
              {t('pms.completeSprint')}
            </Button>
          )}
          {canEditScrum && (
            <>
              <Button colorScheme="blue" onClick={onEditClick} size="sm">
                <FiEdit style={{ marginRight: 8 }} />
                {t('common.edit')}
              </Button>
              <Button colorScheme="red" onClick={onDelete} size="sm">
                <FiTrash2 style={{ marginRight: 8 }} />
                {t('common.delete')}
              </Button>
            </>
          )}
          {/* Sprint 看板：导航按钮，放最右边与操作按钮分隔 */}
          <Button
            as={NextLink}
            href={`/projects/${projectSlug}/tasks?view=board&sprint=${sprint.slug}`}
            colorScheme="green"
            size="sm"
            ml="auto"
          >
            <FiColumns style={{ marginRight: 8 }} />
            {t('pms.sprintPlanning')}
          </Button>
        </Flex>
      </Flex>

      {/* 迭代信息卡片（目标/描述/统计） */}
      <Card>
        <CardBody>
          <Stack spacing={4}>
            {sprint.goal && (
              <Box>
                <Text fontSize="sm" fontWeight="medium" color="gray.500" mb={2}>{t('pms.sprintGoal')}</Text>
                <Text color="gray.700">{sprint.goal}</Text>
              </Box>
            )}

            {sprint.description && (
              <Box>
                <Text fontSize="sm" fontWeight="medium" color="gray.500" mb={2}>{t('pms.sprintDescription')}</Text>
                <Text color="gray.700">{sprint.description}</Text>
              </Box>
            )}

            <Divider />

            <Grid templateColumns={{ base: '1fr', md: 'repeat(2, 1fr)' }} gap={4}>
              <Box>
                <Text fontSize="sm" color="gray.500">{t('pms.tasks')}</Text>
                <Text fontSize="xl" fontWeight="bold" color="gray.900">{tasks.length}</Text>
              </Box>
              <Box>
                <Text fontSize="sm" color="gray.500">{t('pms.storyPoints')}</Text>
                <Text fontSize="xl" fontWeight="bold" color="gray.900">
                  {tasks.reduce((sum, t) => sum + (t.storyPoints || 0), 0)}
                </Text>
              </Box>
            </Grid>
          </Stack>
        </CardBody>
      </Card>

      {/* 完成迭代确认对话框：显示任务统计，未完成任务提示保留在迭代中 */}
      {/* blockScrollOnMount={false}：避免锁定 body 滚动时引入 padding-right 切换，关闭时导致页面抖动 */}
      <AlertDialog
        isOpen={isCompleteDialogOpen}
        leastDestructiveRef={cancelRef}
        onClose={() => setIsCompleteDialogOpen(false)}
        blockScrollOnMount={false}
      >
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              {t('pms.completeSprint')}
            </AlertDialogHeader>
            <AlertDialogBody>
              <Text>{t('pms.completeSprintConfirm')}</Text>
              <Box mt={4} p={3} bg="gray.50" borderRadius="md">
                <HStack spacing={6} fontSize="sm">
                  <Box>
                    <Text color="gray.500">{t('pms.totalTasks')}</Text>
                    <Text fontWeight="bold">{tasks.length}</Text>
                  </Box>
                  <Box>
                    <Text color="gray.500">{t('pms.completedTasks')}</Text>
                    <Text fontWeight="bold" color="green.600">{completedCount}</Text>
                  </Box>
                  <Box>
                    <Text color="gray.500">{t('pms.incompleteTasks')}</Text>
                    <Text fontWeight="bold" color={incompleteCount > 0 ? 'orange.600' : 'gray.900'}>{incompleteCount}</Text>
                  </Box>
                </HStack>
              </Box>
              {/* Sprint 目标回顾：关闭迭代前对照承诺 */}
              {sprint.goal && (
                <Box mt={3} p={3} bg="blue.50" borderRadius="md">
                  <HStack spacing={2} mb={1}>
                    <FiTarget size={13} color="#3182ce" />
                    <Text fontSize="sm" fontWeight="600" color="gray.700">{t('pms.sprintGoal')}</Text>
                  </HStack>
                  <Text fontSize="sm" color="gray.700" whiteSpace="pre-wrap">{sprint.goal}</Text>
                </Box>
              )}
              {incompleteCount > 0 && (
                <Text mt={3} fontSize="sm" color="gray.600">{t('pms.completeSprintWarning')}</Text>
              )}
            </AlertDialogBody>
            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={() => setIsCompleteDialogOpen(false)}>
                {t('common.cancel')}
              </Button>
              <Button colorScheme="green" isLoading={completing} onClick={handleComplete} ml={3}>
                {t('pms.completeSprint')}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>
    </>
  )
}
