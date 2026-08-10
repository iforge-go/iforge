'use client'

import {
  Box,
  Button,
  Flex,
  Grid,
  Heading,
  HStack,
  Stack,
  Text,
  Badge,
  Divider,
  Avatar,
} from '@chakra-ui/react'
import { FiArrowLeft, FiEdit, FiTrash2, FiBook } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { MarkdownRenderer } from '@/components/MarkdownRenderer'
import { getStoryStatusColor } from '@/lib/storyStatus'
import UserSelect from '@/components/UserSelect'

const getPriorityLabelKey = (priority: string): string => {
  switch (priority) {
    case 'urgent': return 'pms.urgent'
    case 'high': return 'pms.high'
    case 'medium': return 'pms.medium'
    case 'low': return 'pms.low'
    default: return priority || 'pms.medium'
  }
}

const getPriorityColor = (priority: string) => {
  switch (priority) {
    case 'urgent': return 'red'
    case 'high': return 'orange'
    case 'medium': return 'yellow'
    case 'low': return 'green'
    default: return 'gray'
  }
}

interface StoryHeaderProps {
  story: any
  tasks: any[]
  onEditClick: () => void
  onDelete: () => void
  onBack: () => void
  canEditScrum: boolean
  onUpdateAssignee: (assigneeName: string | null) => Promise<void>
}

export default function StoryHeader({
  story,
  tasks,
  onEditClick,
  onDelete,
  onBack,
  canEditScrum,
  onUpdateAssignee,
}: StoryHeaderProps) {
  const { t } = useI18n()

  return (
    <>
      {/* 顶部操作栏 */}
      <Flex justify="space-between" align="center">
        <Button variant="ghost" onClick={onBack}>
          <FiArrowLeft style={{ marginRight: 8 }} />
          {t('common.back')}
        </Button>
        {canEditScrum && (
          <HStack spacing={2}>
            <Button colorScheme="blue" leftIcon={<FiEdit />} onClick={onEditClick}>
              {t('common.edit')}
            </Button>
            <Button colorScheme="red" leftIcon={<FiTrash2 />} onClick={onDelete}>
              {t('common.delete')}
            </Button>
          </HStack>
        )}
      </Flex>

      {/* 用户故事信息卡片 */}
      <Box bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" overflow="hidden">
        <Box px={6} py={4} borderBottom="1px solid #F4F6F8" bg="gray.50">
          <Flex justify="space-between" align="center">
            <HStack spacing={3}>
              <FiBook size={28} color="#805ad5" />
              <Heading as="h1" size="lg">{story.title}</Heading>
            </HStack>
            <HStack spacing={2}>
              <Badge borderWidth="1px" borderStyle="solid" bg="transparent" color={getStoryStatusColor(story.status)} borderColor={getStoryStatusColor(story.status)} px={3} py={1}>
                {t(`pms.status${story.status.split('_').map((p: string) => p.charAt(0).toUpperCase() + p.slice(1)).join('')}`)}
              </Badge>
              <Badge colorScheme={getPriorityColor(story.priority)} px={3} py={1}>
                {t(getPriorityLabelKey(story.priority))}
              </Badge>
            </HStack>
          </Flex>
        </Box>

        <Box p={6}>
          <Stack spacing={4}>
            {story.description && (
              <Box>
                <Text fontSize="sm" fontWeight="medium" color="gray.500" mb={2}>
                  {t('pms.userStoryDescription')}
                </Text>
                <MarkdownRenderer content={story.description} />
              </Box>
            )}

            {story.acceptanceCriteria && (
              <Box>
                <Text fontSize="sm" fontWeight="medium" color="gray.500" mb={2}>
                  {t('pms.acceptanceCriteria')}
                </Text>
                <MarkdownRenderer content={story.acceptanceCriteria} />
              </Box>
            )}

            <Divider />

            <Grid templateColumns={{ base: '1fr', md: 'repeat(3, 1fr)' }} gap={4}>
              <Box>
                <Text fontSize="sm" color="gray.500">{t('pms.storyPoints')}</Text>
                <Text fontSize="xl" fontWeight="bold" color="gray.900">
                  {story.storyPoints || 0}
                </Text>
              </Box>
              <Box>
                <Text fontSize="sm" color="gray.500">{t('pms.tasks')}</Text>
                <Text fontSize="xl" fontWeight="bold" color="gray.900">{tasks.length}</Text>
              </Box>
              <Box>
                <Text fontSize="sm" color="gray.500" mb={1}>{t('pms.assignee')}</Text>
                {canEditScrum ? (
                  <UserSelect
                    value={story.assigneeName || null}
                    onChange={async (u) => {
                      try {
                        await onUpdateAssignee(u?.userName || null)
                      } catch {
                        // 错误已由父组件 toast 处理
                      }
                    }}
                    size="sm"
                    buttonWidth="auto"
                  />
                ) : (
                  <HStack spacing={2}>
                    {story.assigneeName ? (
                      <>
                        <Avatar size="xs" name={story.assigneeName} bg="gray.200" />
                        <Text fontSize="md" fontWeight="medium" color="gray.900">{story.assigneeName}</Text>
                      </>
                    ) : (
                      <Text fontSize="md" color="gray.400">-</Text>
                    )}
                  </HStack>
                )}
              </Box>
            </Grid>
          </Stack>
        </Box>
      </Box>
    </>
  )
}
