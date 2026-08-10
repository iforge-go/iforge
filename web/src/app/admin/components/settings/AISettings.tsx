'use client'

import {
  Box, VStack, HStack, Text, Button, FormControl, FormLabel, Input, Switch, Icon, Select,
  Modal, ModalOverlay, ModalContent, ModalHeader, ModalBody, ModalFooter, ModalCloseButton,
  AlertDialog, AlertDialogBody, AlertDialogContent, AlertDialogFooter, AlertDialogHeader, AlertDialogOverlay,
  Badge, IconButton, Spinner, Tooltip, useDisclosure,
} from '@chakra-ui/react'
import { useEffect, useRef, useState } from 'react'
import { api } from '@/lib/api'
import { AIModelConfig, AIModelConfigInput } from '@/lib/types'
import { FiCpu, FiPlus, FiEdit, FiTrash2, FiEye, FiEyeOff, FiZap, FiStar } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

// Provider 预设：选择预设 provider 时自动填充 baseUrl 和 model
const PROVIDER_PRESETS = [
  { value: 'openai', labelKey: 'admin.aiProviderOpenAI', baseUrl: 'https://api.openai.com/v1', model: 'gpt-4o-mini' },
  { value: 'deepseek', labelKey: 'admin.aiProviderDeepSeek', baseUrl: 'https://api.deepseek.com/v1', model: 'deepseek-chat' },
  { value: 'qwen', labelKey: 'admin.aiProviderQwen', baseUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1', model: 'qwen-plus' },
  { value: 'zhipu', labelKey: 'admin.aiProviderZhipu', baseUrl: 'https://open.bigmodel.cn/api/paas/v4', model: 'glm-4-flash' },
  { value: 'moonshot', labelKey: 'admin.aiProviderMoonshot', baseUrl: 'https://api.moonshot.cn/v1', model: 'moonshot-v1-8k' },
  { value: 'custom', labelKey: 'admin.aiProviderCustom', baseUrl: '', model: '' },
]

// 空表单默认值
const EMPTY_FORM: AIModelConfigInput = {
  name: '',
  provider: 'deepseek',
  baseUrl: 'https://api.deepseek.com/v1',
  apiKey: '',
  model: 'deepseek-chat',
  timeout: 60,
  maxTokens: 8196,
  enabled: true,
  isDefault: false,
}

export default function AISettingsPage() {
  const toast = useGithubToast()
  const { t } = useI18n()
  const [configs, setConfigs] = useState<AIModelConfig[]>([])
  const [loading, setLoading] = useState(true)

  // 编辑/创建弹窗状态
  const { isOpen: isModalOpen, onOpen: onModalOpen, onClose: onModalClose } = useDisclosure()
  const [form, setForm] = useState<AIModelConfigInput>(EMPTY_FORM)
  const [editingId, setEditingId] = useState<number | null>(null) // null = 创建模式
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [testingId, setTestingId] = useState<number | null>(null) // 列表卡片测试中
  const [showPassword, setShowPassword] = useState(false)

  // 删除确认弹窗状态
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()
  const [deletingId, setDeletingId] = useState<number | null>(null)
  const [deleting, setDeleting] = useState(false)
  const cancelRef = useRef(null as any)

  const loadConfigs = async () => {
    try {
      const data = await api.listAIModelConfigs()
      setConfigs(data || [])
    } catch (error) {
      console.error('Failed to load AI model configs:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadConfigs()
  }, [])

  // 切换 provider：若 baseUrl 为空或属于其他预设的值，自动填充新 provider 的 baseUrl 和 model
  const handleProviderChange = (newProvider: string) => {
    const preset = PROVIDER_PRESETS.find((p) => p.value === newProvider)
    if (!preset) return
    const allPresetBaseUrls = PROVIDER_PRESETS.map((p) => p.baseUrl).filter(Boolean)
    const currentBaseUrl = form.baseUrl || ''
    const shouldAutoFill = !currentBaseUrl || allPresetBaseUrls.includes(currentBaseUrl)
    setForm({
      ...form,
      provider: newProvider,
      baseUrl: shouldAutoFill ? preset.baseUrl : currentBaseUrl,
      model: shouldAutoFill ? preset.model : (form.model || ''),
    })
  }

  // 打开创建弹窗
  const handleCreate = () => {
    setForm(EMPTY_FORM)
    setEditingId(null)
    setShowPassword(false)
    onModalOpen()
  }

  // 打开编辑弹窗
  const handleEdit = (cfg: AIModelConfig) => {
    setForm({
      name: cfg.name,
      provider: cfg.provider,
      baseUrl: cfg.baseUrl,
      apiKey: cfg.apiKey, // 后端返回的 mask 形式
      model: cfg.model,
      timeout: cfg.timeout,
      maxTokens: cfg.maxTokens,
      enabled: cfg.enabled,
      isDefault: cfg.isDefault,
    })
    setEditingId(cfg.id)
    setShowPassword(false)
    onModalOpen()
  }

  // 保存（创建或更新）
  const handleSave = async () => {
    if (!form.name.trim()) {
      toast({ title: t('admin.aiConfigName') + '不能为空', status: 'warning', duration: 2000 })
      return
    }
    setSaving(true)
    try {
      // API Key mask 处理：若 apiKey 包含 ****（后端 GET 返回的 mask 形式），不传该字段让后端保留原值
      const payload: AIModelConfigInput = { ...form }
      if (payload.apiKey && payload.apiKey.includes('****')) {
        payload.apiKey = ''
      }
      if (editingId === null) {
        await api.createAIModelConfig(payload)
        toast({ title: t('admin.aiConfigCreated'), status: 'success', duration: 2000 })
      } else {
        await api.updateAIModelConfig(editingId, payload)
        toast({ title: t('admin.aiConfigUpdated'), status: 'success', duration: 2000 })
      }
      onModalClose()
      await loadConfigs()
    } catch (error: any) {
      const msg = error?.message || ''
      if (msg.includes('name already exists') || msg.includes('名称已存在')) {
        toast({ title: t('admin.aiConfigNameConflict'), status: 'error', duration: 3000 })
      } else {
        toast({ title: t('admin.saveFailed'), description: msg, status: 'error', duration: 3000 })
      }
    } finally {
      setSaving(false)
    }
  }

  // 测试连接（编辑弹窗内，用表单当前值测试）
  const handleTest = async () => {
    if (editingId === null) return // 创建模式禁用
    setTesting(true)
    try {
      // 若 apiKey 是 mask 形式，不传 apiKey 让后端从数据库取原值
      const input: Partial<AIModelConfigInput> = {
        provider: form.provider,
        baseUrl: form.baseUrl,
        model: form.model,
        timeout: form.timeout,
        maxTokens: form.maxTokens,
      }
      if (form.apiKey && !form.apiKey.includes('****')) {
        input.apiKey = form.apiKey
      }
      await api.testAIModelConfig(editingId, input)
      toast({ title: t('admin.aiTestSuccess'), status: 'success', duration: 3000 })
      // 测试后刷新列表以更新测试状态和测试时间
      await loadConfigs()
    } catch (error: any) {
      toast({ title: t('admin.aiTestFailed'), description: error?.message, status: 'error', duration: 3000 })
      // 测试失败也刷新列表（后端保存了 failed 状态）
      await loadConfigs()
    } finally {
      setTesting(false)
    }
  }

  // 从列表卡片直接测试（使用已保存的配置，不传 body）
  const handleTestFromList = async (cfg: AIModelConfig) => {
    setTestingId(cfg.id)
    try {
      await api.testAIModelConfig(cfg.id, {})
      toast({ title: t('admin.aiTestSuccess'), status: 'success', duration: 3000 })
      await loadConfigs()
    } catch (error: any) {
      toast({ title: t('admin.aiTestFailed'), description: error?.message, status: 'error', duration: 3000 })
      await loadConfigs()
    } finally {
      setTestingId(null)
    }
  }

  // 格式化测试时间为相对时间显示
  const formatTestTime = (testedAt: string | null): string => {
    if (!testedAt) return ''
    const d = new Date(testedAt)
    const now = new Date()
    const diffMs = now.getTime() - d.getTime()
    const diffMin = Math.floor(diffMs / 60000)
    if (diffMin < 1) return t('common.justNow')
    if (diffMin < 60) return t('common.minutesAgo', { count: diffMin })
    const diffHour = Math.floor(diffMin / 60)
    if (diffHour < 24) return t('common.hoursAgo', { count: diffHour })
    return d.toLocaleString()
  }

  // 设为默认
  const handleSetDefault = async (id: number) => {
    try {
      await api.setDefaultAIModelConfig(id)
      toast({ title: t('admin.aiConfigDefaultSet'), status: 'success', duration: 2000 })
      await loadConfigs()
    } catch (error: any) {
      toast({ title: t('admin.saveFailed'), description: error?.message, status: 'error', duration: 3000 })
    }
  }

  // 打开删除确认
  const handleDeleteClick = (cfg: AIModelConfig) => {
    setDeletingId(cfg.id)
    onDeleteOpen()
  }

  // 确认删除
  const handleDeleteConfirm = async () => {
    if (deletingId === null) return
    setDeleting(true)
    try {
      await api.deleteAIModelConfig(deletingId)
      toast({ title: t('admin.aiConfigDeleted'), status: 'success', duration: 2000 })
      onDeleteClose()
      await loadConfigs()
    } catch (error: any) {
      const msg = error?.message || ''
      if (msg.includes('cannot delete the default') || msg.includes('默认')) {
        toast({ title: t('admin.aiConfigDeleteFailed'), status: 'error', duration: 3000 })
      } else {
        toast({ title: t('admin.saveFailed'), description: msg, status: 'error', duration: 3000 })
      }
    } finally {
      setDeleting(false)
    }
  }

  if (loading) {
    return <Spinner size="md" />
  }

  return (
    <Box>
      {/* 标题 + 新建按钮 */}
      <HStack spacing={3} mb={4} justify="space-between">
        <HStack spacing={3}>
          <Icon as={FiCpu} w={5} h={5} color="primary.500" />
          <Box>
            <Text fontWeight="bold" fontSize="lg" color="myGray.900">
              {t('admin.aiConfigTitle')}
            </Text>
            <Text fontSize="sm" color="myGray.500">
              {t('admin.aiConfigSubtitle')}
            </Text>
          </Box>
        </HStack>
        <Button leftIcon={<Icon as={FiPlus} />} variant="primary" size="sm" onClick={handleCreate}>
          {t('admin.aiConfigAdd')}
        </Button>
      </HStack>

      {/* 配置列表 */}
      {configs.length === 0 ? (
        <Box borderWidth="1px" borderRadius="lg" bg="white" p={8} textAlign="center">
          <Text color="myGray.500">{t('admin.aiConfigEmpty')}</Text>
        </Box>
      ) : (
        <VStack spacing={3} align="stretch">
          {configs.map((cfg) => (
            <Box
              key={cfg.id}
              borderWidth={cfg.isDefault ? '2px' : '1px'}
              borderRadius="lg"
              bg={cfg.isDefault ? 'green.50' : 'white'}
              borderColor={cfg.isDefault ? 'green.400' : 'myGray.200'}
              p={4}
              _hover={{ borderColor: cfg.isDefault ? 'green.500' : 'myGray.300' }}
            >
              <HStack spacing={4} align="center" justify="space-between">
                {/* 左侧：名称 + 徽章 + 摘要 */}
                <HStack spacing={3} flex={1} minW={0}>
                  <Box minW={0} flex={1}>
                    <HStack spacing={2} mb={1}>
                      <Text fontWeight="semibold" color="myGray.900" isTruncated>
                        {cfg.name}
                      </Text>
                      {cfg.isDefault && (
                        <Badge colorScheme="green" variant="subtle">
                          {t('admin.aiConfigDefault')}
                        </Badge>
                      )}
                      <Badge colorScheme={cfg.enabled ? 'blue' : 'gray'} variant="subtle">
                        {cfg.enabled ? t('admin.aiConfigEnabled') : t('admin.aiConfigDisabled')}
                      </Badge>
                      {cfg.testStatus === 'success' && (
                        <Badge colorScheme="green" variant="outline">
                          {t('admin.aiTestStatusSuccess')}
                        </Badge>
                      )}
                      {cfg.testStatus === 'failed' && (
                        <Badge colorScheme="red" variant="outline">
                          {t('admin.aiTestStatusFailed')}
                        </Badge>
                      )}
                    </HStack>
                    <Text fontSize="sm" color="myGray.500" isTruncated>
                      {cfg.provider} · {cfg.model}
                    </Text>
                    {cfg.testedAt && (
                      <Text fontSize="xs" color="myGray.400" mt={1}>
                        {t('admin.aiTestLastTime')}: {formatTestTime(cfg.testedAt)}
                      </Text>
                    )}
                  </Box>
                </HStack>

                {/* 右侧：操作按钮 */}
                <HStack spacing={1}>
                  {!cfg.isDefault && (
                    <Tooltip label={t('admin.aiConfigSetDefault')} hasArrow>
                      <IconButton
                        aria-label={t('admin.aiConfigSetDefault')}
                        icon={<Icon as={FiStar} />}
                        size="xs"
                        variant="ghost"
                        color="gray.400"
                        _hover={{ color: 'green.500' }}
                        onClick={() => handleSetDefault(cfg.id)}
                      />
                    </Tooltip>
                  )}
                  <Tooltip label={t('admin.aiTestConnection')} hasArrow>
                    <IconButton
                      aria-label={t('admin.aiTestConnection')}
                      icon={testingId === cfg.id ? <Spinner size="xs" /> : <Icon as={FiZap} />}
                      size="xs"
                      variant="ghost"
                      color="gray.400"
                      _hover={{ color: 'orange.500' }}
                      isDisabled={testingId !== null}
                      onClick={() => handleTestFromList(cfg)}
                    />
                  </Tooltip>
                  <Tooltip label={t('admin.aiConfigEdit')} hasArrow>
                    <IconButton
                      aria-label={t('admin.aiConfigEdit')}
                      icon={<Icon as={FiEdit} />}
                      size="xs"
                      variant="ghost"
                      color="gray.400"
                      _hover={{ color: 'primary.500' }}
                      onClick={() => handleEdit(cfg)}
                    />
                  </Tooltip>
                  <Tooltip label={cfg.isDefault ? t('admin.aiConfigDeleteFailed') : t('common.delete')} hasArrow>
                    <IconButton
                      aria-label={t('common.delete')}
                      icon={<Icon as={FiTrash2} />}
                      size="xs"
                      variant="ghost"
                      color="gray.400"
                      isDisabled={cfg.isDefault}
                      _hover={cfg.isDefault ? {} : { color: 'red.500' }}
                      onClick={() => handleDeleteClick(cfg)}
                    />
                  </Tooltip>
                </HStack>
              </HStack>
            </Box>
          ))}
        </VStack>
      )}

      {/* 编辑/创建弹窗 — blockScrollOnMount 防止开关弹窗时页面抖动 */}
      <Modal isOpen={isModalOpen} onClose={onModalClose} size="lg" blockScrollOnMount={false}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>
            {editingId === null ? t('admin.aiConfigCreate') : t('admin.aiConfigEdit')}
          </ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4} align="stretch">
              <FormControl>
                <FormLabel>{t('admin.aiConfigName')}</FormLabel>
                <Input
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder={t('admin.aiConfigNamePlaceholder')}
                />
              </FormControl>

              <HStack spacing={4}>
                <FormControl display="flex" alignItems="center">
                  <FormLabel htmlFor="cfg-enabled" mb="0" flex="1">
                    {t('admin.aiConfigEnabled')}
                  </FormLabel>
                  <Switch
                    id="cfg-enabled"
                    colorScheme="primary"
                    isChecked={form.enabled}
                    onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
                  />
                </FormControl>
                <FormControl display="flex" alignItems="center">
                  <FormLabel htmlFor="cfg-default" mb="0" flex="1">
                    {t('admin.aiConfigIsDefault')}
                  </FormLabel>
                  <Switch
                    id="cfg-default"
                    colorScheme="green"
                    isChecked={form.isDefault}
                    onChange={(e) => setForm({ ...form, isDefault: e.target.checked })}
                  />
                </FormControl>
              </HStack>

              <FormControl>
                <FormLabel>{t('admin.aiProvider')}</FormLabel>
                <Select value={form.provider} onChange={(e) => handleProviderChange(e.target.value)}>
                  {PROVIDER_PRESETS.map((p) => (
                    <option key={p.value} value={p.value}>
                      {t(p.labelKey)}
                    </option>
                  ))}
                </Select>
              </FormControl>

              <FormControl>
                <FormLabel>{t('admin.aiBaseUrl')}</FormLabel>
                <Input
                  value={form.baseUrl}
                  onChange={(e) => setForm({ ...form, baseUrl: e.target.value })}
                  placeholder="https://api.example.com/v1"
                />
              </FormControl>

              <FormControl>
                <FormLabel>{t('admin.aiApiKey')}</FormLabel>
                <HStack>
                  <Input
                    type={showPassword ? 'text' : 'password'}
                    value={form.apiKey}
                    onChange={(e) => setForm({ ...form, apiKey: e.target.value })}
                    placeholder="sk-••••••••"
                  />
                  <IconButton
                    aria-label={showPassword ? t('common.hide') : t('common.show')}
                    icon={<Icon as={showPassword ? FiEyeOff : FiEye} />}
                    size="sm"
                    variant="ghost"
                    onClick={() => setShowPassword(!showPassword)}
                  />
                </HStack>
              </FormControl>

              <FormControl>
                <FormLabel>{t('admin.aiModel')}</FormLabel>
                <Input
                  value={form.model}
                  onChange={(e) => setForm({ ...form, model: e.target.value })}
                  placeholder="gpt-4o-mini"
                />
              </FormControl>

              <HStack spacing={4}>
                <FormControl>
                  <FormLabel>{t('admin.aiTimeout')}</FormLabel>
                  <Input
                    type="number"
                    value={form.timeout}
                    onChange={(e) => setForm({ ...form, timeout: parseInt(e.target.value) || 60 })}
                    min={5}
                    max={300}
                  />
                </FormControl>
                <FormControl>
                  <FormLabel>{t('admin.aiMaxTokens')}</FormLabel>
                  <Input
                    type="number"
                    value={form.maxTokens}
                    onChange={(e) => setForm({ ...form, maxTokens: parseInt(e.target.value) || 8196 })}
                    min={100}
                    max={32000}
                  />
                </FormControl>
              </HStack>
            </VStack>
          </ModalBody>

          <ModalFooter>
            <HStack spacing={3}>
              {editingId !== null && (
                <Button variant="whiteBase" isLoading={testing} onClick={handleTest} leftIcon={<Icon as={FiZap} />}>
                  {t('admin.aiTestConnection')}
                </Button>
              )}
              <Button variant="ghost" onClick={onModalClose}>
                {t('common.cancel')}
              </Button>
              <Button variant="primary" isLoading={saving} onClick={handleSave}>
                {t('common.save')}
              </Button>
            </HStack>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* 删除确认 AlertDialog */}
      <AlertDialog
        isOpen={isDeleteOpen}
        leastDestructiveRef={cancelRef}
        onClose={onDeleteClose}
        blockScrollOnMount={false}
      >
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              {t('common.delete')}
            </AlertDialogHeader>
            <AlertDialogBody>
              {t('admin.aiConfigDeleteConfirm')}
            </AlertDialogBody>
            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={onDeleteClose}>
                {t('common.cancel')}
              </Button>
              <Button colorScheme="red" onClick={handleDeleteConfirm} ml={3} isLoading={deleting}>
                {t('common.delete')}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>
    </Box>
  )
}
