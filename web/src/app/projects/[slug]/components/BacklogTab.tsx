'use client'

import { useState, useMemo, useRef } from 'react'
import {
  Box,
  Flex,
  HStack,
  VStack,
  Text,
  Input,
  InputGroup,
  InputLeftElement,
  Button,
  Grid,
  useToast,
  AlertDialog,
  AlertDialogBody,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
  AlertDialogFooter,
  Badge,
} from '@chakra-ui/react'
import { FiPlus, FiSearch, FiRepeat } from 'react-icons/fi'
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useSensor,
  useSensors,
  closestCorners,
  type DragStartEvent,
  type DragEndEvent,
} from '@dnd-kit/core'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useProject } from '../ProjectContext'
import UserStoryDrawer, { type UserStoryFormData } from '@/components/UserStoryDrawer'
import SprintDrawer, { type SprintFormData } from '@/components/SprintDrawer'
import EpicDrawer, { type EpicFormData } from '../epics/components/EpicDrawer'
import EpicSidebar from './backlog/EpicSidebar'
import SprintColumn from './backlog/SprintColumn'
import BacklogColumn from './backlog/BacklogColumn'

// 优先级色点颜色(与 BacklogStoryRow 一致)
const PRIORITY_DOT_COLORS: Record<string, string> = {
  urgent: '#e53e3e',
  high: '#d69e2e',
  medium: '#3182ce',
  low: '#718096',
}

interface BacklogTabProps {
  projectSlug: string
  allStories: any[]
  sprints: any[]
  epics: any[]
  onAllStoriesChange: (stories: any[]) => void
  onSprintsChange?: (sprints: any[]) => void
  onEpicsChange?: (epics: any[]) => void
}

