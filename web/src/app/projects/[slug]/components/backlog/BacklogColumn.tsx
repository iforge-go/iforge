'use client'

import { Box, Flex, Text, Stack } from '@chakra-ui/react'
import { useDroppable } from '@dnd-kit/core'
import { useI18n } from '@/contexts/I18nContext'
import BacklogStoryRow from './BacklogStoryRow'

export interface BacklogColumnProps {
  stories: any[]                       // sprintSlug=null 的 Story
  onStoryClick: (story: any) => void
  canEditScrum: boolean
  highlightedStorySlug: string | null
}

// 下方需求池区:单一 droppable,id='backlog'
// Story 从 Sprint 拖到这里 = 移回需求池;从这里拖到 Sprint = 规划进 Sprint
export default function BacklogColumn({
  stories,
  onStoryClick,
  canEditScrum,
  highlightedStorySlug,
}: BacklogColumnProps) {
  const { t } = useI18n()
  const { setNodeRef, isOver } = useDroppable({
    id: 'backlog',
    data: { type: 'backlog', sprintSlug: null },
  })

  const totalPoints = stories.reduce((sum, s) => sum + (s.storyPoints || 0), 0)

  return (
    <Box
      ref={setNodeRef}
      minH="200px"
      borderRadius="lg"
      border={isOver ? '2px dashed #3b82f6' : '1px solid #DFE2EA'}
      bg={isOver ? 'blue.50' : 'white'}
      transition="all 0.2s ease"
      overflow="hidden"
    >
      {/* Backlog 标题行 */}
      <Flex align="center" px={4} py={2} bg="gray.50" borderBottom="1px solid #F4F6F8">
        <Text fontWeight="bold" fontSize="sm" color="gray.800">{t('pms.backlog')}</Text>
        <Text fontSize="xs" color="gray.500" ml={2}>
          {stories.length} {t('pms.storiesUnit')} · {totalPoints} {t('pms.storyPoints')}
        </Text>
      </Flex>

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
          <Text fontSize="sm" color="gray.400" py={6} textAlign="center">
            {t('pms.noBacklogStories')}
          </Text>
        )}
      </Stack>
    </Box>
  )
}
