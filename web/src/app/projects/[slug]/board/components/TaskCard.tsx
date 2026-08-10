'use client'

import {
  Box,
  Flex,
  Text,
  Badge,
  Avatar,
  HStack,
  Tooltip,
} from '@chakra-ui/react'
import { keyframes } from '@emotion/react'
import { FiMessageSquare, FiCheckSquare, FiBook, FiAlertCircle } from 'react-icons/fi'
import NextLink from 'next/link'
import {
  useSortable,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useI18n } from '@/contexts/I18nContext'
import { getPriorityBorderColor } from '@/lib/taskPriority'
import { getTaskTypeIcon } from '@/lib/taskType'
import { getEpicColor } from '@/lib/epicColor'

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
  subtaskCount?: number
  subtaskCompletedCount?: number
  position?: number
  // Epic 关联信息(后端 sideload 填充),用于卡片顶部显示 Epic 彩色标签
  epicSlug?: string | null
  epicTitle?: string | null
  // UserStory 关联信息(后端 sideload 填充),用于卡片显示所属故事徽标(任务→故事可视链接)
  userStorySlug?: string | null
  userStoryTitle?: string | null
  // 未完成前置依赖数(后端 sideload 填充),>0 时卡片显示阻塞标记
  blockedByCount?: number
  projectSlug?: string
}

function getPriorityLabelKey(priority: string): string {
  switch (priority) {
    case 'urgent': return 'pms.urgent'
    case 'high': return 'pms.high'
    case 'medium': return 'pms.medium'
    case 'low': return 'pms.low'
    default: return priority || 'pms.medium'
  }
}

// 任务编辑后高亮动画：黄色闪烁后渐变为淡蓝色背景，持续数秒后淡出（与需求池用户故事高亮一致）
const highlightPulse = keyframes`
  0% { background-color: #fef3c7; box-shadow: inset 4px 0 0 #f59e0b; }
  30% { background-color: #fef3c7; box-shadow: inset 4px 0 0 #f59e0b; }
  100% { background-color: #ebf8ff; box-shadow: inset 4px 0 0 #3182ce; }
`

interface TaskCardProps {
  task: TaskItem
  onClick: () => void
  isHighlighted?: boolean
  // 聚合视图下显示的 Sprint 名称(用于在卡片标注所属 Sprint)
  sprintTitle?: string
}