// Jira 风格 Backlog 容器
// 布局:左侧 Epic 面板 + 右侧 Sprint 区/Backlog 区 + 拖拽规划
// 职责单一:只做规划(拖拽 Story 进/出 Sprint),编辑去右侧 UserStoryDrawer
export default function BacklogTab({
  projectSlug,
  allStories,
  sprints,
  epics,
  onAllStoriesChange,
  onSprintsChange,
  onEpicsChange,
}: BacklogTabProps) {
  const { t } = useI18n()
  const toast = useToast()
  const { canEditScrum } = useProject()

  // --- 状态(精简为 8 个,从原 26 个大幅削减) ---
  const [searchKeyword, setSearchKeyword] = useState('')
  const [selectedEpicFilter, setSelectedEpicFilter] = useState('all')
  const [epicPanelCollapsed, setEpicPanelCollapsed] = useState(false)
  const [activeDragStorySlug, setActiveDragStorySlug] = useState<string | null>(null)
  const [highlightedStorySlug, setHighlightedStorySlug] = useState<string | null>(null)

  // UserStoryDrawer 状态
  const [storyDrawerOpen, setStoryDrawerOpen] = useState(false)
  const [storyDrawerMode, setStoryDrawerMode] = useState<'create' | 'edit' | 'view'>('create')
  const [editingStory, setEditingStory] = useState<any>(null)

  // SprintDrawer 状态（新建/编辑 Sprint，对齐 Jira Backlog 中的 Create/Edit Sprint）
  const [sprintDrawerOpen, setSprintDrawerOpen] = useState(false)
  const [sprintDrawerMode, setSprintDrawerMode] = useState<'create' | 'edit'>('create')
  const [editingSprint, setEditingSprint] = useState<any>(null)

  // EpicDrawer 状态（从 Epic 侧边栏直接新建 Epic）
  const [epicDrawerOpen, setEpicDrawerOpen] = useState(false)

  // 删除确认
  const [storyToDelete, setStoryToDelete] = useState<string | null>(null)
  const cancelRef = useRef(null as any)

  // --- 数据派生(useMemo,单一数据源 allStories) ---
  // 1. 按 Epic 过滤
  const epicFilteredStories = useMemo(() => {
    if (selectedEpicFilter === 'all') return allStories
    if (selectedEpicFilter === '__unassigned__') return allStories.filter(s => !s.epicSlug)
    return allStories.filter(s => s.epicSlug === selectedEpicFilter)
  }, [allStories, selectedEpicFilter])

  // 2. 按搜索关键词过滤
  const searchFiltered = useMemo(() => {
    const q = searchKeyword.toLowerCase().trim()
    if (!q) return epicFilteredStories
    return epicFilteredStories.filter(s => s.title?.toLowerCase().includes(q))
  }, [epicFilteredStories, searchKeyword])

  // 3. 按 Sprint 归属分组
  const { storiesBySprint, backlogStories } = useMemo(() => {
    const map = new Map<string, any[]>()
    const backlog: any[] = []
    for (const s of searchFiltered) {
      if (s.sprintSlug) {
        if (!map.has(s.sprintSlug)) map.set(s.sprintSlug, [])
        map.get(s.sprintSlug)!.push(s)
      } else {
        backlog.push(s)
      }
    }
    return { storiesBySprint: map, backlogStories: backlog }
  }, [searchFiltered])

  // 4. 可见 Sprint
  // 统一逻辑（不分全部/Epic 过滤）：
  //   - open/active Sprint 始终显示（含空 Sprint，供从待办事项拖拽故事进 Sprint）
  //   - closed Sprint 仅在有匹配故事时显示（否则已关闭 Sprint 中的故事会消失）
  const visibleSprints = useMemo(() => {
    return sprints
      .filter(s => s.status === 'open' || s.status === 'active' || storiesBySprint.has(s.slug))
      .sort((a, b) => {
        const rank = (s: any) => (s.status === 'active' ? 0 : s.status === 'open' ? 1 : 2)
        return rank(a) - rank(b)
      })
  }, [sprints, storiesBySprint])

  // 5. Epic 面板统计
  const unassignedEpicCount = useMemo(
    () => allStories.filter(s => !s.epicSlug).length,
    [allStories]
  )

  // --- DnD 配置(复用 KanbanBoard 模式) ---
  const pointerSensor = useSensor(PointerSensor, { activationConstraint: { distance: 8 } })
  const sensors = useSensors(...(canEditScrum ? [pointerSensor] : []))

  const activeDragStory = activeDragStorySlug
    ? allStories.find(s => s.slug === activeDragStorySlug) || null
    : null

  function handleDragStart(event: DragStartEvent) {
    setActiveDragStorySlug(event.active.id as string)
  }

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    setActiveDragStorySlug(null)
    if (!over || !canEditScrum) return

    const storySlug = active.id as string
    const story = allStories.find(s => s.slug === storySlug)
    if (!story) return

    const overData = over.data.current as { type: 'sprint' | 'backlog'; sprintSlug: string | null } | undefined
    const targetSprintSlug = overData?.type === 'sprint' ? overData.sprintSlug : null

    // 同区域拖拽 = 无操作(v1 不支持区域内排序)
    if (story.sprintSlug === targetSprintSlug) return

    // 乐观更新:先移动本地数据
    const prevStories = allStories
    onAllStoriesChange(
      allStories.map(s => s.slug === storySlug ? { ...s, sprintSlug: targetSprintSlug } : s)
    )

    // 持久化
    api.updateUserStory(storySlug, { sprintSlug: targetSprintSlug })
      .then(() => {
        toast({
          title: targetSprintSlug ? t('pms.storyMovedToSprint') : t('pms.storyMovedToBacklog'),
          status: 'success',
          duration: 2000,
        })
      })
      .catch((error: any) => {
        onAllStoriesChange(prevStories)  // 回滚
        toast({
          title: t('common.error'),
          description: error.message,
          status: 'error',
          duration: 3000,
        })
      })
  }

  // --- Story 交互 ---
  function handleStoryClick(story: any) {
    setEditingStory(story)
    setStoryDrawerMode('view')
    setStoryDrawerOpen(true)
  }

  function handleCreateStory() {
    setEditingStory(null)
    setStoryDrawerMode('create')
    setStoryDrawerOpen(true)
  }

  // Sprint 保存回调（新建/编辑通用，根据 sprintDrawerMode 分支）
  async function handleSaveSprint(data: SprintFormData) {
    try {
      if (sprintDrawerMode === 'edit' && editingSprint) {
        await api.updateSprint(editingSprint.slug, {
          title: data.title,
          description: data.description || undefined,
          goal: data.goal || undefined,
          status: data.status,
          startDate: data.startDate || undefined,
          endDate: data.endDate || undefined,
        })
        const fresh = await api.getSprints(projectSlug)
        onSprintsChange?.(fresh || [])
        setSprintDrawerOpen(false)
        toast({ title: t('pms.sprintUpdated'), status: 'success', duration: 2000 })
      } else {
        await api.createSprint(projectSlug, {
          title: data.title,
          description: data.description || undefined,
          goal: data.goal || undefined,
          status: data.status,
          startDate: data.startDate || undefined,
          endDate: data.endDate || undefined,
        })
        const fresh = await api.getSprints(projectSlug)
        onSprintsChange?.(fresh || [])
        setSprintDrawerOpen(false)
        toast({ title: t('pms.sprintCreated'), status: 'success', duration: 2000 })
      }
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    }
  }

  // 编辑 Sprint：打开 SprintDrawer（edit 模式，填充已有数据）
  function handleEditSprint(sprint: any) {
    setEditingSprint(sprint)
    setSprintDrawerMode('edit')
    setSprintDrawerOpen(true)
  }

  // 开始 Sprint：将状态改为 active（对齐 Jira Backlog 的 Start Sprint）
  async function handleStartSprint(sprint: any) {
    try {
      await api.updateSprint(sprint.slug, { status: 'active' })
      const fresh = await api.getSprints(projectSlug)
      onSprintsChange?.(fresh || [])
      toast({ title: t('pms.sprintStarted'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    }
  }

  // 新建 Epic（从 Epic 侧边栏直接创建）
  async function handleSaveEpic(data: EpicFormData) {
    try {
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
      onEpicsChange?.([created, ...epics])
      setEpicDrawerOpen(false)
      toast({ title: t('pms.epicCreated'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    }
  }

  async function handleSaveStory(data: UserStoryFormData) {
    try {
      if (storyDrawerMode === 'create') {
        const created = await api.createUserStory(projectSlug, {
          title: data.title,
          description: data.description,
          status: data.status,
          priority: data.priority,
          storyPoints: data.storyPoints,
          acceptanceCriteria: data.acceptanceCriteria,
          epicSlug: data.epicSlug || undefined,
        })
        // 刷新列表 + 高亮新建的 Story
        const fresh = await api.getUserStories(projectSlug)
        onAllStoriesChange(fresh || [])
        setHighlightedStorySlug(created.slug)
        setTimeout(() => setHighlightedStorySlug(null), 3000)
        toast({ title: t('pms.userStoryCreated'), status: 'success', duration: 2000 })
      } else if (storyDrawerMode === 'edit' && editingStory) {
        await api.updateUserStory(editingStory.slug, {
          title: data.title,
          description: data.description,
          status: data.status,
          priority: data.priority,
          storyPoints: data.storyPoints,
          acceptanceCriteria: data.acceptanceCriteria,
          epicSlug: data.epicSlug || null,
        })
        const fresh = await api.getUserStories(projectSlug)
        onAllStoriesChange(fresh || [])
        setHighlightedStorySlug(editingStory.slug)
        setTimeout(() => setHighlightedStorySlug(null), 3000)
        toast({ title: t('pms.userStoryUpdated'), status: 'success', duration: 2000 })
      }
      setStoryDrawerOpen(false)
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    }
  }

  function handleDeleteStory() {
    if (editingStory) {
      setStoryToDelete(editingStory.slug)
      setStoryDrawerOpen(false)
    }
  }

  async function confirmDelete() {
    if (!storyToDelete) return
    try {
      await api.deleteUserStory(storyToDelete)
      const fresh = await api.getUserStories(projectSlug)
      onAllStoriesChange(fresh || [])
      setStoryToDelete(null)
      toast({ title: t('common.success'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    }
  }

  return (
    <Box>
      {/* 顶部工具栏:搜索框 + 新建按钮 + 折叠面板按钮 */}
      <Flex justify="space-between" align="center" mb={4}>
        <HStack spacing={3}>
          <InputGroup w="240px">
            <InputLeftElement pointerEvents="none">
              <FiSearch color="gray.400" size={14} />
            </InputLeftElement>
            <Input
              placeholder={t('pms.searchBacklog')}
              value={searchKeyword}
              onChange={(e) => setSearchKeyword(e.target.value)}
              size="sm"
              borderRadius="md"
            />
          </InputGroup>
        </HStack>
        <HStack spacing={2}>
          {canEditScrum && (
            <Button
              size="sm"
              colorScheme="blue"
              leftIcon={<FiPlus />}
              onClick={handleCreateStory}
            >
              {t('pms.newUserStory')}
            </Button>
          )}
        </HStack>
      </Flex>

      {/* 主体:左侧 Epic 面板 + 右侧 Sprint/Backlog 区 */}
      <Grid templateColumns={epicPanelCollapsed ? '40px 1fr' : '220px 1fr'} gap={4} alignItems="start">
        <EpicSidebar
          epics={epics}
          selectedEpicFilter={selectedEpicFilter}
          onSelect={setSelectedEpicFilter}
          totalStoriesCount={allStories.length}
          unassignedCount={unassignedEpicCount}
          projectSlug={projectSlug}
          collapsed={epicPanelCollapsed}
          onCollapse={() => setEpicPanelCollapsed(true)}
          onExpand={() => setEpicPanelCollapsed(false)}
          onCreateEpic={() => setEpicDrawerOpen(true)}
          canEditScrum={canEditScrum}
        />

        {/* 右侧:DndContext 包裹的 Sprint/Backlog 区 */}
        <DndContext
          sensors={sensors}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          collisionDetection={closestCorners}
        >
          <VStack spacing={4} align="stretch">
            {/* 上方活跃 Sprint 区 + 新建 Sprint 按钮 */}
            <Flex justify="space-between" align="center">
              <HStack spacing={2}>
                <FiRepeat color="#3182ce" size={16} />
                <Text fontSize="sm" fontWeight="medium" color="gray.700">{t('pms.sprints')}</Text>
              </HStack>
              {canEditScrum && (
                <Button
                  size="xs"
                  variant="outline"
                  leftIcon={<FiPlus />}
                  onClick={() => {
                    setEditingSprint(null)
                    setSprintDrawerMode('create')
                    setSprintDrawerOpen(true)
                  }}
                >
                  {t('pms.newSprint')}
                </Button>
              )}
            </Flex>
            <SprintColumn
              sprints={visibleSprints}
              storiesBySprint={storiesBySprint}
              projectSlug={projectSlug}
              onStoryClick={handleStoryClick}
              canEditScrum={canEditScrum}
              highlightedStorySlug={highlightedStorySlug}
              onStartSprint={handleStartSprint}
              onEditSprint={handleEditSprint}
            />

            {/* 下方需求池区 */}
            <BacklogColumn
              stories={backlogStories}
              onStoryClick={handleStoryClick}
              canEditScrum={canEditScrum}
              highlightedStorySlug={highlightedStorySlug}
            />
          </VStack>

          {/* DragOverlay:拖拽中的 Story 预览 */}
          <DragOverlay dropAnimation={{ duration: 200, easing: 'ease' }}>
            {activeDragStory ? (
              <Box
                bg="white"
                borderRadius="md"
                borderWidth="1px"
                borderColor="#3b82f6"
                boxShadow="0 8px 25px rgba(0,0,0,0.15)"
                px={3}
                py={2}
                w="400px"
              >
                <Flex align="center">
                  <Box
                    w="8px"
                    h="8px"
                    borderRadius="full"
                    bg={PRIORITY_DOT_COLORS[activeDragStory.priority || 'medium']}
                    mr={2}
                    flexShrink={0}
                  />
                  <Text flex={1} fontSize="sm" fontWeight="medium" noOfLines={1}>
                    {activeDragStory.title}
                  </Text>
                  {activeDragStory.storyPoints > 0 && (
                    <Badge colorScheme="purple" variant="subtle" fontSize="xs" ml={2}>
                      {activeDragStory.storyPoints} pt
                    </Badge>
                  )}
                </Flex>
              </Box>
            ) : null}
          </DragOverlay>
        </DndContext>
      </Grid>

      {/* UserStoryDrawer:查看/编辑/新建 */}
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
          epicSlug: editingStory.epicSlug || null,
        } : (storyDrawerMode === 'create' && selectedEpicFilter !== 'all' && selectedEpicFilter !== '__unassigned__' ? {
          epicSlug: selectedEpicFilter,
        } : undefined)}
        onSave={handleSaveStory}
        onEdit={canEditScrum ? () => setStoryDrawerMode('edit') : undefined}
        onDelete={canEditScrum && storyDrawerMode === 'edit' ? handleDeleteStory : undefined}
      />

      {/* 删除确认对话框 */}
      <AlertDialog
        isOpen={storyToDelete !== null}
        leastDestructiveRef={cancelRef}
        onClose={() => setStoryToDelete(null)}
      >
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              {t('common.delete')}
            </AlertDialogHeader>
            <AlertDialogBody>
              {t('pms.deleteUserStoryConfirm')}
            </AlertDialogBody>
            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={() => setStoryToDelete(null)}>
                {t('common.cancel')}
              </Button>
              <Button colorScheme="red" onClick={confirmDelete} ml={3}>
                {t('common.delete')}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>

      {/* SprintDrawer:新建/编辑 Sprint */}
      <SprintDrawer
        isOpen={sprintDrawerOpen}
        onClose={() => setSprintDrawerOpen(false)}
        mode={sprintDrawerMode}
        projectSlug={projectSlug}
        initialData={editingSprint || undefined}
        onSave={handleSaveSprint}
      />

      {/* EpicDrawer:从侧边栏直接新建 Epic */}
      <EpicDrawer
        isOpen={epicDrawerOpen}
        onClose={() => setEpicDrawerOpen(false)}
        mode="create"
        projectSlug={projectSlug}
        onSave={handleSaveEpic}
      />
    </Box>
  )
}
