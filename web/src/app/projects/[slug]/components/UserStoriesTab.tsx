'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import {
  Box,
  Button,
  Flex,
  Grid,
  Stack,
  Text,
  Badge,
  useToast,
} from '@chakra-ui/react'
import { keyframes } from '@emotion/react'
import { FiPlus, FiCheckSquare } from 'react-icons/fi'
import { api } from '@/lib/api'
import { getStoryStatusColor } from '@/lib/storyStatus'
import { useI18n } from '@/contexts/I18nContext'
import UserStoryDrawer, { UserStoryFormData } from '@/components/UserStoryDrawer'

// 新建故事高亮动画：黄色闪烁后渐变为淡蓝色背景，持续数秒后淡出
const highlightPulse = keyframes`
  0% { background-color: #fef3c7; box-shadow: inset 4px 0 0 #f59e0b; }
  30% { background-color: #fef3c7; box-shadow: inset 4px 0 0 #f59e0b; }
  100% { background-color: #ebf8ff; box-shadow: inset 4px 0 0 #3182ce; }
`

interface UserStoriesTabProps {
  projectSlug: string
  userStories: any[]
  tasks: any[]
  onUserStoriesChange: (stories: any[]) => void
  onTasksChange: (tasks: any[]) => void
}

export default function UserStoriesTab({ projectSlug, userStories, tasks, onUserStoriesChange, onTasksChange }: UserStoriesTabProps) {
  const { t } = useI18n()
  const toast = useToast()

  // 用户故事 Drawer：共享 UserStoryDrawer 组件
  const [storyDrawerOpen, setStoryDrawerOpen] = useState(false)
  const [storyDrawerMode, setStoryDrawerMode] = useState<'view' | 'create' | 'edit'>('create')
  const [editingStory, setEditingStory] = useState<any>(null)

  const storyList = userStories || []
  const taskList = tasks || []

  // 新建故事高亮：记录刚创建故事的 slug，用于在列表中高亮显示
  const [highlightedStorySlug, setHighlightedStorySlug] = useState<string | null>(null)

  // 高亮 5 秒后自动清除
  useEffect(() => {
    if (!highlightedStorySlug) return
    const timer = setTimeout(() => setHighlightedStorySlug(null), 5000)
    return () => clearTimeout(timer)
  }, [highlightedStorySlug])

  const getStatusName = (slug: string): string => {
    const camelSlug = slug.split('_').map(part => part.charAt(0).toUpperCase() + part.slice(1)).join('')
    const key = `pms.taskStatus${camelSlug}`
    const translated = t(key)
    return translated === key ? slug : translated
  }

  const getPriorityLabelKey = (priority: string): string => {
    switch (priority) {
      case 'urgent': return 'pms.urgent'
      case 'high': return 'pms.high'
      case 'medium': return 'pms.medium'
      case 'low': return 'pms.low'
      default: return priority || 'pms.medium'
    }
  }

  const openDrawer = (mode: 'view' | 'create' | 'edit' = 'view', editingItem: any = null) => {
    setStoryDrawerMode(mode)
    setEditingStory(editingItem)
    setStoryDrawerOpen(true)
  }

  const handleSaveStory = async (data: UserStoryFormData) => {
    if (!data.title?.trim()) {
      toast({
        title: t('common.error'),
        description: t('pms.inputUserStoryTitle'),
        status: 'error',
        duration: 2000,
      })
      return
    }

    try {
      const payload = {
        title: data.title,
        description: data.description || '',
        status: data.status,
        priority: data.priority,
        storyPoints: data.storyPoints || 0,
        acceptanceCriteria: data.acceptanceCriteria || '',
      }

      if (storyDrawerMode === 'edit' && editingStory) {
        const updated = await api.updateUserStory(editingStory.slug, payload)
        // 编辑保存后同样高亮，让用户确认改动已生效
        setHighlightedStorySlug(updated.slug || editingStory.slug)
      } else {
        const created = await api.createUserStory(projectSlug, payload)
        // 高亮新建故事，让用户一眼看到
        setHighlightedStorySlug(created.slug)
      }

      api.getUserStories(projectSlug).then((data) => onUserStoriesChange(data || []))
      api.getTasks(projectSlug).then((data) => onTasksChange(data || []))
    } catch (error: any) {
      toast({
        title: t('common.error'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  return (
    <>
      <Flex justify="space-between" align="center" mb={4}>
        <Text fontSize="sm" color="gray.500">{t('pms.userStories')} ({storyList.length})</Text>
        <Button size="sm" colorScheme="blue" leftIcon={<FiPlus />} onClick={() => openDrawer('create')}>
          {t('pms.newUserStory')}
        </Button>
      </Flex>
      {storyList && storyList.length > 0 ? (
        <Stack spacing={2}>
          {storyList.map((story: any) => {
            const storyTasks = taskList.filter((t: any) => t.userStorySlug === story.slug) || []
            const isHighlighted = highlightedStorySlug === story.slug
            return (
              <Link
                key={story.slug}
                href={`/projects/${projectSlug}/story/${story.slug}`}
                style={{ textDecoration: 'none' }}
              >
                <Box
                  p={3}
                  bg={isHighlighted ? 'blue.50' : 'gray.50'}
                  borderRadius="md"
                  cursor="pointer"
                  _hover={{ bg: 'gray.100' }}
                  transition="background-color 0.6s ease"
                  sx={isHighlighted ? {
                    animation: `${highlightPulse} 1.2s ease-out`,
                  } : undefined}
                >
                  <Flex justify="space-between" align="center">
                    <Box>
                      <Text fontWeight="medium">{story.title}</Text>
                      <Flex gap={4} mt={1}>
                        <Text fontSize="sm" color="gray.500">{t('common.priority')}: {t(getPriorityLabelKey(story.priority))}</Text>
                        {story.status && (
                          <Text fontSize="sm" color="gray.500">{t('common.status')}: {getStatusName(story.status)}</Text>
                        )}
                        <Text fontSize="sm" color="purple.600">
                          <FiCheckSquare size={12} style={{ marginRight: 4 }} /> {storyTasks.length} {t('pms.tasks')}
                        </Text>
                      </Flex>
                    </Box>
                    <Badge borderWidth="1px" borderStyle="solid" bg="transparent" color={getStoryStatusColor(story.status || 'open')} borderColor={getStoryStatusColor(story.status || 'open')}>
                      {getStatusName(story.status || 'open')}
                    </Badge>
                  </Flex>
                </Box>
              </Link>
            )
          })}
        </Stack>
      ) : (
        <Text color="gray.500">{t('pms.noUserStories')}</Text>
      )}

      {/* 用户故事 Drawer：复用共享 UserStoryDrawer 组件 */}
      <UserStoryDrawer
        isOpen={storyDrawerOpen}
        onClose={() => setStoryDrawerOpen(false)}
        mode={storyDrawerMode}
        projectSlug={projectSlug}
        initialData={editingStory ? {
          title: editingStory.title,
          description: editingStory.description || '',
          status: editingStory.status,
          priority: editingStory.priority,
          storyPoints: editingStory.storyPoints || 0,
          acceptanceCriteria: editingStory.acceptanceCriteria || '',
        } : undefined}
        onSave={handleSaveStory}
        onEdit={() => setStoryDrawerMode('edit')}
      />
    </>
  )
}
