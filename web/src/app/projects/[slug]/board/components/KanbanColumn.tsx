'use client'

import type React from 'react'
import {
  Box,
  Flex,
  Text,
  HStack,
  Tooltip,
  IconButton,
  VStack,
} from '@chakra-ui/react'
import { FiPlus } from 'react-icons/fi'
import { useDroppable } from '@dnd-kit/core'
import { useI18n } from '@/contexts/I18nContext'

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
  position?: number
}

interface ColumnDef {
  id: number
  status: string
  name: string
  color: string
  // 看板列在制品上限(对齐 Jira Kanban WIP);null/undefined=不限制,>0 时列头显示 N/Limit 超限标红
  wipLimit?: number | null
}

interface KanbanColumnProps {
  column: ColumnDef
  tasks: TaskItem[]
  isOver: boolean
  onAddTask?: () => void
  children: React.ReactNode
}

// --- Column Component ---
export default function KanbanColumn({
  column,
  tasks,
  isOver,
  onAddTask,
  children,
}: KanbanColumnProps) {
  const { t } = useI18n()
  const { setNodeRef: setDroppableRef } = useDroppable({
    id: `col-${column.status}`,
    data: { status: column.status },
  })

  // WIP(在制品上限)计算:对齐 Jira Kanban WIP。limit>0 时列头显示 N/Limit,超限标红提醒。
  const taskCount = tasks.length
  const wipLimit = column.wipLimit
  const hasWipLimit = wipLimit != null && wipLimit > 0
  const overLimit = hasWipLimit && taskCount > (wipLimit as number)

  return (
    <Flex
      direction="column"
      flex="1"
      minW="200px"
      bg={isOver ? '#f0f7ff' : '#f8fafc'}
      borderRadius="12px"
      border={isOver ? '2px dashed #3b82f6' : '1px solid #e2e8f0'}
      transition="all 0.2s ease"
      maxH="calc(100vh - 200px)"
      data-status={column.status}
    >
      {/* Column Header */}
      <Box
        px="14px"
        py="12px"
        borderBottom={isOver ? 'none' : '1px solid #e2e8f0'}
        bg={isOver ? '#e8f2ff' : 'transparent'}
        borderRadius={isOver ? '10px 10px 0 0' : '12px 12px 0 0'}
      >
        <Flex align="center" justify="space-between">
          <HStack spacing="8px">
            <Box
              w="8px"
              h="8px"
              borderRadius="full"
              bg={column.color}
            />
            <Text
              fontSize="13px"
              fontWeight="600"
              color="#334155"
              letterSpacing="0.02em"
            >
              {column.name}
            </Text>
            <Box
              bg={overLimit ? '#fee2e2' : '#e2e8f0'}
              color={overLimit ? '#dc2626' : '#64748b'}
              fontSize="11px"
              fontWeight="600"
              minW="20px"
              h="20px"
              borderRadius="10px"
              display="flex"
              alignItems="center"
              justifyContent="center"
              px="6px"
            >
              {hasWipLimit ? `${taskCount}/${wipLimit}` : taskCount}
            </Box>
          </HStack>
          {onAddTask && (
            <Tooltip label={t('pms.addTask')} fontSize="11px">
              <IconButton
                aria-label={t('pms.addTask')}
                icon={<FiPlus size={16} />}
                size="xs"
                variant="ghost"
                color="#94a3b8"
                _hover={{ color: '#3b82f6', bg: 'transparent' }}
                onClick={onAddTask}
              />
            </Tooltip>
          )}
        </Flex>
      </Box>

      {/* Cards Area */}
      <Box
        ref={setDroppableRef}
        flex="1"
        overflowY="auto"
        px="8px"
        py="8px"
        css={{
          '&::-webkit-scrollbar': { width: '4px' },
          '&::-webkit-scrollbar-track': { background: 'transparent' },
          '&::-webkit-scrollbar-thumb': { background: '#cbd5e1', borderRadius: '4px' },
        }}
      >
        <VStack spacing="6px" align="stretch">
          {children}
        </VStack>

        {/* Empty State */}
        {tasks.length === 0 && (
          <Flex
            justify="center"
            align="center"
            py="40px"
            direction="column"
            gap="8px"
          >
            <Box
              w="40px"
              h="40px"
              borderRadius="10px"
              bg="#e2e8f0"
              display="flex"
              alignItems="center"
              justifyContent="center"
            >
              <Text fontSize="18px" color="#94a3b8">
                ○
              </Text>
            </Box>
            <Text fontSize="12px" color="#94a3b8">
              {t('pms.noTasks')}
            </Text>
          </Flex>
        )}
      </Box>
    </Flex>
  )
}
