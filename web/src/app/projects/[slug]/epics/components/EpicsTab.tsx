'use client'

import { useState, useMemo, useEffect } from 'react'
import Link from 'next/link'
import {
  Box,
  Button,
  Card,
  CardBody,
  Flex,
  HStack,
  IconButton,
  Input,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Progress,
  Select,
  Stack,
  Stat,
  StatGroup,
  StatLabel,
  StatNumber,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  Badge,
  useToast,
} from '@chakra-ui/react'
import { FiPlus, FiMoreVertical, FiEdit, FiTrash2, FiTarget } from 'react-icons/fi'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useProject } from '../../ProjectContext'
import { getStoryStatusColor } from '@/lib/storyStatus'
import EpicDrawer, { EpicFormData } from './EpicDrawer'
import { Epic } from '@/lib/types'
import { ConfirmDialog } from '@/components/ConfirmDialog'

interface EpicsTabProps {
  projectSlug: string
  epics: Epic[]
  onEpicsChange: (epics: Epic[]) => void
  newEpicTrigger?: number
}

export default function EpicsTab({ projectSlug, epics, onEpicsChange, newEpicTrigger }: EpicsTabProps) {
  const { t } = useI18n()
  const toast = useToast()
  const { canEditScrum, members } = useProject()

  const [isDrawerOpen, setIsDrawerOpen] = useState(false)
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create')
  const [editingEpic, setEditingEpic] = useState<Epic | null>(null)
  const [statusFilter, setStatusFilter] = useState('')
  const [priorityFilter, setPriorityFilter] = useState('')
  const [search, setSearch] = useState('')
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [confirmAction, setConfirmAction] = useState<(() => void) | null>(null)

  // 页面层"新建史诗"按钮触发：通过递增计数器驱动 drawer 打开
  useEffect(() => {
    if (newEpicTrigger && newEpicTrigger > 0) {
      setDrawerMode('create')
      setEditingEpic(null)
      setIsDrawerOpen(true)
    }
  }, [newEpicTrigger])

  // 统计
  const stats = useMemo(() => {
    const total = epics.length
    const inProgress = epics.filter(e => e.status === 'in_progress').length
    const done = epics.filter(e => e.status === 'done' || e.status === 'closed').length
    const totalPoints = epics.reduce((sum, e) => sum + (e.totalStoryPoints || 0), 0)
    return { total, inProgress, done, totalPoints }
  }, [epics])

  const filteredEpics = useMemo(() => {
    return epics.filter(e => {
      if (statusFilter && e.status !== statusFilter) return false
      if (priorityFilter && e.priority !== priorityFilter) return false
      if (search.trim()) {
        const q = search.toLowerCase().trim()
        if (!e.title.toLowerCase().includes(q)) return false
      }
      return true
    })
  }, [epics, statusFilter, priorityFilter, search])

  const handleCreateClick = () => {
    setDrawerMode('create')
    setEditingEpic(null)
    setIsDrawerOpen(true)
  }

  const handleEditClick = (epic: Epic) => {
    setDrawerMode('edit')
    setEditingEpic(epic)
    setIsDrawerOpen(true)
  }

  const handleSave = async (data: EpicFormData) => {
    try {
      if (drawerMode === 'create') {
        const created = await api.createEpic(projectSlug, {
          title: data.title,
          description: data.description || undefined,
          status: data.status || 'open',
          priority: data.priority || 'medium',
          goal: data.goal || undefined,
          startDate: data.startDate || undefined,
          targetDate: data.targetDate || undefined,
          ownerName: data.ownerName || undefined,
        })
        onEpicsChange([created, ...epics])
        toast({ title: t('common.success'), description: t('pms.epicCreated'), status: 'success', duration: 2000 })
      } else if (editingEpic) {
        const updated = await api.updateEpic(editingEpic.slug, {
          title: data.title,
          description: data.description || null,
          goal: data.goal || null,
          status: data.status,
          priority: data.priority,
          startDate: data.startDate || null,
          targetDate: data.targetDate || null,
          ownerName: data.ownerName || null,
        })
        onEpicsChange(epics.map(e => e.slug === updated.slug ? updated : e))
        toast({ title: t('common.success'), description: t('pms.epicUpdated'), status: 'success', duration: 2000 })
      }
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
      throw error
    }
  }

  const handleDelete = (epic: Epic) => {
    setConfirmAction(() => async () => {
      try {
        await api.deleteEpic(epic.slug)
        onEpicsChange(epics.filter(e => e.slug !== epic.slug))
        toast({ title: t('common.success'), status: 'success', duration: 2000 })
      } catch (error: any) {
        toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
      }
    })
    setConfirmOpen(true)
  }

  const handleConfirm = () => {
    confirmAction?.()
    setConfirmOpen(false)
    setConfirmAction(null)
  }

  // 状态显示
  const getStatusName = (status: string): string => {
    const map: Record<string, string> = {
      open: t('pms.statusOpen'),
      in_progress: t('pms.statusInProgress'),
      done: t('pms.statusDone'),
      closed: t('pms.statusClosed'),
    }
    return map[status] || status
  }

  const getPriorityName = (priority: string): string => {
    const map: Record<string, string> = {
      low: t('pms.priorityLow'),
      medium: t('pms.priorityMedium'),
      high: t('pms.priorityHigh'),
      urgent: t('pms.priorityUrgent'),
    }
    return map[priority] || priority
  }

  const getPriorityColor = (priority: string): string => {
    const map: Record<string, string> = {
      low: 'gray.500',
      medium: 'blue.500',
      high: 'orange.500',
      urgent: 'red.500',
    }
    return map[priority] || 'gray.500'
  }

  return (
    <Stack spacing={6}>
      {/* 统计卡片 */}
      <StatGroup>
        <Stat>
          <StatLabel color="gray.500" fontSize="xs">{t('pms.epicTotal')}</StatLabel>
          <StatNumber fontSize="2xl">{stats.total}</StatNumber>
        </Stat>
        <Stat>
          <StatLabel color="gray.500" fontSize="xs">{t('pms.statusInProgress')}</StatLabel>
          <StatNumber fontSize="2xl" color="blue.500">{stats.inProgress}</StatNumber>
        </Stat>
        <Stat>
          <StatLabel color="gray.500" fontSize="xs">{t('pms.statusDone')}</StatLabel>
          <StatNumber fontSize="2xl" color="green.500">{stats.done}</StatNumber>
        </Stat>
        <Stat>
          <StatLabel color="gray.500" fontSize="xs">{t('pms.totalStoryPoints')}</StatLabel>
          <StatNumber fontSize="2xl" color="purple.500">{stats.totalPoints}</StatNumber>
        </Stat>
      </StatGroup>

      {/* 过滤栏 */}
      <Flex gap={4} align="center">
        <Input
          placeholder={t('pms.searchEpicPlaceholder')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          maxW="300px"
        />
        <Select
          placeholder={t('common.allStatus')}
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          maxW="160px"
        >
          <option value="open">{t('pms.statusOpen')}</option>
          <option value="in_progress">{t('pms.statusInProgress')}</option>
          <option value="done">{t('pms.statusDone')}</option>
          <option value="closed">{t('pms.statusClosed')}</option>
        </Select>
        <Select
          placeholder={t('common.allPriorities')}
          value={priorityFilter}
          onChange={(e) => setPriorityFilter(e.target.value)}
          maxW="160px"
        >
          <option value="low">{t('pms.priorityLow')}</option>
          <option value="medium">{t('pms.priorityMedium')}</option>
          <option value="high">{t('pms.priorityHigh')}</option>
          <option value="urgent">{t('pms.priorityUrgent')}</option>
        </Select>
      </Flex>

      {/* Epic 表格 */}
      <Card bg="white" borderRadius="lg" border="1px solid" borderColor="myGray.200" overflow="hidden">
        <CardBody p={0}>
          {filteredEpics.length === 0 ? (
            <Box p={10} textAlign="center">
              <FiTarget size={40} style={{ margin: '0 auto', color: '#CBD5E0' }} />
              <Text mt={4} color="gray.500">{t('pms.noEpics')}</Text>
              {canEditScrum && (
                <Button mt={4} colorScheme="blue" leftIcon={<FiPlus />} onClick={handleCreateClick}>
                  {t('pms.newEpic')}
                </Button>
              )}
            </Box>
          ) : (
            <Table size="sm">
              <Thead bg="gray.50">
                <Tr>
                  <Th>{t('pms.epicTitle')}</Th>
                  <Th width="100px">{t('pms.epicStatus')}</Th>
                  <Th width="90px">{t('pms.epicPriority')}</Th>
                  <Th width="180px">{t('pms.epicProgress')}</Th>
                  <Th width="80px">{t('pms.epicSprints')}</Th>
                  <Th width="120px">{t('pms.epicTargetDate')}</Th>
                  <Th width="100px">{t('pms.epicOwner')}</Th>
                  {canEditScrum && <Th width="60px"></Th>}
                </Tr>
              </Thead>
              <Tbody>
                {filteredEpics.map(epic => (
                  <Tr key={epic.slug} _hover={{ bg: 'gray.50' }}>
                    <Td>
                      <Link href={`/projects/${projectSlug}/epics/${epic.slug}`} style={{ textDecoration: 'none' }}>
                        <Text color="blue.600" _hover={{ color: 'blue.800' }} fontWeight="medium">
                          {epic.title}
                        </Text>
                        {epic.totalStories > 0 && (
                          <Text fontSize="xs" color="gray.500">
                            {epic.doneStories}/{epic.totalStories} {t('pms.stories').toLowerCase()}
                          </Text>
                        )}
                      </Link>
                    </Td>
                    <Td>
                      <Badge
                        borderWidth="1px"
                        borderStyle="solid"
                        bg="transparent"
                        color={getStoryStatusColor(epic.status)}
                        borderColor={getStoryStatusColor(epic.status)}
                      >
                        {getStatusName(epic.status)}
                      </Badge>
                    </Td>
                    <Td>
                      <HStack spacing={1}>
                        <Box w="8px" h="8px" borderRadius="full" bg={getPriorityColor(epic.priority)} />
                        <Text fontSize="xs">{getPriorityName(epic.priority)}</Text>
                      </HStack>
                    </Td>
                    <Td>
                      <Box>
                        <Progress
                          value={epic.progressPercent || 0}
                          size="sm"
                          colorScheme="green"
                          borderRadius="full"
                          mb={1}
                        />
                        <Text fontSize="2xs" color="gray.500">
                          {epic.progressPercent || 0}% · {epic.doneStoryPoints || 0}/{epic.totalStoryPoints || 0} pt
                        </Text>
                      </Box>
                    </Td>
                    <Td>
                      <Text fontSize="sm" color="gray.600">{epic.sprintCount || 0}</Text>
                    </Td>
                    <Td>
                      <Text fontSize="sm" color="gray.600">
                        {epic.targetDate ? new Date(epic.targetDate).toLocaleDateString() : '—'}
                      </Text>
                    </Td>
                    <Td>
                      <Text fontSize="sm" color="gray.600">{epic.ownerName || '—'}</Text>
                    </Td>
                    {canEditScrum && (
                      <Td>
                        <Menu>
                          <MenuButton
                            as={IconButton}
                            icon={<FiMoreVertical />}
                            variant="ghost"
                            size="xs"
                            aria-label="actions"
                          />
                          <MenuList>
                            <MenuItem icon={<FiEdit />} onClick={() => handleEditClick(epic)}>
                              {t('common.edit')}
                            </MenuItem>
                            <MenuItem icon={<FiTrash2 />} color="red.500" onClick={() => handleDelete(epic)}>
                              {t('common.delete')}
                            </MenuItem>
                          </MenuList>
                        </Menu>
                      </Td>
                    )}
                  </Tr>
                ))}
              </Tbody>
            </Table>
          )}
        </CardBody>
      </Card>

      {/* Epic 创建/编辑 Drawer */}
      <EpicDrawer
        isOpen={isDrawerOpen}
        onClose={() => setIsDrawerOpen(false)}
        mode={drawerMode}
        projectSlug={projectSlug}
        initialData={editingEpic ? {
          title: editingEpic.title,
          description: editingEpic.description || '',
          goal: editingEpic.goal || '',
          status: editingEpic.status,
          priority: editingEpic.priority,
          startDate: editingEpic.startDate || '',
          targetDate: editingEpic.targetDate || '',
          ownerName: editingEpic.ownerName || '',
        } : undefined}
        onSave={handleSave}
        members={members}
      />

      <ConfirmDialog
        isOpen={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={handleConfirm}
        title={t('common.confirm')}
        message={t('pms.deleteEpicConfirm')}
      />
    </Stack>
  )
}
