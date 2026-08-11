'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams } from 'next/navigation'
import NextLink from 'next/link'
import {
  Box,
  Text,
  Spinner,
  Flex,
  Stack,
  Heading,
  Button,
  Input,
  Select,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Checkbox,
  Switch,
  IconButton,
  HStack,
  useToast,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
} from '@chakra-ui/react'
import { FiArrowLeft, FiPlus, FiTrash2, FiAlignJustify } from 'react-icons/fi'
import {
  DndContext,
  closestCorners,
  PointerSensor,
  useSensor,
  useSensors,
  type DragStartEvent,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import { api } from '@/lib/api'
import { useProject } from '../ProjectContext'
import type { TaskStatus, TaskStatusTransition } from '@/lib/types'

// 状态三态分类(对齐 Jira status category):驱动燃尽图/速度图口径与看板列分组
const CATEGORIES = [
  { value: 'todo', labelKey: 'statusConfig.categoryTodo' },
  { value: 'in_progress', labelKey: 'statusConfig.categoryInProgress' },
  { value: 'done', labelKey: 'statusConfig.categoryDone' },
] as const

export default function StatusesPage() {
  const params = useParams()
  const projectSlug = params.slug as string
  const { t } = useI18n()
  const toast = useToast()
  const { user, authLoading } = useCurrentUser()
  const { project, loading: projectLoading, canManageProject } = useProject()

  const [statuses, setStatuses] = useState<TaskStatus[]>([])
  const [loadingData, setLoadingData] = useState(true)
  // 流转矩阵本地勾选态:key = `${fromSlug}→${toSlug}`,value=true 表示允许
  const [matrix, setMatrix] = useState<Record<string, boolean>>({})
  const [matrixDirty, setMatrixDirty] = useState(false)
  const [savingMatrix, setSavingMatrix] = useState(false)

  // 新增状态表单
  const [newName, setNewName] = useState('')
  const [newSlug, setNewSlug] = useState('')
  const [newColor, setNewColor] = useState('#94a3b8')
  const [newCategory, setNewCategory] = useState('todo')
  const [newIsClosed, setNewIsClosed] = useState(false)
  const [adding, setAdding] = useState(false)

  // 删除确认
  const [deleteTarget, setDeleteTarget] = useState<TaskStatus | null>(null)
  const [deleting, setDeleting] = useState(false)
  const cancelRef = useRef(null as any)

  const loadData = useCallback(async () => {
    setLoadingData(true)
    try {
      const [st, tr] = await Promise.all([
        api.getTaskStatuses(projectSlug),
        api.getTransitions(projectSlug),
      ])
      setStatuses(st || [])
      const m: Record<string, boolean> = {}
      ;(tr || []).forEach((x: TaskStatusTransition) => {
        m[`${x.fromSlug}→${x.toSlug}`] = true
      })
      setMatrix(m)
      setMatrixDirty(false)
    } catch {
      // 加载失败静默(权限不足等场景由 layout 拦截)
    } finally {
      setLoadingData(false)
    }
  }, [projectSlug])

  useEffect(() => {
    if (project) loadData()
  }, [project, loadData])

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    })
  )

  if (authLoading) return null
  if (!user) {
    return (
      <Box py={20}>
        <Text textAlign="center">{t('common.pleaseLogin')}</Text>
      </Box>
    )
  }
  if (projectLoading || loadingData) {
    return (
      <Flex justify="center" align="center" minH="60vh">
        <Spinner size="xl" color="blue.500" />
      </Flex>
    )
  }
  if (!project) return null

  const canEdit = canManageProject
  const sorted = [...statuses].sort((a, b) => a.position - b.position)

  const handleDragStart = (_event: DragStartEvent) => {}

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event
    if (!over || active.id === over.id) return
    if (!canEdit) return

    const oldIndex = sorted.findIndex((s) => s.id === active.id)
    const newIndex = sorted.findIndex((s) => s.id === over.id)
    if (oldIndex === -1 || newIndex === -1) return

    const reordered = arrayMove(sorted, oldIndex, newIndex)
    reordered.forEach((s, i) => {
      if (s.position !== i) {
        patchStatus(s.id, { position: i })
      }
    })
  }

  // inline 更新单个状态字段(每个字段独立保存,避免互相干扰)
  async function patchStatus(id: number, patch: Partial<TaskStatus>) {
    if (!canEdit) return
    try {
      await api.updateTaskStatus(id, patch as any)
      setStatuses((prev) =>
        prev.map((s) => (s.id === id ? ({ ...s, ...patch } as TaskStatus) : s))
      )
    } catch {
      toast({ title: t('statusConfig.saveFailed'), status: 'error', duration: 3000 })
    }
  }

  async function confirmDelete() {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await api.deleteTaskStatus(deleteTarget.id)
      setDeleteTarget(null)
      await loadData()
    } catch (e: any) {
      toast({ title: t('statusConfig.saveFailed'), description: e.message, status: 'error', duration: 3000 })
    } finally {
      setDeleting(false)
    }
  }

  async function handleAdd() {
    if (!newName.trim()) {
      toast({ title: t('statusConfig.nameRequired'), status: 'warning', duration: 3000 })
      return
    }
    if (!newSlug.trim()) {
      toast({ title: t('statusConfig.slugRequired'), status: 'warning', duration: 3000 })
      return
    }
    setAdding(true)
    try {
      await api.createTaskStatus(projectSlug, {
        name: newName.trim(),
        slug: newSlug.trim(),
        color: newColor,
        category: newCategory,
        isClosed: newIsClosed,
        wipLimit: 0,
      })
      setNewName('')
      setNewSlug('')
      setNewColor('#94a3b8')
      setNewCategory('todo')
      setNewIsClosed(false)
      await loadData()
    } catch (e: any) {
      toast({ title: t('statusConfig.saveFailed'), description: e.message, status: 'error', duration: 3000 })
    } finally {
      setAdding(false)
    }
  }

  function toggleMatrix(from: string, to: string) {
    if (!canEdit) return
    const key = `${from}→${to}`
    setMatrix((prev) => {
      const next = { ...prev }
      if (next[key]) delete next[key]
      else next[key] = true
      return next
    })
    setMatrixDirty(true)
  }

  async function handleSaveMatrix() {
    setSavingMatrix(true)
    try {
      const pairs = Object.entries(matrix)
        .filter(([, v]) => v)
        .map(([k]) => {
          const [from, to] = k.split('→')
          return { from, to }
        })
      await api.setTransitions(projectSlug, pairs)
      setMatrixDirty(false)
      toast({ title: t('statusConfig.saved'), status: 'success', duration: 2000 })
    } catch (e: any) {
      toast({ title: t('statusConfig.saveFailed'), description: e.message, status: 'error', duration: 3000 })
    } finally {
      setSavingMatrix(false)
    }
  }

  return (
    <Stack spacing={6}>
      {/* 标题区 */}
      <Flex justify="space-between" align="center" wrap="wrap" gap={3}>
        <Box>
          <Heading size="md">{t('statusConfig.title')}</Heading>
          <Text fontSize="sm" color="gray.500" mt={1}>
            {t('statusConfig.description')}
          </Text>
        </Box>
        <Button
          as={NextLink}
          href={`/projects/${projectSlug}/tasks?view=board`}
          variant="outline"
          size="sm"
          leftIcon={<FiArrowLeft />}
        >
          {t('statusConfig.backToBoard')}
        </Button>
      </Flex>

      {!canEdit && (
        <Text fontSize="sm" color="gray.500">
          {t('statusConfig.readOnly')}
        </Text>
      )}

      {/* 状态列表(inline 编辑 category/WIP/isClosed) */}
      <Box bg="white" border="1px solid" borderColor="myGray.200" borderRadius="lg" p={5}>
        <Heading size="sm" mb={4}>
          {t('statusConfig.statuses')}
        </Heading>
        <Box overflowX="auto">
          <DndContext
            sensors={sensors}
            collisionDetection={closestCorners}
            onDragStart={handleDragStart}
            onDragEnd={handleDragEnd}
          >
            <SortableContext items={sorted.map((s) => s.id)} strategy={verticalListSortingStrategy}>
              <Table size="sm">
                <Thead>
                  <Tr>
                    <Th w="40px"></Th>
                    <Th>{t('statusConfig.statusColor')}</Th>
                    <Th>{t('statusConfig.statusName')}</Th>
                    <Th>{t('statusConfig.category')}</Th>
                    <Th>{t('statusConfig.wipLimit')}</Th>
                    <Th>{t('statusConfig.isClosed')}</Th>
                    <Th></Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {sorted.map((s) => (
                    <SortableStatusRow
                      key={s.id}
                      status={s}
                      canEdit={canEdit}
                      t={t}
                      patchStatus={patchStatus}
                      onDelete={() => setDeleteTarget(s)}
                      CATEGORIES={CATEGORIES}
                    />
                  ))}
                </Tbody>
              </Table>
            </SortableContext>
          </DndContext>
        </Box>

        {/* 新增状态表单 */}
        {canEdit && (
          <HStack spacing={2} mt={4} wrap="wrap">
            <Input
              placeholder={t('statusConfig.statusName')}
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              size="sm"
              w="140px"
            />
            <Input
              placeholder={t('statusConfig.statusSlug')}
              value={newSlug}
              onChange={(e) => setNewSlug(e.target.value)}
              size="sm"
              w="140px"
            />
            <Input
              type="color"
              value={newColor}
              onChange={(e) => setNewColor(e.target.value)}
              w="42px"
              h="32px"
              p="0"
              border="1px solid #e2e8f0"
              borderRadius="md"
            />
            <Select
              value={newCategory}
              onChange={(e) => setNewCategory(e.target.value)}
              size="sm"
              w="120px"
            >
              {CATEGORIES.map((c) => (
                <option key={c.value} value={c.value}>
                  {t(c.labelKey)}
                </option>
              ))}
            </Select>
            <HStack spacing={1}>
              <Switch
                isChecked={newIsClosed}
                onChange={(e) => setNewIsClosed(e.target.checked)}
                size="sm"
              />
              <Text fontSize="xs" color="gray.500">
                {t('statusConfig.isClosed')}
              </Text>
            </HStack>
            <Button
              leftIcon={<FiPlus />}
              size="sm"
              colorScheme="primary"
              onClick={handleAdd}
              isLoading={adding}
            >
              {t('statusConfig.addStatus')}
            </Button>
          </HStack>
        )}
      </Box>

      {/* 工作流流转规则矩阵 */}
      <Box bg="white" border="1px solid" borderColor="myGray.200" borderRadius="lg" p={5}>
        <Flex justify="space-between" align="center" mb={1}>
          <Heading size="sm">{t('statusConfig.workflowRules')}</Heading>
          {canEdit && (
            <Button
              size="sm"
              colorScheme="primary"
              onClick={handleSaveMatrix}
              isLoading={savingMatrix}
              isDisabled={!matrixDirty}
            >
              {t('statusConfig.save')}
            </Button>
          )}
        </Flex>
        <Text fontSize="sm" color="gray.500" mb={4}>
          {t('statusConfig.workflowRulesDesc')}
        </Text>
        {sorted.length === 0 ? (
          <Text fontSize="sm" color="gray.400">
            {t('statusConfig.noTransitions')}
          </Text>
        ) : (
          <Box overflowX="auto">
            <Table size="sm">
              <Thead>
                <Tr>
                  <Th position="sticky" left={0} bg="white" zIndex={1}>
                    {t('statusConfig.fromTo')}
                  </Th>
                  {sorted.map((s) => (
                    <Th key={s.slug} minW="110px">
                      <HStack spacing={1}>
                        <Box w="8px" h="8px" borderRadius="full" bg={s.color} />
                        <Text fontSize="xs">{s.name}</Text>
                      </HStack>
                    </Th>
                  ))}
                </Tr>
              </Thead>
              <Tbody>
                {sorted.map((from) => (
                  <Tr key={from.slug}>
                    <Td position="sticky" left={0} bg="white" zIndex={1} fontWeight="600">
                      <HStack spacing={1}>
                        <Box w="8px" h="8px" borderRadius="full" bg={from.color} />
                        <Text fontSize="xs">{from.name}</Text>
                      </HStack>
                    </Td>
                    {sorted.map((to) => (
                      <Td key={to.slug}>
                        {from.slug === to.slug ? (
                          <Text color="gray.300" textAlign="center">
                            —
                          </Text>
                        ) : (
                          <Checkbox
                            isChecked={!!matrix[`${from.slug}→${to.slug}`]}
                            isDisabled={!canEdit}
                            onChange={() => toggleMatrix(from.slug, to.slug)}
                          />
                        )}
                      </Td>
                    ))}
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </Box>
        )}
      </Box>

      {/* 删除确认(统一 modal,符合用户偏好:不用 browser confirm) */}
      <AlertDialog
        isOpen={!!deleteTarget}
        leastDestructiveRef={cancelRef}
        onClose={() => setDeleteTarget(null)}
      >
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg">
              {t('statusConfig.deleteStatus')}
            </AlertDialogHeader>
            <AlertDialogBody>{t('statusConfig.confirmDelete')}</AlertDialogBody>
            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={() => setDeleteTarget(null)}>
                {t('common.cancel')}
              </Button>
              <Button colorScheme="red" onClick={confirmDelete} ml={3} isLoading={deleting}>
                {t('statusConfig.deleteStatus')}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>
    </Stack>
  )
}

function SortableStatusRow({
  status,
  canEdit,
  t,
  patchStatus,
  onDelete,
  CATEGORIES,
}: {
  status: TaskStatus
  canEdit: boolean
  t: (key: string) => string
  patchStatus: (id: number, patch: Partial<TaskStatus>) => void
  onDelete: () => void
  CATEGORIES: readonly { value: string; labelKey: string }[]
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: status.id,
    data: { type: 'status', status },
    disabled: !canEdit,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <Tr
      ref={setNodeRef}
      style={style}
      cursor={canEdit ? 'grab' : 'default'}
      _active={{ cursor: 'grabbing' }}
    >
      <Td>
        <Box
          {...attributes}
          {...listeners}
          display="flex"
          alignItems="center"
          justifyContent="center"
          h="full"
          color={canEdit ? 'gray.400' : 'transparent'}
          _hover={{ color: 'gray.600' }}
        >
          <FiAlignJustify size={18} />
        </Box>
      </Td>
      <Td>
        <Input
          type="color"
          defaultValue={status.color}
          w="42px"
          h="28px"
          p="0"
          border="1px solid #e2e8f0"
          borderRadius="md"
          isDisabled={!canEdit}
          onBlur={(e) => {
            if (e.target.value !== status.color) patchStatus(status.id, { color: e.target.value })
          }}
        />
      </Td>
      <Td>
        <Input
          defaultValue={status.name}
          size="sm"
          w="140px"
          isDisabled={!canEdit}
          onBlur={(e) => {
            if (e.target.value !== status.name) patchStatus(status.id, { name: e.target.value })
          }}
        />
      </Td>
      <Td>
        <Select
          size="sm"
          w="120px"
          value={status.category}
          isDisabled={!canEdit}
          onChange={(e) => patchStatus(status.id, { category: e.target.value })}
        >
          {CATEGORIES.map((c) => (
            <option key={c.value} value={c.value}>
              {t(c.labelKey)}
            </option>
          ))}
        </Select>
      </Td>
      <Td>
        <Input
          type="number"
          min={0}
          defaultValue={status.wipLimit ?? ''}
          size="sm"
          w="80px"
          isDisabled={!canEdit}
          placeholder={t('statusConfig.wipLimitHint')}
          onBlur={(e) => {
            const v = e.target.value.trim()
            const num = v === '' ? 0 : parseInt(v, 10)
            const newLimit = isNaN(num) || num <= 0 ? 0 : num
            const oldLimit = status.wipLimit ?? 0
            if (newLimit !== oldLimit) patchStatus(status.id, { wipLimit: newLimit })
          }}
        />
      </Td>
      <Td>
        <Switch
          isChecked={status.isClosed}
          isDisabled={!canEdit}
          onChange={(e) => patchStatus(status.id, { isClosed: e.target.checked })}
        />
      </Td>
      <Td>
        <IconButton
          aria-label={t('statusConfig.deleteStatus')}
          icon={<FiTrash2 />}
          size="sm"
          variant="ghost"
          color="red.400"
          isDisabled={!canEdit}
          onClick={onDelete}
        />
      </Td>
    </Tr>
  )
}
