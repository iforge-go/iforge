'use client'

import { Box, Flex, Text, Badge, VStack, HStack, Stack, Link, Button, IconButton } from '@chakra-ui/react'
import { FiEdit, FiPlay, FiTarget } from 'react-icons/fi'
import { useDroppable } from '@dnd-kit/core'
import { useI18n } from '@/contexts/I18nContext'
import BacklogStoryRow from './BacklogStoryRow'

export interface SprintColumnProps {
  sprints: any[]                       // 已过滤为 open/active
  storiesBySprint: Map<string, any[]>  // sprintSlug -> stories
  projectSlug: string
  onStoryClick: (story: any) => void
  canEditScrum: boolean
  highlightedStorySlug: string | null
  onStartSprint: (sprint: any) => void   // 开始迭代
  onEditSprint: (sprint: any) => void    // 编辑迭代
}

// 格式化 Sprint 日期范围
function formatSprintDateRange(startDate: string | null, endDate: string | null): string {
  const fmt = (d: string | null) => d ? new Date(d).toLocaleDateString('zh-CN', { month: 'numeric', day: 'numeric' }) : '—'
  return `${fmt(startDate)} - ${fmt(endDate)}`
}

// 上方活跃 Sprint 区:每个 Sprint 一个独立 droppable 区域
// Story 从 Backlog 拖到这里 = 规划进 Sprint;从这里拖到 Backlog = 移回需求池
export default function SprintColumn({
  sprints,
  storiesBySprint,
  projectSlug,
  onStoryClick,
  canEditScrum,
  highlightedStorySlug,
  onStartSprint,
  onEditSprint,
}: SprintColumnProps) {
  const { t } = useI18n()

  if (sprints.length === 0) {
    return (
      <Box p={4} textAlign="center" color="gray.400" fontSize="sm" borderRadius="lg" border="1px solid #DFE2EA" bg="white">
        {t('pms.noActiveSprint')}
      </Box>
    )
  }

  return (
    <VStack spacing={3} align="stretch">
      {sprints.map(sprint => (
        <SprintSection
          key={sprint.slug}
          sprint={sprint}
          stories={storiesBySprint.get(sprint.slug) || []}
          projectSlug={projectSlug}
          onStoryClick={onStoryClick}
          canEditScrum={canEditScrum}
          highlightedStorySlug={highlightedStorySlug}
          onStartSprint={onStartSprint}
          onEditSprint={onEditSprint}
        />
      ))}
    </VStack>
  )
}

function SprintSection({ sprint, stories, projectSlug, onStoryClick, canEditScrum, highlightedStorySlug, onStartSprint, onEditSprint }: {
  sprint: any
  stories: any[]
  projectSlug: string
  onStoryClick: (story: any) => void
  canEditScrum: boolean
  highlightedStorySlug: string | null
  onStartSprint: (sprint: any) => void
  onEditSprint: (sprint: any) => void
}) {
  const { t } = useI18n()
  const { setNodeRef, isOver } = useDroppable({
    id: `sprint-${sprint.slug}`,
    data: { type: 'sprint', sprintSlug: sprint.slug },
  })

  // 故事点统计:done/total
  const totalPoints = stories.reduce((sum, s) => sum + (s.storyPoints || 0), 0)
  const donePoints = stories
    .filter(s => s.status === 'done' || s.status === 'closed' || s.status === 'completed')
    .reduce((sum, s) => sum + (s.storyPoints || 0), 0)

  const statusLabel = sprint.status === 'active' ? t('pms.statusActive')
    : sprint.status === 'closed' ? t('pms.statusClosed')
    : t('pms.statusOpen')
  const statusColor = sprint.status === 'active' ? 'green'
    : sprint.status === 'closed' ? 'red'
    : 'gray'

  return (
    <Box
      ref={setNodeRef}
      borderRadius="lg"
      border={isOver ? '2px dashed #3b82f6' : '1px solid #DFE2EA'}
      bg={isOver ? 'blue.50' : 'white'}
      transition="all 0.2s ease"
      overflow="hidden"
    >
      {/* Sprint 标题行 */}
      <Flex align="center" px={4} py={2} bg="gray.50" borderBottom="1px solid #F4F6F8">
        <Link href={`/projects/${projectSlug}/sprint/${sprint.slug}`} style={{ textDecoration: 'none' }} _hover={{ textDecoration: 'underline' }}>
          <Text fontWeight="bold" fontSize="sm" color="gray.800">{sprint.title}</Text>
        </Link>
        <Text fontSize="xs" color="gray.500" ml={2}>
          {formatSprintDateRange(sprint.startDate, sprint.endDate)}
        </Text>
        {/* 操作按钮（canEditScrum 时显示） */}
        {canEditScrum && (
          <HStack spacing={1} ml="auto" onClick={(e) => e.stopPropagation()}>
            {sprint.status === 'open' && (
              <Button
                size="xs"
                colorScheme="blue"
                variant="outline"
                leftIcon={<FiPlay size={12} />}
                onClick={() => onStartSprint(sprint)}
              >
                {t('pms.startSprint')}
              </Button>
            )}
            <IconButton
              aria-label={t('pms.editSprint')}
              icon={<FiEdit size={12} />}
              size="xs"
              variant="ghost"
              color="gray.400"
              _hover={{ color: 'blue.500' }}
              onClick={() => onEditSprint(sprint)}
            />
          </HStack>
        )}
        {!canEditScrum && (
          <Badge ml="auto" colorScheme={statusColor} fontSize="xs">{statusLabel}</Badge>
        )}
        {canEditScrum && (
          <Badge colorScheme={statusColor} fontSize="xs" ml={3}>{statusLabel}</Badge>
        )}
        <Text fontSize="xs" color="purple.500" fontWeight="bold" ml={3}>
          {donePoints}/{totalPoints} pts
        </Text>
      </Flex>

      {/* Sprint 目标（规划时可见的承诺锚点） */}
      {sprint.goal && (
        <Flex px={4} py={2} bg="blue.50" borderBottom="1px solid #F4F6F8" align="flex-start">
          <Box color="blue.400" mt="2px" mr={2} flexShrink={0}>
            <FiTarget size={13} />
          </Box>
          <Text fontSize="xs" color="gray.600" noOfLines={2}>
            {sprint.goal}
          </Text>
        </Flex>
      )}

      {/* Story 列表 */}
      <Stack spacing={0}>
        {stories.map(story => (
          <BacklogStoryRow
            key={story.slug}
            story={story}
            onClick={() => onStoryClick(story)}
            canDrag={canEditScrum}
            isHighlighted={highlightedStorySlug === story.slug}
            showEpicBadge
          />
        ))}
        {stories.length === 0 && (
          <Text fontSize="sm" color="gray.400" py={4} textAlign="center">
            {t('pms.noStoriesInSprint')}
          </Text>
        )}
      </Stack>
    </Box>
  )
}