// --- Sortable Task Card ---
export default function SortableTaskCard({
  task,
  onClick,
  isHighlighted,
  sprintTitle,
}: TaskCardProps) {
  const { t } = useI18n()
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: task.taskId,
    data: { status: task.status },
  })

  const style = {
    transform: CSS.Translate.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
  }

  const typeInfo = getTaskTypeIcon(task.taskType)
  const borderColor = getPriorityBorderColor(task.priority)

  return (
    <Box
      ref={setNodeRef}
      style={style}
      bg={isHighlighted ? 'blue.50' : 'white'}
      borderRadius="8px"
      borderWidth="1px"
      borderColor={isHighlighted ? '#3182ce' : '#e2e8f0'}
      cursor="grab"
      _active={{ cursor: 'grabbing' }}
      _hover={{
        borderColor: '#94a3b8',
        boxShadow: '0 1px 3px rgba(0,0,0,0.08)',
      }}
      transition="all 0.15s ease"
      onClick={onClick}
      position="relative"
      overflow="hidden"
      sx={isHighlighted ? { animation: `${highlightPulse} 1.2s ease-out` } : undefined}
      {...attributes}
      {...listeners}
    >
      {/* Left priority border */}
      <Box
        position="absolute"
        left={0}
        top={0}
        bottom={0}
        w="3px"
        bg={borderColor}
      />

      <Box pl="14px" pr="12px" py="10px">
        {/* Top row: type icon + task ID */}
        <Flex align="center" justify="space-between" mb="6px">
          <HStack spacing="6px">
            <Text fontSize="13px" lineHeight="1">{typeInfo.label}</Text>
            <Text fontSize="11px" color="#94a3b8" fontWeight="500">
              TASK-{task.taskId}
            </Text>
          </HStack>
          {task.storyPoints != null && task.storyPoints > 0 && (
            <Box
              bg="#f1f5f9"
              color="#475569"
              fontSize="10px"
              fontWeight="600"
              px="5px"
              py="1px"
              borderRadius="4px"
            >
              {task.storyPoints}
            </Box>
          )}
        </Flex>

        {/* Epic 彩色标签 + Sprint Badge + Story Badge(参考 Jira:卡片顶部显示归属,强化在看板的存在感) */}
        {((task.epicSlug && task.epicTitle) || sprintTitle || (task.userStorySlug && task.userStoryTitle)) && (
          <HStack spacing="6px" mb="6px" align="center">
            {task.epicSlug && task.epicTitle && (
              <Box
                bg={getEpicColor(task.epicSlug)}
                color="white"
                fontSize="10px"
                fontWeight="600"
                px="6px"
                py="1px"
                borderRadius="3px"
                maxW="100%"
                overflow="hidden"
                textOverflow="ellipsis"
                whiteSpace="nowrap"
              >
                {task.epicTitle}
              </Box>
            )}
            {sprintTitle && (
              <Badge
                colorScheme="blue"
                variant="subtle"
                fontSize="10px"
                fontWeight="600"
                px="6px"
                py="0"
                borderRadius="3px"
                maxW="100%"
                overflow="hidden"
                textOverflow="ellipsis"
                whiteSpace="nowrap"
              >
                {sprintTitle}
              </Badge>
            )}
            {task.userStorySlug && task.userStoryTitle && (
              <NextLink
                href={`/projects/${task.projectSlug}/story/${task.userStorySlug}`}
                onClick={(e) => e.stopPropagation()}
                onPointerDown={(e) => e.stopPropagation()}
              >
                <Badge
                  colorScheme="purple"
                  variant="subtle"
                  fontSize="10px"
                  fontWeight="600"
                  px="6px"
                  py="0"
                  borderRadius="3px"
                  maxW="100%"
                  overflow="hidden"
                  textOverflow="ellipsis"
                  whiteSpace="nowrap"
                  cursor="pointer"
                  _hover={{ opacity: 0.8 }}
                >
                  <Flex as="span" align="center" gap="3px">
                    <FiBook size={9} />
                    {task.userStoryTitle}
                  </Flex>
                </Badge>
              </NextLink>
            )}
          </HStack>
        )}

        {/* Title */}
        <Text
          fontSize="13px"
          fontWeight="500"
          color="#1e293b"
          lineHeight="1.4"
          noOfLines={2}
          mb={task.description ? '4px' : 0}
        >
          {task.title}
        </Text>

        {/* Description preview */}
        {task.description && (
          <Text
            fontSize="12px"
            color="#94a3b8"
            lineHeight="1.4"
            noOfLines={2}
            mb="8px"
          >
            {task.description}
          </Text>
        )}

        {/* Bottom row: assignee + meta */}
        <Flex align="center" justify="space-between" mt="6px">
          <HStack spacing="6px">
            {(task.assignees && task.assignees.length > 0) ? (
              task.assignees.slice(0, 3).map((a, i) => (
                <Tooltip key={a.userName + i} label={a.fullName || a.userName} fontSize="11px">
                  <Avatar
                    size="xs"
                    name={a.fullName || a.userName}
                    src={a.image || undefined}
                    fontSize="9px"
                    ml={i > 0 ? '-6px' : '0'}
                    borderWidth="1px"
                    borderColor="white"
                  />
                </Tooltip>
              ))
            ) : task.assigneeName ? (
              <Tooltip label={task.assigneeName} fontSize="11px">
                <Avatar
                  size="xs"
                  name={task.assigneeName}
                  fontSize="9px"
                />
              </Tooltip>
            ) : (
              <Box
                w="20px"
                h="20px"
                borderRadius="full"
                bg="#f1f5f9"
                display="flex"
                alignItems="center"
                justifyContent="center"
              >
                <Text fontSize="10px" color="#cbd5e1">?</Text>
              </Box>
            )}
          </HStack>

          <HStack spacing="8px">
            {task.blockedByCount != null && task.blockedByCount > 0 && (
              <Tooltip label={t('pms.blockedByTooltip', { count: task.blockedByCount })} fontSize="11px">
                <HStack spacing="3px">
                  <FiAlertCircle size={12} color="#f59e0b" />
                  <Text fontSize="11px" color="#f59e0b" fontWeight="600">{task.blockedByCount}</Text>
                </HStack>
              </Tooltip>
            )}
            {task.commentCount != null && task.commentCount > 0 && (
              <HStack spacing="3px">
                <FiMessageSquare size={12} color="#94a3b8" />
                <Text fontSize="11px" color="#94a3b8">{task.commentCount}</Text>
              </HStack>
            )}
            {task.subtaskCount != null && task.subtaskCount > 0 && (
              <Tooltip label={t('pms.subtaskProgress', { completed: task.subtaskCompletedCount || 0, total: task.subtaskCount })} fontSize="11px">
                <HStack spacing="3px">
                  <FiCheckSquare size={12} color={task.subtaskCompletedCount === task.subtaskCount ? '#22c55e' : '#94a3b8'} />
                  <Text fontSize="11px" color={task.subtaskCompletedCount === task.subtaskCount ? '#22c55e' : '#94a3b8'}>
                    {task.subtaskCompletedCount || 0}/{task.subtaskCount}
                  </Text>
                </HStack>
              </Tooltip>
            )}
            <Badge
              fontSize="10px"
              fontWeight="500"
              color={getPriorityBorderColor(task.priority)}
              bg={`${getPriorityBorderColor(task.priority)}15`}
              px="5px"
              py="0"
              borderRadius="4px"
              variant="unstyled"
              display="flex"
              alignItems="center"
            >
              {t(getPriorityLabelKey(task.priority))}
            </Badge>
          </HStack>
        </Flex>
      </Box>
    </Box>
  )
}
