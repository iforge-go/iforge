'use client'

import { useState, useEffect } from 'react'
import {
  Box,
  Button,
  Flex,
  Grid,
  Stack,
  Text,
  Drawer,
  DrawerOverlay,
  DrawerContent,
  DrawerCloseButton,
  DrawerHeader,
  DrawerBody,
  DrawerFooter,
  Input,
  Textarea,
  Select,
} from '@chakra-ui/react'
import { FiCpu, FiEdit, FiTarget, FiTrash2 } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { AIOptimizeResult, api } from '@/lib/api'
import AIOptimizeModal from '@/components/AIOptimizeModal'
import { getStoryStatusColor } from '@/lib/storyStatus'

export interface UserStoryFormData {
  title: string
  description: string
  status: string
  priority: string
  storyPoints: number
  acceptanceCriteria: string
  epicSlug: string | null
}

export interface UserStoryDrawerProps {
  isOpen: boolean
  onClose: () => void
  /** 'create' | 'edit' | 'view' — view 模式下字段只读、不显示 AI 优化按钮和保存按钮 */
  mode: 'create' | 'edit' | 'view'
  /** AI 优化 API 需要 projectSlug */
  projectSlug: string
  /** 编辑/查看时传入初始数据；新建时可不传 */
  initialData?: Partial<UserStoryFormData>
  /** 保存回调，由父组件负责 API 调用、刷新列表、toast 等 */
  onSave?: (data: UserStoryFormData) => Promise<void>
  /** view 模式下显示「编辑」按钮，点击后由父组件切换 mode 为 edit */
  onEdit?: () => void
  /** edit 模式下显示删除按钮（view/create 模式忽略）。点击后由父组件负责确认 + API 调用 */
  onDelete?: () => void
}

