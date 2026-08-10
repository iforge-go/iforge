'use client'

import { Box, Flex, Grid, Heading, Text, Progress, Stat, StatLabel, StatNumber, StatGroup } from '@chakra-ui/react'
import { FiCheckCircle, FiCircle, FiClock, FiTrendingUp } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

interface TaskProgressCardProps {
  tasks: any[]
  /** Optional title; defaults to pms.sprintProgress */
  title?: string
  /** Show story-points stats (default true) */
  showPoints?: boolean
}

// 任务状态分组（3 桶：To Do / In Progress / Done）
// done: done / completed / closed / archived（历史 slug 均视为已完成）
// inProgress: in_progress / review（Review 归入 In Progress）
// todo: todo / open（兼容历史 slug）
function classifyTaskStatus(status: string): 'done' | 'inProgress' | 'todo' {
  const s = (status || '').toLowerCase()
  if (s === 'done' || s === 'completed' || s === 'closed' || s === 'archived') return 'done'
  if (s === 'in_progress' || s === 'review' || s === 'active' || s === 'doing') return 'inProgress'
  return 'todo'
}

// 颜色配置（与 Badge colorScheme 对齐）
const STATUS_COLORS = {
  done: '#38a169',       // green
  inProgress: '#3182ce', // blue
  todo: '#a0aec0',       // gray
}

