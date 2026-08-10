'use client'

import { useState, useEffect } from 'react'
import {
  Box,
  Button,
  Flex,
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
  HStack,
} from '@chakra-ui/react'
import { FiCpu, FiEdit } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import AIOptimizeModal from '@/components/AIOptimizeModal'
import { AIOptimizeSprintResult } from '@/lib/api'

export interface SprintFormData {
  title: string
  description: string
  goal: string
  status: string
  startDate: string
  endDate: string
}

export interface SprintDrawerProps {
  isOpen: boolean
  onClose: () => void
  /** 'create' | 'edit' | 'view' — view 模式下字段只读、不显示保存按钮 */
  mode: 'create' | 'edit' | 'view'
  projectSlug: string
  /** 编辑/查看时传入初始数据；新建时可不传 */
  initialData?: Partial<SprintFormData>
  /** 保存回调，由父组件负责 API 调用、刷新列表、toast 等 */
  onSave?: (data: SprintFormData) => Promise<void>
  /** view 模式下显示「编辑」按钮，点击后由父组件切换 mode 为 edit */
  onEdit?: () => void
}

export default function SprintDrawer({
  isOpen,
  onClose,
  mode,
  projectSlug,
  initialData,
  onSave,
  onEdit,
}: SprintDrawerProps) {
  const { t } = useI18n()
  const isReadOnly = mode === 'view'

  const [formData, setFormData] = useState<SprintFormData>({
    title: '',
    description: '',
    goal: '',
    status: 'open',
    startDate: '',
    endDate: '',
  })
  const [isSaving, setIsSaving] = useState(false)
  // AI 优化 Modal 开关
  const [isAIOptimizeOpen, setIsAIOptimizeOpen] = useState(false)

  // AI 优化结果应用：把优化后的 title/description/goal 回填到表单
  const handleAIOptimizeApply = (result: AIOptimizeSprintResult) => {
    setFormData(prev => ({
      ...prev,
      title: result.title,
      description: result.description,
      goal: result.goal,
    }))
  }

  // 打开抽屉时，用 initialData 初始化表单
  useEffect(() => {
    if (isOpen) {
      setFormData({
        title: initialData?.title || '',
        description: initialData?.description || '',
        goal: initialData?.goal || '',
        status: initialData?.status || 'open',
        // 日期字段后端返回 ISO 字符串，截取 YYYY-MM-DD 部分给 <input type="date">
        startDate: initialData?.startDate ? String(initialData.startDate).split('T')[0] : '',
        endDate: initialData?.endDate ? String(initialData.endDate).split('T')[0] : '',
      })
    }
  }, [isOpen, initialData])

  // 设置开始日期时，若结束日期为空则自动填充为开始日期 +2 周（14 天）
  const update = <K extends keyof SprintFormData>(key: K, value: SprintFormData[K]) => {
    setFormData(prev => {
      const next = { ...prev, [key]: value }
      if (key === 'startDate' && value && !prev.endDate) {
        const d = new Date(value + 'T00:00:00')
        d.setDate(d.getDate() + 14)
        const y = d.getFullYear()
        const m = String(d.getMonth() + 1).padStart(2, '0')
        const day = String(d.getDate()).padStart(2, '0')
        next.endDate = `${y}-${m}-${day}`
      }
      return next
    })
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

  const headerTitle = mode === 'edit'
    ? t('pms.editSprint')
    : mode === 'view'
      ? t('pms.viewSprint')
      : t('pms.newSprint')

  return (
    <>
    <Drawer isOpen={isOpen} placement="right" onClose={onClose} size="md" blockScrollOnMount={false}>
      <DrawerOverlay />
      <DrawerContent>
        <DrawerCloseButton />
        <DrawerHeader borderBottomWidth="1px">{headerTitle}</DrawerHeader>

        <DrawerBody>
          <Stack spacing={4}>
            {/* 迭代标题 + AI 优化按钮 */}
            <Box>
              <Flex justify="space-between" align="center" mb={2}>
                <Text fontSize="sm" color="gray.700">{t('pms.sprintTitle')}</Text>
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
                placeholder={t('pms.sprintTitlePlaceholder')}
                isReadOnly={isReadOnly}
              />
            </Box>

            {/* 迭代描述 */}
            <Box>
              <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.sprintDescription')}</Text>
              <Textarea
                value={formData.description}
                onChange={(e) => update('description', e.target.value)}
                placeholder={t('pms.sprintDescriptionPlaceholder')}
                resize="none"
                rows={4}
                isReadOnly={isReadOnly}
              />
            </Box>

            {/* 迭代目标 */}
            <Box>
              <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.sprintGoal')}</Text>
              <Textarea
                value={formData.goal}
                onChange={(e) => update('goal', e.target.value)}
                placeholder={t('pms.sprintGoalPlaceholder')}
                resize="none"
                rows={3}
                isReadOnly={isReadOnly}
              />
            </Box>

            {/* 状态 */}
            <Box>
              <Text fontSize="sm" color="gray.700" mb={2}>{t('common.status')}</Text>
              <Select
                value={formData.status}
                onChange={(e) => update('status', e.target.value)}
                isDisabled={isReadOnly}
                borderColor={formData.status === 'active' ? 'green.500' : formData.status === 'closed' ? 'red.500' : 'gray.500'}
                color={formData.status === 'active' ? 'green.500' : formData.status === 'closed' ? 'red.500' : 'gray.500'}
                sx={{ option: { color: 'gray.700' } }}
              >
                <option value="open">{t('pms.statusOpen')}</option>
                <option value="active">{t('pms.statusActive')}</option>
                <option value="closed">{t('pms.statusClosed')}</option>
              </Select>
            </Box>

            {/* 开始日期 + 结束日期（同一行显示） */}
            <HStack spacing={4} align="flex-start">
              <Box flex={1}>
                <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.startDate')}</Text>
                <Input
                  type="date"
                  value={formData.startDate}
                  onChange={(e) => update('startDate', e.target.value)}
                  isReadOnly={isReadOnly}
                />
              </Box>
              <Box flex={1}>
                <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.endDate')}</Text>
                <Input
                  type="date"
                  value={formData.endDate}
                  onChange={(e) => update('endDate', e.target.value)}
                  isReadOnly={isReadOnly}
                />
              </Box>
            </HStack>
          </Stack>
        </DrawerBody>

        <DrawerFooter borderTopWidth="1px">
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

      {/* AI 优化迭代草稿弹窗（新建/编辑共用） */}
      <AIOptimizeModal
        isOpen={isAIOptimizeOpen}
        onClose={() => setIsAIOptimizeOpen(false)}
        projectSlug={projectSlug}
        type="sprint"
        draft={{
          title: formData.title,
          description: formData.description,
          goal: formData.goal,
        }}
        onApply={handleAIOptimizeApply}
      />
    </>
  )
}
