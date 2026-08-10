'use client'

import { Box, HStack, VStack, Text, Icon, Circle } from '@chakra-ui/react'
import { Fragment, useMemo } from 'react'
import {
  FiFileText,
  FiList,
  FiCalendar,
  FiUser,
  FiCode,
  FiEye,
  FiCheckCircle,
} from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

interface WorkflowProgressProps {
  story: any
  tasks: any[]
}

type NodeStatus = 'done' | 'current' | 'pending'

interface WorkflowNode {
  key: string
  labelKey: string
  defaultLabel: string
  icon: any
  done: boolean
  status: NodeStatus
}

export function WorkflowProgress({ story, tasks }: WorkflowProgressProps) {
  const { t } = useI18n()

  const nodes: WorkflowNode[] = useMemo(() => {
    const hasTasks = tasks.length > 0
    const hasSprint = tasks.some((t) => t.sprintSlug || t.sprintId)
    const hasAssignee = tasks.some(
      (t) => t.assigneeName || (Array.isArray(t.assignees) && t.assignees.length > 0)
    )
    const hasDevelopment = tasks.some(
      (t) => t.status === 'in_progress' || t.status === 'done' || t.status === 'closed'
    )
    const hasReview = tasks.some(
      (t) => t.status === 'review' || t.status === 'done' || t.status === 'closed'
    )
    const allDone =
      tasks.length > 0 &&
      tasks.every((t) => t.status === 'done' || t.status === 'closed')
    const storyCompleted = story?.status === 'done' || story?.status === 'closed'

    const configs = [
      { key: 'requirement', labelKey: 'workflow.requirement', defaultLabel: '提需求', icon: FiFileText, done: true },
      { key: 'decompose', labelKey: 'workflow.decompose', defaultLabel: '拆解任务', icon: FiList, done: hasTasks },
      { key: 'sprint', labelKey: 'workflow.sprint', defaultLabel: '规划Sprint', icon: FiCalendar, done: hasSprint },
      { key: 'assign', labelKey: 'workflow.assign', defaultLabel: '指派经办人', icon: FiUser, done: hasAssignee },
      { key: 'develop', labelKey: 'workflow.develop', defaultLabel: '开始开发', icon: FiCode, done: hasDevelopment },
      { key: 'review', labelKey: 'workflow.review', defaultLabel: '评审', icon: FiEye, done: hasReview },
      { key: 'complete', labelKey: 'workflow.complete', defaultLabel: '完成', icon: FiCheckCircle, done: storyCompleted || allDone },
    ]

    const currentIndex = configs.findIndex((n) => !n.done)
    return configs.map((n, i) => ({
      ...n,
      status: n.done ? ('done' as NodeStatus) : i === currentIndex ? ('current' as NodeStatus) : ('pending' as NodeStatus),
    }))
  }, [story, tasks])

  const colors: Record<NodeStatus, { bg: string; text: string; line: string }> = {
    done: { bg: 'green.500', text: 'green.600', line: 'green.400' },
    current: { bg: 'blue.500', text: 'blue.600', line: 'gray.200' },
    pending: { bg: 'gray.300', text: 'gray.500', line: 'gray.200' },
  }

  const label = (n: WorkflowNode) => {
    const translated = t(n.labelKey)
    return translated === n.labelKey ? n.defaultLabel : translated
  }

  return (
    <Box bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" p={4}>
      <HStack spacing={0} align="flex-start" justify="space-between">
        {nodes.map((node, i) => (
          <Fragment key={node.key}>
            <VStack spacing={1} flexShrink={0} minW="72px">
              <Circle
                size="32px"
                bg={colors[node.status].bg}
                color="white"
                position="relative"
                zIndex={1}
                boxShadow={node.status === 'current' ? '0 0 0 3px rgba(66,153,225,0.25)' : 'none'}
              >
                <Icon as={node.icon} w={4} h={4} />
              </Circle>
              <Text
                fontSize="xs"
                fontWeight={node.status === 'current' ? 'bold' : 'medium'}
                color={colors[node.status].text}
                whiteSpace="nowrap"
              >
                {label(node)}
              </Text>
            </VStack>
            {i < nodes.length - 1 && (
              <Box
                flex={1}
                h="2px"
                bg={node.status === 'done' ? colors.done.line : colors.pending.line}
                mt="15px"
                borderRadius="1px"
                transition="background 0.2s"
              />
            )}
          </Fragment>
        ))}
      </HStack>
    </Box>
  )
}
