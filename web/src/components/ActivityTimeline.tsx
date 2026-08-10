'use client'

import { Box, Text, HStack, VStack, Avatar, Badge, Flex } from '@chakra-ui/react'
import { FiPlus, FiEdit, FiTrash2, FiArrowRight, FiCheck, FiGitCommit, FiGitPullRequest, FiXCircle } from 'react-icons/fi'
import { useMemo } from 'react'
import { useI18n } from '@/contexts/I18nContext'
import { getStatusColor } from './task-drawer/taskMeta'
import { formatRelativeTime } from '@/lib/time'
import type { ScrumActivity, Participant } from '@/lib/types'

interface ActivityTimelineProps {
  activities: ScrumActivity[]
  /** Optional: task statuses with color config (hex) for status change coloring */
  taskStatuses?: { slug: string; color?: string }[]
  /** Optional: sideloaded participants for rendering real avatars (userName → image/fullName) */
  participants?: Participant[]
}

const actionIcons: Record<string, any> = {
  created: FiPlus,
  updated: FiEdit,
  status_changed: FiArrowRight,
  deleted: FiTrash2,
  commit_pushed: FiGitCommit,
  branch_created: FiGitCommit,
  mr_created: FiGitPullRequest,
  mr_merged: FiCheck,
  mr_closed: FiXCircle,
}

const actionColors: Record<string, string> = {
  created: 'green',
  updated: 'blue',
  status_changed: 'orange',
  deleted: 'red',
  commit_pushed: 'purple',
  branch_created: 'cyan',
  mr_created: 'purple',
  mr_merged: 'green',
  mr_closed: 'red',
}

export default function ActivityTimeline({ activities, taskStatuses, participants }: ActivityTimelineProps) {
  const { t } = useI18n()

  // 按状态 slug 查找后端配置的 hex 颜色，fallback 到色系名
  const getStatusHexColor = (slug: string): string | undefined => {
    const status = taskStatuses?.find(s => s.slug === slug)
    return status?.color
  }

  // Sideload participants 查找表：userName → 头像信息，避免每条活动都 .find
  const participantMap = useMemo(() => {
    const m = new Map<string, Participant>()
    if (participants) {
      for (const p of participants) m.set(p.userName, p)
    }
    return m
  }, [participants])

  if (!activities || activities.length === 0) {
    return (
      <Text color="gray.500" fontSize="sm" textAlign="center" py={4}>
        {t('pms.noActivities')}
      </Text>
    )
  }

  const formatField = (field: string): string => {
    const key = `pms.field_${field}`
    const translated = t(key)
    return translated === key ? field : translated
  }

  const formatValue = (value: string, field: string): string => {
    if (!value || value === '<nil>' || value === '') return t('pms.empty')
    // Try to translate status values
    if (field === 'status') {
      const key = `pms.taskStatus${value.split('_').map(p => p.charAt(0).toUpperCase() + p.slice(1)).join('')}`
      const translated = t(key)
      return translated === key ? value : translated
    }
    // sprintId / userStoryId: 历史数据可能是纯数字 ID，显示 #{id}；新数据已是标题
    if (field === 'sprintId' || field === 'userStoryId') {
      if (/^\d+$/.test(value)) return `#${value}`
      return value
    }
    return value
  }

  return (
    <VStack spacing={0} align="stretch">
      {activities.map((activity, index) => {
        const Icon = actionIcons[activity.action] || FiCheck
        const color = actionColors[activity.action] || 'gray'
        const isLast = index === activities.length - 1

        return (
          <Box key={index} position="relative" pb={isLast ? 0 : 4}>
            <Flex gap={3}>
              {/* Timeline dot + line */}
              <Flex direction="column" align="center">
                <Box
                  w="32px"
                  h="32px"
                  borderRadius="full"
                  bg={`${color}.100`}
                  color={`${color}.500`}
                  display="flex"
                  alignItems="center"
                  justifyContent="center"
                  flexShrink={0}
                >
                  <Icon size={14} />
                </Box>
                {!isLast && (
                  <Box w="2px" flex={1} bg="gray.200" minH="20px" mt={1} />
                )}
              </Flex>

              {/* Content */}
              <Box flex={1} pt={1}>
                <HStack spacing={2} mb={1} flexWrap="wrap" align="center">
                  <Avatar size="2xs" src={participantMap.get(activity.userName)?.image || undefined} name={activity.userName} />
                  <Text fontSize="sm" fontWeight="medium" color="gray.700">
                    {participantMap.get(activity.userName)?.fullName || activity.userName}
                  </Text>
                  <Text fontSize="sm" color="gray.500">
                    {t(`pms.action_${activity.action}`) || activity.action}
                  </Text>
                  {activity.field && (
                    <Badge colorScheme={color} variant="subtle" fontSize="2xs">
                      {formatField(activity.field)}
                    </Badge>
                  )}
                </HStack>

                {/* Value change */}
                {(activity.oldValue || activity.newValue) && (
                  <HStack spacing={2} fontSize="xs" color="gray.500" mb={1}>
                    {activity.oldValue && (
                      <Text as="span" textDecoration="line-through" color={activity.field === 'status' ? (getStatusHexColor(activity.oldValue) || `${getStatusColor(activity.oldValue)}.400`) : 'red.400'}>
                        {formatValue(activity.oldValue, activity.field)}
                      </Text>
                    )}
                    {activity.oldValue && activity.newValue && (
                      <Text as="span">→</Text>
                    )}
                    {activity.newValue && (
                      <Text as="span" color={activity.field === 'status' ? (getStatusHexColor(activity.newValue) || `${getStatusColor(activity.newValue)}.500`) : 'green.500'} fontWeight="medium">
                        {formatValue(activity.newValue, activity.field)}
                      </Text>
                    )}
                  </HStack>
                )}

                {/* Time */}
                <Text fontSize="2xs" color="gray.400">
                  {formatRelativeTime(activity.createdAt, t)}
                </Text>
              </Box>
            </Flex>
          </Box>
        )
      })}
    </VStack>
  )
}