export default function TaskProgressCard({ tasks, title, showPoints = true }: TaskProgressCardProps) {
  const { t } = useI18n()

  if (!tasks || tasks.length === 0) {
    return null
  }

  // 状态分布
  const total = tasks.length
  const counts = { done: 0, review: 0, inProgress: 0, todo: 0 }
  tasks.forEach((task) => {
    counts[classifyTaskStatus(task.status)]++
  })

  // 故事点统计
  const totalPoints = tasks.reduce((sum, t) => sum + (t.storyPoints || 0), 0)
  const donePoints = tasks
    .filter((t) => classifyTaskStatus(t.status) === 'done')
    .reduce((sum, t) => sum + (t.storyPoints || 0), 0)
  const remainingPoints = Math.max(0, totalPoints - donePoints)

  // 完成度（按任务数）
  const completionRate = total > 0 ? Math.round((counts.done / total) * 100) : 0

  // 环形图参数
  const size = 140
  const stroke = 16
  const radius = (size - stroke) / 2
  const circumference = 2 * Math.PI * radius
  const center = size / 2

  // 计算每段的弧长
  const segments = [
    { key: 'done', value: counts.done, color: STATUS_COLORS.done },
    { key: 'inProgress', value: counts.inProgress, color: STATUS_COLORS.inProgress },
    { key: 'todo', value: counts.todo, color: STATUS_COLORS.todo },
  ].filter((s) => s.value > 0)

  // 生成环形图路径
  let cumulativeOffset = 0
  const arcs = segments.map((seg) => {
    const fraction = seg.value / total
    const arcLength = fraction * circumference
    const dashArray = `${arcLength} ${circumference - arcLength}`
    const dashOffset = -cumulativeOffset
    cumulativeOffset += arcLength
    return {
      ...seg,
      dashArray,
      dashOffset,
      fraction,
    }
  })

  const statusLabel = (key: string) => {
    if (key === 'done') return t('pms.taskStatusDone')
    if (key === 'review') return t('pms.taskStatusReview')
    if (key === 'inProgress') return t('pms.taskStatusInProgress')
    return t('pms.taskStatusTodo')
  }

  const headerTitle = title || t('pms.sprintProgress')

  return (
    <Box
      bg="white"
      borderRadius="lg"
      border="1px solid #DFE2EA"
      boxShadow="1"
      p={5}
    >
      <Heading size="xs" color="gray.500" mb={4} letterSpacing="wide" textTransform="uppercase">
        {headerTitle}
      </Heading>

      <Grid templateColumns={{ base: '1fr', md: 'auto 1fr' }} gap={6} alignItems="center">
        {/* 环形图 */}
        <Flex direction="column" align="center" justify="center">
          <Box position="relative" width={size} height={size}>
            <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
              {/* 背景圆 */}
              <circle
                cx={center}
                cy={center}
                r={radius}
                fill="none"
                stroke="#EDF2F7"
                strokeWidth={stroke}
              />
              {/* 状态分段 */}
              {arcs.map((arc) => (
                <circle
                  key={arc.key}
                  cx={center}
                  cy={center}
                  r={radius}
                  fill="none"
                  stroke={arc.color}
                  strokeWidth={stroke}
                  strokeDasharray={arc.dashArray}
                  strokeDashoffset={arc.dashOffset}
                  strokeLinecap="butt"
                  transform={`rotate(-90 ${center} ${center})`}
                />
              ))}
            </svg>
            {/* 中心数字 */}
            <Box
              position="absolute"
              top="50%"
              left="50%"
              transform="translate(-50%, -50%)"
              textAlign="center"
            >
              <Text fontSize="2xl" fontWeight="bold" color="gray.900" lineHeight="1">
                {completionRate}%
              </Text>
              <Text fontSize="xs" color="gray.500" mt={1}>
                {t('pms.completionRate')}
              </Text>
            </Box>
          </Box>

          {/* 图例 */}
          <Flex gap={4} mt={3} wrap="wrap" justify="center">
            {segments.map((seg) => (
              <Flex key={seg.key} align="center" gap={1}>
                <Box width="10px" height="10px" borderRadius="sm" bg={seg.color} />
                <Text fontSize="xs" color="gray.600">
                  {statusLabel(seg.key)} ({seg.value})
                </Text>
              </Flex>
            ))}
          </Flex>
        </Flex>

        {/* 右侧统计 */}
        <Flex direction="column" gap={4}>
          <StatGroup>
            <Stat>
              <StatLabel color="gray.500" fontSize="xs">
                <Flex align="center" gap={1}>
                  <FiCircle size={11} />
                  {t('pms.taskStatusTodo')}
                </Flex>
              </StatLabel>
              <StatNumber fontSize="xl" color={STATUS_COLORS.todo}>
                {counts.todo}
              </StatNumber>
            </Stat>
            <Stat>
              <StatLabel color="gray.500" fontSize="xs">
                <Flex align="center" gap={1}>
                  <FiClock size={11} />
                  {t('pms.taskStatusInProgress')}
                </Flex>
              </StatLabel>
              <StatNumber fontSize="xl" color={STATUS_COLORS.inProgress}>
                {counts.inProgress}
              </StatNumber>
            </Stat>
            <Stat>
              <StatLabel color="gray.500" fontSize="xs">
                <Flex align="center" gap={1}>
                  <FiCheckCircle size={11} />
                  {t('pms.taskStatusDone')}
                </Flex>
              </StatLabel>
              <StatNumber fontSize="xl" color={STATUS_COLORS.done}>
                {counts.done}
              </StatNumber>
            </Stat>
          </StatGroup>

          <Box>
            <Flex justify="space-between" align="center" mb={1}>
              <Text fontSize="xs" color="gray.500">
                {t('pms.progress')} ({counts.done}/{total})
              </Text>
              <Text fontSize="xs" color="gray.500">
                {completionRate}%
              </Text>
            </Flex>
            <Progress value={completionRate} size="sm" colorScheme="green" borderRadius="full" />
          </Box>

          {showPoints && (
            <StatGroup>
              <Stat>
                <StatLabel color="gray.500" fontSize="xs">
                  <Flex align="center" gap={1}>
                    <FiTrendingUp size={11} />
                    {t('pms.totalStoryPoints')}
                  </Flex>
                </StatLabel>
                <StatNumber fontSize="xl" color="purple.500">
                  {totalPoints}
                </StatNumber>
              </Stat>
              <Stat>
                <StatLabel color="gray.500" fontSize="xs">
                  {t('pms.completedPoints')}
                </StatLabel>
                <StatNumber fontSize="xl" color="green.500">
                  {donePoints}
                </StatNumber>
              </Stat>
              <Stat>
                <StatLabel color="gray.500" fontSize="xs">
                  {t('pms.remainingPoints')}
                </StatLabel>
                <StatNumber fontSize="xl" color="orange.500">
                  {remainingPoints}
                </StatNumber>
              </Stat>
            </StatGroup>
          )}
        </Flex>
      </Grid>
    </Box>
  )
}
