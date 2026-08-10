'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import {
  Box,
  Button,
  Card,
  CardBody,
  CardHeader,
  Flex,
  Heading,
  Stack,
  Text,
  Badge,
  useToast,
  IconButton,
  HStack,
} from '@chakra-ui/react'
import { keyframes } from '@emotion/react'
import { FiPlus, FiCheckSquare, FiRepeat, FiEdit, FiBarChart2 } from 'react-icons/fi'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import SprintDrawer, { SprintFormData } from '@/components/SprintDrawer'
import VelocityChart from '@/components/VelocityChart'
import type { VelocityDataPoint } from '@/lib/types'
import { useProject } from '../ProjectContext'

interface SprintsTabProps {
  projectSlug: string
  sprints: any[]
  tasks: any[]
  onSprintsChange: (sprints: any[]) => void
}

// 编辑/新建高亮动画：黄色闪烁后渐变为淡蓝色背景，持续数秒后淡出
// 与需求池用户故事高亮保持一致
const highlightPulse = keyframes`
  0% { background-color: #fef3c7; box-shadow: inset 4px 0 0 #f59e0b; }
  30% { background-color: #fef3c7; box-shadow: inset 4px 0 0 #f59e0b; }
  100% { background-color: #ebf8ff; box-shadow: inset 4px 0 0 #3182ce; }
`

