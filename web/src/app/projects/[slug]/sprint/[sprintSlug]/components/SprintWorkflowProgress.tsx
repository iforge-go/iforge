'use client'

import { Box, HStack, VStack, Text, Icon, Circle, Heading } from '@chakra-ui/react'
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

interface SprintWorkflowProgressProps {
  stories: any[]
  tasks: any[]
}

type StageStatus = 'done' | 'partial' | 'pending'

interface StageAgg {
  key: string
  labelKey: string
  defaultLabel: string
  icon: any
  done: number // 已通过此阶段的 story 数
  total: number // story 总数
  status: StageStatus
}

/**
 * Sprint 研发流程聚合进度条：遍历 Sprint 内所有 story，统计每个流程阶段有多少 story 已通过。
 *
 * 与 story 详情页的 WorkflowProgress 区别：
 *   - WorkflowProgress：单 story 视角，节点状态为 done/current/pending
 *   - SprintWorkflowProgress：聚合视角，节点状态为 done(全部通过)/partial(部分通过)/pending(无人通过)
 *
 * 阶段判定逻辑（与 WorkflowProgress 保持一致）：
 *   - 提需求：story 存在即视为通过
 *   - 拆解任务：story 至少有一个 task
 *   - 规划Sprint：story 的 task 至少有一个关联 sprint
 *   - 指派经办人：story 的 task 至少有一个 assignee
 *   - 开始开发：story 的 task 至少有一个 in_progress/review/done/closed
 *   - 评审：story 的 task 至少有一个 review/done/closed
 *   - 完成：story 状态为 done/closed，或所有 task 都 done/closed
 *
 * 注意：tasks 入参是 Sprint 内的 tasks（已 sideload assignees）。
 * story 的 tasks 通过 userStorySlug 过滤得到，反映该 story 在本 Sprint 的进展。
 */
export function SprintWorkflowProgress({ stories, tasks }: SprintWorkflowProgressProps) {
  const { t } = useI18n()

  const stages: StageAgg[] = useMemo(() => {
    const total = stories.length
    if (total === 0) return []

    let countRequirement = 0
    let countDecompose = 0
    let countSprint = 0
    let countAssign = 0
    let countDevelop = 0
    let countReview = 0
    let countComplete = 0

    for (const story of stories) {
      const storyTasks = tasks.filter((tk) => tk.userStorySlug === story.slug)

      const hasTasks = storyTasks.length > 0
      const hasSprint = storyTasks.some((tk) => tk.sprintSlug || tk.sprintId)
      const hasAssignee = storyTasks.some(
        (tk) => tk.assigneeName || (Array.isArray(tk.assignees) && tk.assignees.length > 0)
      )
      const hasDevelopment = storyTasks.some(
        (tk) => tk.status === 'in_progress' || tk.status === 'done' || tk.status === 'closed'
      )
      const hasReview = storyTasks.some(
        (tk) => tk.status === 'review' || tk.status === 'done' || tk.status === 'closed'
      )
      const allDone =
        storyTasks.length > 0 &&
        storyTasks.every((tk) => tk.status === 'done' || tk.status === 'closed')
      const storyCompleted = story.status === 'done' || story.status === 'closed'

      // 提需求：story 存在即通过
      countRequirement++
      if (hasTasks) countDecompose++
      if (hasSprint) countSprint++
      if (hasAssignee) countAssign++
      if (hasDevelopment) countDevelop++
      if (hasReview) countReview++
      if (storyCompleted || allDone) countComplete++
    }

    const configs = [
      { key: 'requirement', labelKey: 'workflow.requirement', defaultLabel: '提需求', icon: FiFileText, done: countRequirement },
      { key: 'decompose', labelKey: 'workflow.decompose', defaultLabel: '拆解任务', icon: FiList, done: countDecompose },
      { key: 'sprint', labelKey: 'workflow.sprint', defaultLabel: '规划Sprint', icon: FiCalendar, done: countSprint },
      { key: 'assign', labelKey: 'workflow.assign', defaultLabel: '指派经办人', icon: FiUser, done: countAssign },
      { key: 'develop', labelKey: 'workflow.develop', defaultLabel: '开始开发', icon: FiCode, done: countDevelop },
      { key: 'review', labelKey: 'workflow.review', defaultLabel: '评审', icon: FiEye, done: countReview },
      { key: 'complete', labelKey: 'workflow.complete', defaultLabel: '完成', icon: FiCheckCircle, done: countComplete },
    ]

    return configs.map((c) => {
      const status: StageStatus =
        c.done === 0 ? 'pending' : c.done === total ? 'done' : 'partial'
      return { ...c, total, status }
    })
  }, [stories, tasks])

  const colors: Record<StageStatus, { bg: string; text: string; line: string }> = {
    done: { bg: 'green.500', text: 'green.600', line: 'green.400' },
    partial: { bg: 'blue.500', text: 'blue.600', line: 'gray.200' },
    pending: { bg: 'gray.300', text: 'gray.500', line: 'gray.200' },
  }

  const label = (n: StageAgg) => {
    const translated = t(n.labelKey)
    return translated === n.labelKey ? n.defaultLabel : translated
  }

  if (stages.length === 0) return null

  return (
    <Box
      bg="white"
      borderRadius="lg"
      border="1px solid #DFE2EA"
      boxShadow="1"
      p={5}
    >
      <Heading size="xs" color="gray.500" mb={4} letterSpacing="wide" textTransform="uppercase">
        {t('workflow.title')}
      </Heading>
      <HStack spacing={0} align="flex-start" justify="space-between">
        {stages.map((stage, i) => (
          <Fragment key={stage.key}>
            <VStack spacing={1} flexShrink={0} minW="72px">
              <Circle
                size="32px"
                bg={colors[stage.status].bg}
                color="white"
                position="relative"
                zIndex={1}
                boxShadow={stage.status === 'partial' ? '0 0 0 3px rgba(66,153,225,0.25)' : 'none'}
              >
                <Icon as={stage.icon} w={4} h={4} />
              </Circle>
              <Text
                fontSize="xs"
                fontWeight={stage.status === 'partial' ? 'bold' : 'medium'}
                color={colors[stage.status].text}
                whiteSpace="nowrap"
              >
                {label(stage)}
              </Text>
              <Text
                fontSize="xs"
                color={colors[stage.status].text}
                fontWeight="medium"
              >
                {stage.done}/{stage.total}
              </Text>
            </VStack>
            {i < stages.length - 1 && (
              <Box
                flex={1}
                h="2px"
                bg={stage.status === 'done' ? colors.done.line : colors.pending.line}
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
