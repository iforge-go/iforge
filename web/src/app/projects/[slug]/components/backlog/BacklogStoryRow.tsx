'use client'

import { Flex, Box, Text, Badge } from '@chakra-ui/react'
import { useDraggable } from '@dnd-kit/core'
import { CSS } from '@dnd-kit/utilities'
import NextLink from 'next/link'
import { FiCheckSquare, FiPlus } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { getEpicColor, getEpicBgColor } from '@/lib/epicColor'

// 优先级色点颜色(与 BacklogTab 旧实现一致)
const PRIORITY_DOT_COLORS: Record<string, string> = {
  urgent: '#e53e3e',
  high: '#d69e2e',
  medium: '#3182ce',
  low: '#718096',
}

export interface BacklogStoryRowProps {
  story: any
  onClick: () => void
  canDrag: boolean
  isHighlighted: boolean
  showEpicBadge: boolean
}

// 可拖拽的 Story 行(Backlog/Sprint 区通用)
// 用 useDraggable 而非 useSortable,因为 v1 不支持区域内排序(UserStory 无 position 字段)
export default function BacklogStoryRow({
  story,
  onClick,
  canDrag,
  isHighlighted,
  showEpicBadge,
}: BacklogStoryRowProps) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: story.slug,
    data: { storySlug: story.slug, currentSprintSlug: story.sprintSlug },
    disabled: !canDrag,
  })

  const style: React.CSSProperties = {
    transform: CSS.Translate.toString(transform),
    opacity: isDragging ? 0.4 : 1,
  }

  const { t } = useI18n()
  const priorityColor = PRIORITY_DOT_COLORS[story.priority || 'medium'] || PRIORITY_DOT_COLORS.medium
  const epicColor = story.epicSlug ? getEpicColor(story.epicSlug) : null

  return (
    <Flex
      ref={setNodeRef}
      style={style}
      align="center"
      px={3}
      py={2}
      bg={isHighlighted ? 'blue.50' : 'white'}
      borderBottom="1px solid #F4F6F8"
      cursor={canDrag ? 'grab' : 'pointer'}
      _hover={{ bg: 'gray.50' }}
      _active={canDrag ? { cursor: 'grabbing' } : undefined}
      onClick={onClick}
      {...(canDrag ? attributes : {})}
      {...(canDrag ? listeners : {})}
    >
      {/* 优先级色点 */}
      <Box w="8px" h="8px" borderRadius="full" bg={priorityColor} flexShrink={0} mr={2} />

      {/* 标题 */}
      <Text flex={1} fontSize="sm" fontWeight="medium" color="gray.700" noOfLines={1}>
        {story.title}
      </Text>

      {/* 故事点 */}
      {story.storyPoints != null && story.storyPoints > 0 && (
        <Badge colorScheme="purple" variant="subtle" fontSize="xs" lineHeight="1" ml={2}>
          {story.storyPoints} pt
        </Badge>
      )}

      {/* Epic 彩色标签 */}
      {showEpicBadge && story.epicSlug && story.epicTitle && epicColor && (
        <Badge
          borderWidth="1px"
          borderStyle="solid"
          bg={getEpicBgColor(epicColor)}
          color={epicColor}
          borderColor={epicColor}
          fontSize="xs"
          fontWeight="medium"
          px={1.5}
          ml={2}
          maxW="120px"
          noOfLines={1}
        >
          {story.epicTitle}
        </Badge>
      )}

      {/* 任务数徽标 / 未拆解提示:点击进入故事详情页拆解(故事→任务可视链接 + 拆解入口) */}
      <NextLink
        href={`/projects/${story.projectSlug}/story/${story.slug}`}
        onClick={(e) => e.stopPropagation()}
        onPointerDown={(e) => e.stopPropagation()}
        style={{ textDecoration: 'none', display: 'flex', alignItems: 'center' }}
      >
        {story.taskCount != null && story.taskCount > 0 ? (
          <Badge colorScheme="purple" variant="subtle" fontSize="xs" lineHeight="1" ml={2} cursor="pointer" _hover={{ opacity: 0.8 }}>
            <Flex as="span" align="center" gap={1}>
              <FiCheckSquare size={10} />
              {story.taskCount}
            </Flex>
          </Badge>
        ) : (
          <Badge colorScheme="orange" variant="subtle" fontSize="xs" lineHeight="1" ml={2} cursor="pointer" _hover={{ opacity: 0.8 }}>
            <Flex as="span" align="center" gap={1}>
              <FiPlus size={10} />
              {t('pms.notDecomposed')}
            </Flex>
          </Badge>
        )}
      </NextLink>
    </Flex>
  )
}