export default function SprintsTab({ projectSlug, sprints, tasks, onSprintsChange }: SprintsTabProps) {
  const { t } = useI18n()
  const toast = useToast()
  const { canEditScrum } = useProject()

  // Sprint Drawer：共享 SprintDrawer 组件
  const [sprintDrawerOpen, setSprintDrawerOpen] = useState(false)
  const [sprintDrawerMode, setSprintDrawerMode] = useState<'view' | 'create' | 'edit'>('create')
  const [editingSprint, setEditingSprint] = useState<any>(null)

  // 编辑/新建高亮：记录刚操作过的 sprint slug，用于在列表中高亮显示
  const [highlightedSprintSlug, setHighlightedSprintSlug] = useState<string | null>(null)

  // 速度图数据：最近 N 个已关闭 Sprint 的承诺/完成点数
  const [velocityData, setVelocityData] = useState<VelocityDataPoint[]>([])

  // 高亮 5 秒后自动清除
  useEffect(() => {
    if (!highlightedSprintSlug) return
    const timer = setTimeout(() => setHighlightedSprintSlug(null), 5000)
    return () => clearTimeout(timer)
  }, [highlightedSprintSlug])

  // 获取速度图数据（sprints 变化时刷新，如关闭 Sprint 后）
  useEffect(() => {
    api.getVelocity(projectSlug).then(setVelocityData).catch(() => setVelocityData([]))
  }, [projectSlug, sprints])

  const sprintList = sprints || []
  // 统计口径已由后端 ListSprints 返回(sprint.taskCount/storyCount/totalStoryPoints),
  // 兼容 Story 级 Sprint 规划(含父 Story 在该 Sprint 的继承 task),前端无需再按 sprintSlug 过滤 taskList。
  void tasks

  // Sprint 状态颜色（open/active/closed，统一使用 .500 色阶）
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'open': return 'gray.500'
      case 'active': return 'green.500'
      case 'closed': return 'red.500'
      default: return 'gray.500'
    }
  }

  // Sprint 状态翻译（与编辑表单的 i18n 键一致）
  const getSprintStatusLabel = (status: string) => {
    switch (status) {
      case 'open': return t('pms.statusOpen')
      case 'active': return t('pms.statusActive')
      case 'closed': return t('pms.statusClosed')
      default: return status
    }
  }

  const openDrawer = (mode: 'view' | 'create' | 'edit' = 'view', editingItem: any = null) => {
    setSprintDrawerMode(mode)
    setEditingSprint(editingItem)
    setSprintDrawerOpen(true)
  }

  const handleSaveSprint = async (data: SprintFormData) => {
    if (!data.title?.trim()) {
      toast({
        title: t('common.error'),
        description: t('pms.inputSprintTitle'),
        status: 'error',
        duration: 2000,
      })
      return
    }

    try {
      const payload = {
        title: data.title,
        description: data.description || '',
        goal: data.goal || '',
        status: data.status,
        startDate: data.startDate || undefined,
        endDate: data.endDate || undefined,
      }

      if (sprintDrawerMode === 'edit' && editingSprint) {
        const updated = await api.updateSprint(editingSprint.slug, payload)
        // 编辑保存后高亮，让用户确认改动已生效
        setHighlightedSprintSlug(updated.slug || editingSprint.slug)
      } else {
        const created = await api.createSprint(projectSlug, payload)
        // 高亮新建的迭代
        setHighlightedSprintSlug(created.slug)
      }

      api.getSprints(projectSlug).then((data) => onSprintsChange(data || []))
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
      <Flex justify="space-between" align="center">
        <Flex align="center" gap={3}>
          <FiRepeat color="#3182ce" />
          <Heading size="md">{t('pms.sprints')} ({sprintList.length})</Heading>
        </Flex>
        {canEditScrum && (
          <Button size="sm" colorScheme="blue" leftIcon={<FiPlus />} onClick={() => openDrawer('create')}>
            {t('pms.newSprint')}
          </Button>
        )}
      </Flex>

      {/* 速度图（Velocity Chart）：展示已关闭 Sprint 的承诺 vs 完成点数 */}
      {velocityData.length > 0 && (
        <Card bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1">
          <CardHeader pb={2}>
            <Flex align="center" gap={2}>
              <FiBarChart2 color="#3182ce" />
              <Heading size="sm">{t('pms.velocityChart')}</Heading>
            </Flex>
          </CardHeader>
          <CardBody pt={2}>
            <VelocityChart data={velocityData} />
          </CardBody>
        </Card>
      )}

      {sprintList && sprintList.length > 0 ? (
        <Box borderRadius="lg" border="1px solid #DFE2EA" bg="white" overflow="hidden">
          <Stack spacing={0}>
            {sprintList.map((sprint: any, idx: number) => {
              const isHighlighted = highlightedSprintSlug === sprint.slug
              return (
                <Box
                  key={sprint.slug}
                  p={3}
                  pl={4}
                  bg={isHighlighted ? 'blue.50' : 'white'}
                  cursor="pointer"
                  borderLeft="4px solid"
                  borderLeftColor={getStatusColor(sprint.status)}
                  _hover={{ bg: 'gray.50' }}
                  borderBottom={idx < sprintList.length - 1 ? '1px solid #F4F6F8' : 'none'}
                  transition="background-color 0.6s ease"
                  sx={isHighlighted ? {
                    animation: `${highlightPulse} 1.2s ease-out`,
                  } : undefined}
                >
                  <Flex justify="space-between" align="center">
                    <Link href={`/projects/${projectSlug}/sprint/${sprint.slug}`} style={{ textDecoration: 'none', flex: 1 }} onClick={(e) => e.stopPropagation()}>
                      <Box cursor="pointer">
                        <Text fontWeight="medium">{sprint.title}</Text>
                        <Flex gap={4} mt={1} align="center">
                          <Text fontSize="sm" color="gray.500">
                            {sprint.startDate || sprint.endDate
                              ? `${sprint.startDate ? new Date(sprint.startDate).toLocaleDateString() : '?'} - ${sprint.endDate ? new Date(sprint.endDate).toLocaleDateString() : '?'}`
                              : t('pms.noDatesSet')}
                          </Text>
                          <Text fontSize="sm" color="gray.500">
                            <FiCheckSquare size={12} style={{ marginRight: 4 }} /> {sprint.taskCount ?? 0} {t('pms.tasks')}
                          </Text>
                          <Text fontSize="sm" color="gray.500">
                            {sprint.storyCount ?? 0} {t('pms.stories')}
                          </Text>
                          <Text fontSize="sm" color="purple.500" fontWeight="medium">
                            {sprint.totalStoryPoints ?? 0} {t('pms.storyPoints')}
                          </Text>
                        </Flex>
                      </Box>
                    </Link>
                    <HStack spacing={2}>
                      <Badge borderWidth="1px" borderStyle="solid" bg="transparent" color={getStatusColor(sprint.status)} borderColor={getStatusColor(sprint.status)}>
                        {getSprintStatusLabel(sprint.status)}
                      </Badge>
                      {canEditScrum && (
                        <IconButton
                          aria-label={t('common.edit')}
                          icon={<FiEdit />}
                          size="xs"
                          variant="ghost"
                          onClick={() => openDrawer('edit', sprint)}
                        />
                      )}
                    </HStack>
                  </Flex>
                </Box>
              )
            })}
          </Stack>
        </Box>
      ) : (
        <Box borderRadius="lg" border="1px solid #DFE2EA" bg="white" p={6}>
          <Text color="gray.500">{t('pms.noSprints')}</Text>
        </Box>
      )}

      {/* Sprint Drawer：复用共享 SprintDrawer 组件 */}
      <SprintDrawer
        isOpen={sprintDrawerOpen}
        onClose={() => setSprintDrawerOpen(false)}
        mode={sprintDrawerMode}
        projectSlug={projectSlug}
        initialData={editingSprint ? {
          title: editingSprint.title,
          description: editingSprint.description || '',
          goal: editingSprint.goal || '',
          status: editingSprint.status,
          startDate: editingSprint.startDate || '',
          endDate: editingSprint.endDate || '',
        } : undefined}
        onSave={handleSaveSprint}
        onEdit={canEditScrum ? () => setSprintDrawerMode('edit') : undefined}
      />
    </>
  )
}
