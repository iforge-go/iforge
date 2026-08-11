'use client'

import { useState, useEffect } from 'react'
import {
  Box,
  Button,
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
import { FiEdit } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

export interface EpicFormData {
  title: string
  description: string
  goal: string
  status: string
  priority: string
  startDate: string
  targetDate: string
  ownerName: string
}

export interface EpicDrawerProps {
  isOpen: boolean
  onClose: () => void
  /** 'create' | 'edit' | 'view' — view 模式下字段只读、不显示保存按钮 */
  mode: 'create' | 'edit' | 'view'
  projectSlug: string
  /** 编辑/查看时传入初始数据;新建时可不传 */
  initialData?: Partial<EpicFormData>
  /** 保存回调,由父组件负责 API 调用、刷新列表、toast 等 */
  onSave?: (data: EpicFormData) => Promise<void>
  /** view 模式下显示「编辑」按钮,点击后由父组件切换 mode 为 edit */
  onEdit?: () => void
  /** 项目成员列表,用于 owner 选择 */
  members?: { userName: string; fullName?: string; image?: string | null }[]
}

export default function EpicDrawer({
  isOpen,
  onClose,
  mode,
  projectSlug: _projectSlug,
  initialData,
  onSave,
  onEdit,
  members,
}: EpicDrawerProps) {
  const { t } = useI18n()
  const isReadOnly = mode === 'view'

  const [formData, setFormData] = useState<EpicFormData>({
    title: '',
    description: '',
    goal: '',
    status: 'open',
    priority: 'medium',
    startDate: '',
    targetDate: '',
    ownerName: '',
  })
  const [isSaving, setIsSaving] = useState(false)

  // 打开抽屉时,用 initialData 初始化表单
  useEffect(() => {
    if (isOpen) {
      setFormData({
        title: initialData?.title || '',
        description: initialData?.description || '',
        goal: initialData?.goal || '',
        status: initialData?.status || 'open',
        priority: initialData?.priority || 'medium',
        startDate: initialData?.startDate ? String(initialData.startDate).split('T')[0] : '',
        targetDate: initialData?.targetDate ? String(initialData.targetDate).split('T')[0] : '',
        ownerName: initialData?.ownerName || '',
      })
    }
  }, [isOpen, initialData])

  const update = <K extends keyof EpicFormData>(key: K, value: EpicFormData[K]) => {
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

  const headerTitle = mode === 'edit'
    ? t('pms.editEpic')
    : mode === 'view'
      ? t('pms.viewEpic')
      : t('pms.newEpic')

  return (
    <Drawer isOpen={isOpen} placement="right" onClose={onClose} size="md" blockScrollOnMount={false}>
      <DrawerOverlay />
      <DrawerContent>
        <DrawerCloseButton />
        <DrawerHeader borderBottomWidth="1px">{headerTitle}</DrawerHeader>

        <DrawerBody>
          <Stack spacing={4}>
            {/* Epic 标题(必填) */}
            <Box>
              <Text fontSize="sm" color="gray.700" mb={2}>
                {t('pms.epicTitle')} <Text as="span" color="red.500">*</Text>
              </Text>
              <Input
                value={formData.title}
                onChange={(e) => update('title', e.target.value)}
                placeholder={t('pms.epicTitlePlaceholder')}
                isReadOnly={isReadOnly}
              />
            </Box>

            {/* Epic 描述 */}
            <Box>
              <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.epicDescription')}</Text>
              <Textarea
                value={formData.description}
                onChange={(e) => update('description', e.target.value)}
                placeholder={t('pms.epicDescriptionPlaceholder')}
                resize="none"
                rows={4}
                isReadOnly={isReadOnly}
              />
            </Box>

            {/* Epic 目标 */}
            <Box>
              <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.epicGoal')}</Text>
              <Textarea
                value={formData.goal}
                onChange={(e) => update('goal', e.target.value)}
                placeholder={t('pms.epicGoalPlaceholder')}
                resize="none"
                rows={3}
                isReadOnly={isReadOnly}
              />
            </Box>

            {/* 状态 + 优先级(同一行) */}
            <HStack spacing={4} align="flex-start">
              <Box flex={1}>
                <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.epicStatus')}</Text>
                <Select
                  value={formData.status}
                  onChange={(e) => update('status', e.target.value)}
                  isDisabled={isReadOnly}
                >
                  <option value="open">{t('pms.statusOpen')}</option>
                  <option value="in_progress">{t('pms.statusInProgress')}</option>
                  <option value="done">{t('pms.statusDone')}</option>
                  <option value="closed">{t('pms.statusClosed')}</option>
                </Select>
              </Box>
              <Box flex={1}>
                <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.epicPriority')}</Text>
                <Select
                  value={formData.priority}
                  onChange={(e) => update('priority', e.target.value)}
                  isDisabled={isReadOnly}
                >
                  <option value="low">{t('pms.priorityLow')}</option>
                  <option value="medium">{t('pms.priorityMedium')}</option>
                  <option value="high">{t('pms.priorityHigh')}</option>
                  <option value="urgent">{t('pms.priorityUrgent')}</option>
                </Select>
              </Box>
            </HStack>

            {/* 开始日期 + 目标日期(同一行,Roadmap 时间轴用) */}
            <HStack spacing={4} align="flex-start">
              <Box flex={1}>
                <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.epicStartDate')}</Text>
                <Input
                  type="date"
                  value={formData.startDate}
                  onChange={(e) => update('startDate', e.target.value)}
                  isReadOnly={isReadOnly}
                />
              </Box>
              <Box flex={1}>
                <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.epicTargetDate')}</Text>
                <Input
                  type="date"
                  value={formData.targetDate}
                  onChange={(e) => update('targetDate', e.target.value)}
                  isReadOnly={isReadOnly}
                />
              </Box>
            </HStack>

            {/* 负责人 */}
            <Box>
              <Text fontSize="sm" color="gray.700" mb={2}>{t('pms.epicOwner')}</Text>
              {members && members.length > 0 ? (
                <Select
                  value={formData.ownerName}
                  onChange={(e) => update('ownerName', e.target.value)}
                  isDisabled={isReadOnly}
                  placeholder={t('pms.unassigned')}
                >
                  {members.map((m) => (
                    <option key={m.userName} value={m.userName}>{m.fullName || m.userName}</option>
                  ))}
                </Select>
              ) : (
                <Input
                  value={formData.ownerName}
                  onChange={(e) => update('ownerName', e.target.value)}
                  placeholder={t('pms.epicOwnerPlaceholder')}
                  isReadOnly={isReadOnly}
                />
              )}
            </Box>
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
            <Button colorScheme="blue" onClick={handleSave} isLoading={isSaving} isDisabled={!formData.title.trim()}>
              {mode === 'edit' ? t('common.save') : t('common.create')}
            </Button>
          )}
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  )
}