export default function UserStoryDrawer({
  isOpen,
  onClose,
  mode,
  projectSlug,
  initialData,
  onSave,
  onEdit,
  onDelete,
}: UserStoryDrawerProps) {
  const { t } = useI18n()
  const isReadOnly = mode === 'view'

  const [formData, setFormData] = useState<UserStoryFormData>({
    title: '',
    description: '',
    status: 'open',
    priority: 'medium',
    storyPoints: 0,
    acceptanceCriteria: '',
    epicSlug: null,
  })
  const [isAIOptimizeOpen, setIsAIOptimizeOpen] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  // Epic 列表：用于 Story 关联到 Epic 的下拉选择(可选)
  const [epics, setEpics] = useState<any[]>([])

  // 加载项目 Epic 列表(只在 drawer 打开时加载,降低门槛:可选字段不阻塞表单)
  useEffect(() => {
    if (!isOpen || !projectSlug) return
    api.getEpics(projectSlug).then((data) => setEpics(data || [])).catch(() => setEpics([]))
  }, [isOpen, projectSlug])

  // 打开抽屉时，用 initialData 初始化表单
  useEffect(() => {
    if (isOpen) {
      setFormData({
        title: initialData?.title || '',
        description: initialData?.description || '',
        status: initialData?.status || 'open',
        priority: initialData?.priority || 'medium',
        storyPoints: initialData?.storyPoints || 0,
        acceptanceCriteria: initialData?.acceptanceCriteria || '',
        epicSlug: initialData?.epicSlug ?? null,
      })
    }
  }, [isOpen, initialData])

  const update = <K extends keyof UserStoryFormData>(key: K, value: UserStoryFormData[K]) => {
    setFormData(prev => ({ ...prev, [key]: value }))
  }

  const handleSave = async () => {
    if (!onSave) return
    setIsSaving(true)
    try {
      await onSave(formData)
      onClose()
    } finally {
      setIsSaving(false)
    }
  }

  const handleAIOptimizeApply = (result: AIOptimizeResult) => {
    setFormData(prev => ({
      ...prev,
      title: result.title,
      description: result.description,
      acceptanceCriteria: result.acceptanceCriteria,
      priority: result.priority,
      storyPoints: result.storyPoints,
    }))
  }

  const headerTitle = mode === 'edit'
    ? t('pms.editStory')
    : mode === 'view'
      ? t('pms.viewStory')
      : t('pms.newStory')

  return (
    <>
      <Drawer isOpen={isOpen} placement="right" onClose={onClose} size="md" blockScrollOnMount={false}>
        <DrawerOverlay />
        <DrawerContent>
          <DrawerCloseButton />
          <DrawerHeader borderBottomWidth="1px">{headerTitle}</DrawerHeader>

          <DrawerBody>
            <Stack spacing={4}>
              {/* 故事标题 + AI 优化按钮 */}
              <Box>
                <Flex justify="space-between" align="center" mb={2}>
                  <Text fontSize="sm" color="gray.700">{t('pms.userStoryTitle')}</Text>
                  {!isReadOnly && (
                    <Button
                      size="xs"
                      leftIcon={<FiCpu />}
                      colorScheme="purple"
                      variant="outline"
                      onClick={() => setIsAIOptimizeOpen(true)}
                      isDisabled={!formData.title}
                    >
                      {t('pms.aiOptimize')}
                    </Button>
                  )}
                </Flex>
                <Input
                  value={formData.title}
                  onChange={(e) => update('title', e.target.value)}
                  placeholder={t('pms.userStoryTitlePlaceholder')}
                  isReadOnly={isReadOnly}
                />
              </Box>

              {/* 故事描述 */}
              <Box>
                <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.userStoryDescription')}</Text>
                <Textarea
                  value={formData.description}
                  onChange={(e) => update('description', e.target.value)}
                  placeholder={t('pms.userStoryDescriptionPlaceholder')}
                  resize="none"
                  rows={6}
                  isReadOnly={isReadOnly}
                />
              </Box>

              {/* 验收标准 */}
              <Box>
                <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.acceptanceCriteria')}</Text>
                <Textarea
                  value={formData.acceptanceCriteria}
                  onChange={(e) => update('acceptanceCriteria', e.target.value)}
                  placeholder={t('pms.acceptanceCriteriaPlaceholder')}
                  resize="none"
                  rows={5}
                  isReadOnly={isReadOnly}
                />
              </Box>

              {/* 状态 / 优先级（第一行两列）；故事点（第二行左侧） */}
              <Grid templateColumns="1fr 1fr" gap={3}>
                <Box>
                  <Text fontSize="sm" color="gray.700" mb={2}>{t('common.status')}</Text>
                  <Select
                    value={formData.status}
                    onChange={(e) => update('status', e.target.value)}
                    isDisabled={isReadOnly}
                    borderColor={getStoryStatusColor(formData.status)}
                    color={getStoryStatusColor(formData.status)}
                    sx={{ option: { color: 'gray.700' } }}
                  >
                    <option value="open">{t('pms.statusOpen')}</option>
                    <option value="in_progress">{t('pms.statusInProgress')}</option>
                    <option value="done">{t('pms.statusDone')}</option>
                  </Select>
                </Box>
                <Box>
                  <Text fontSize="sm" color="gray.700" mb={2}>{t('common.priority')}</Text>
                  <Select
                    value={formData.priority}
                    onChange={(e) => update('priority', e.target.value)}
                    isDisabled={isReadOnly}
                  >
                    <option value="urgent">{t('common.urgent')}</option>
                    <option value="high">{t('common.high')}</option>
                    <option value="medium">{t('common.medium')}</option>
                    <option value="low">{t('common.low')}</option>
                  </Select>
                </Box>
                <Box>
                  <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.storyPoints')}</Text>
                  <Input
                    type="number"
                    value={formData.storyPoints}
                    onChange={(e) => update('storyPoints', parseInt(e.target.value) || 0)}
                    min={0}
                    isReadOnly={isReadOnly}
                  />
                </Box>
              </Grid>

              {/* Epic 关联(可选):放最下方降低门槛。空选项="(无 Epic)"对应 epicSlug=null */}
              <Box>
                <Text fontSize="sm" color="gray.700" mb={2}>
                  <Text as="span" display="inline-flex" alignItems="center" mr={1}>
                    <FiTarget size={12} style={{ display: 'inline', marginRight: 4 }} />
                    {t('pms.epic')}
                  </Text>
                  <Text as="span" color="gray.400" fontSize="xs">({t('common.optional', { defaultValue: '可选' })})</Text>
                </Text>
                <Select
                  value={formData.epicSlug || ''}
                  onChange={(e) => update('epicSlug', e.target.value || null)}
                  isDisabled={isReadOnly}
                  placeholder={t('pms.noEpic')}
                >
                  {epics.map((epic: any) => (
                    <option key={epic.slug} value={epic.slug}>{epic.title}</option>
                  ))}
                </Select>
              </Box>
            </Stack>
          </DrawerBody>

          <DrawerFooter borderTopWidth="1px">
            {/* edit 模式下显示删除按钮(左侧,与 Cancel/Save 分离) */}
            {mode === 'edit' && onDelete && (
              <Button
                variant="ghost"
                colorScheme="red"
                leftIcon={<FiTrash2 />}
                onClick={onDelete}
                mr="auto"
              >
                {t('common.delete')}
              </Button>
            )}
            <Button variant="ghost" onClick={onClose} mr={3}>
              {isReadOnly ? t('common.close') : t('common.cancel')}
            </Button>
            {isReadOnly ? (
              onEdit && (
                <Button colorScheme="blue" leftIcon={<FiEdit />} onClick={onEdit}>
                  {t('common.edit')}
                </Button>
              )
            ) : (
              <Button colorScheme="blue" onClick={handleSave} isLoading={isSaving}>
                {mode === 'edit' ? t('common.save') : t('common.create')}
              </Button>
            )}
          </DrawerFooter>
        </DrawerContent>
      </Drawer>

      {/* AI 优化 Modal */}
      <AIOptimizeModal
        isOpen={isAIOptimizeOpen}
        onClose={() => setIsAIOptimizeOpen(false)}
        projectSlug={projectSlug}
        draft={{
          title: formData.title,
          description: formData.description,
          acceptanceCriteria: formData.acceptanceCriteria,
        }}
        onApply={handleAIOptimizeApply}
      />
    </>
  )
}
