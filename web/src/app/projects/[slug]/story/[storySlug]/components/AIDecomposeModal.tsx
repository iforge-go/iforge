'use client'

import { useEffect, useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import {
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  Button,
  VStack,
  HStack,
  Text,
  Input,
  Textarea,
  Select,
  Checkbox,
  IconButton,
  Spinner,
  Flex,
  Box,
  Stack,
  Link,
} from '@chakra-ui/react'
import { FiRefreshCw, FiCheck, FiTrash2 } from 'react-icons/fi'
import { api, AITaskSuggestion } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

interface AIDecomposeModalProps {
  isOpen: boolean
  onClose: () => void
  storySlug: string
  projectSlug: string
  onTasksCreated: (tasks: any[]) => void
}

export default function AIDecomposeModal({
  isOpen,
  onClose,
  storySlug,
  projectSlug,
  onTasksCreated,
}: AIDecomposeModalProps) {
  const router = useRouter()
  const toast = useGithubToast()
  const { t, locale } = useI18n()
  const [loading, setLoading] = useState(false)
  const [suggestions, setSuggestions] = useState<AITaskSuggestion[]>([])
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // 'not_configured' 特殊错误：显示跳转链接
  const [notConfigured, setNotConfigured] = useState(false)
  // SSE 流式文本：实时展示 AI 的思考过程
  const [streamingText, setStreamingText] = useState('')
  // 发给 AI 的完整提示（系统消息 + 用户故事详情），让用户看到需求是怎么提的
  const [promptText, setPromptText] = useState('')
  // AI 的思考过程（reasoning_content，DeepSeek 等模型支持）
  const [thinkingText, setThinkingText] = useState('')
  const streamRef = useRef<HTMLDivElement>(null)

  const loadDecompose = async () => {
    setLoading(true)
    setError(null)
    setNotConfigured(false)
    setSuggestions([])
    setStreamingText('')
    setPromptText('')
    setThinkingText('')
    try {
      const result = await api.aiDecomposeUserStory(
        storySlug,
        locale,
        // onDelta: AI 的最终 JSON 输出
        (delta: string) => {
          setStreamingText((prev) => {
            const next = prev + delta
            autoScroll()
            return next
          })
        },
        // onThinking: AI 的思考过程（reasoning_content）
        (thinking: string) => {
          setThinkingText((prev) => {
            const next = prev + thinking
            autoScroll()
            return next
          })
        },
        // onPrompt: 发给 AI 的完整提示
        (prompt: string) => {
          setPromptText(prompt)
          autoScroll()
        },
      )
      const tasks = result.tasks || []
      setSuggestions(tasks)
      // 默认全选
      setSelected(new Set(tasks.map((_: AITaskSuggestion, i: number) => i)))
    } catch (e: any) {
      const status = e?.status
      const msg: string = e?.message || ''
      if (status === 412 || msg.includes('ai_not_configured') || msg.includes('not_configured')) {
        setNotConfigured(true)
      } else {
        setError(msg || 'Failed to decompose')
      }
    } finally {
      setLoading(false)
    }
  }

  // 自动滚动到容器底部，展示最新内容
  const autoScroll = () => {
    requestAnimationFrame(() => {
      if (streamRef.current) {
        streamRef.current.scrollTop = streamRef.current.scrollHeight
      }
    })
  }

  const toggleSelect = (index: number) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(index)) {
        next.delete(index)
      } else {
        next.add(index)
      }
      return next
    })
  }

  const updateSuggestion = (index: number, patch: Partial<AITaskSuggestion>) => {
    setSuggestions((prev) => prev.map((s, i) => (i === index ? { ...s, ...patch } : s)))
  }

  const removeSuggestion = (index: number) => {
    setSuggestions((prev) => {
      const next = prev.filter((_, i) => i !== index)
      // 重建 selected：保留仍存在且被选中的索引
      setSelected((prevSel) => {
        const newSel = new Set<number>()
        next.forEach((_, i) => {
          if (prevSel.has(i >= index ? i + 1 : i)) {
            newSel.add(i)
          }
        })
        return newSel
      })
      return next
    })
  }

  const handleConfirm = async () => {
    if (selected.size === 0) {
      toast({ title: t('pms.aiDecomposeNoSelection'), status: 'warning', duration: 2000 })
      return
    }
    setCreating(true)
    const picked = suggestions.filter((_, i) => selected.has(i))
    const created: any[] = []
    let failed = 0
    for (const s of picked) {
      try {
        const task = await api.createTask(projectSlug, {
          title: s.title,
          description: s.description || undefined,
          status: 'todo',
          priority: s.priority,
          taskType: s.taskType,
          storyPoints: s.storyPoints || 0,
          userStorySlug: storySlug,
        })
        created.push(task)
      } catch (e) {
        failed++
      }
    }
    setCreating(false)
    if (created.length > 0) {
      onTasksCreated(created)
    }
    if (failed === 0) {
      toast({ title: t('pms.aiDecomposeCreated', { count: created.length }), status: 'success', duration: 2000 })
      onClose()
    } else if (created.length > 0) {
      toast({
        title: t('pms.aiDecomposePartialFailed', { success: created.length, total: picked.length }),
        status: 'warning',
        duration: 3000,
      })
      onClose()
    } else {
      toast({ title: t('common.error'), description: t('pms.aiDecomposeEmpty'), status: 'error', duration: 3000 })
    }
  }

  const handleClose = () => {
    // 重置状态以便下次打开是干净的
    setSuggestions([])
    setSelected(new Set())
    setError(null)
    setNotConfigured(false)
    setStreamingText('')
    setPromptText('')
    setThinkingText('')
    onClose()
  }

  // 打开时自动调用 AI 拆解
  useEffect(() => {
    if (isOpen) {
      loadDecompose()
    }
  }, [isOpen]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Modal isOpen={isOpen} onClose={handleClose} size="2xl" blockScrollOnMount={false}>
      <ModalOverlay />
      <ModalContent>
        <ModalHeader borderBottomWidth="1px">
          <Text fontWeight="bold">{t('pms.aiDecomposeTitle')}</Text>
          <Text fontSize="sm" fontWeight="normal" color="gray.500" mt={1}>
            {t('pms.aiDecomposeSubtitle')}
          </Text>
        </ModalHeader>
        <ModalCloseButton />

        <ModalBody py={6}>
          {loading ? (
            <Stack spacing={4}>
              <HStack spacing={3}>
                <Spinner size="sm" color="purple.500" />
                <Text fontSize="sm" color="gray.600">
                  {t('pms.aiDecomposeLoading')}
                </Text>
              </HStack>
              {/* 可滚动容器：包含请求、思考过程、生成结果三段 */}
              <Box ref={streamRef} maxH="420px" overflowY="auto" pr={1}>
                {/* 1. 发送给 AI 的请求（系统消息 + 用户故事详情） */}
                {promptText && (
                  <Box mb={4}>
                    <Text fontSize="xs" fontWeight="bold" color="blue.600" mb={2}>
                      📋 {t('pms.aiDecomposeRequest')}
                    </Text>
                    <Box
                      bg="blue.50"
                      borderWidth="1px"
                      borderColor="blue.100"
                      borderRadius="md"
                      p={3}
                      fontFamily="Consolas, 'SF Mono', Menlo, Monaco, 'Courier New', monospace"
                      fontSize="xs"
                      lineHeight="1.6"
                      color="gray.700"
                      whiteSpace="pre-wrap"
                      wordBreak="break-all"
                    >
                      {promptText}
                    </Box>
                  </Box>
                )}
                {/* 2. AI 思考过程（reasoning_content，DeepSeek 等模型支持） */}
                {thinkingText && (
                  <Box mb={4}>
                    <Text fontSize="xs" fontWeight="bold" color="orange.600" mb={2}>
                      🧠 {t('pms.aiDecomposeThinking')}
                    </Text>
                    <Box
                      bg="orange.50"
                      borderWidth="1px"
                      borderColor="orange.100"
                      borderRadius="md"
                      p={3}
                      fontSize="sm"
                      lineHeight="1.7"
                      color="gray.700"
                      fontStyle="italic"
                      whiteSpace="pre-wrap"
                      wordBreak="break-all"
                    >
                      {thinkingText}
                    </Box>
                  </Box>
                )}
                {/* 3. 生成结果（AI 的 JSON 输出，终端风格） */}
                <Box>
                  <Text fontSize="xs" fontWeight="bold" color="green.600" mb={2}>
                    📝 {t('pms.aiDecomposeResult')}
                  </Text>
                  <Box
                    bg="gray.900"
                    borderRadius="md"
                    p={3}
                    minH="60px"
                    fontFamily="Consolas, 'SF Mono', Menlo, Monaco, 'Courier New', monospace"
                    fontSize="xs"
                    lineHeight="1.6"
                    color="green.300"
                    whiteSpace="pre-wrap"
                    wordBreak="break-all"
                  >
                    {streamingText || '...'}
                  </Box>
                </Box>
              </Box>
            </Stack>
          ) : notConfigured ? (
            <Flex direction="column" align="center" justify="center" py={12}>
              <Text color="gray.600" mb={4}>
                {t('pms.aiDecomposeNotConfigured')}
              </Text>
              <Button colorScheme="purple" onClick={() => router.push('/admin')}>
                {t('pms.aiDecomposeGoToSettings')}
              </Button>
            </Flex>
          ) : error ? (
            <Flex direction="column" align="center" justify="center" py={12}>
              <Text color="red.500" mb={4}>
                {error}
              </Text>
              <Button leftIcon={<FiRefreshCw />} variant="outline" onClick={loadDecompose}>
                {t('pms.aiDecomposeRegenerate')}
              </Button>
            </Flex>
          ) : suggestions.length === 0 ? (
            <Flex direction="column" align="center" justify="center" py={12}>
              <Text color="gray.500" mb={4}>
                {t('pms.aiDecomposeEmpty')}
              </Text>
              <Button leftIcon={<FiRefreshCw />} variant="outline" onClick={loadDecompose}>
                {t('pms.aiDecomposeRegenerate')}
              </Button>
            </Flex>
          ) : (
            <VStack spacing={3} align="stretch">
              {suggestions.map((s, i) => (
                <Box
                  key={i}
                  p={3}
                  borderWidth="1px"
                  borderRadius="md"
                  borderColor={selected.has(i) ? 'purple.200' : 'gray.200'}
                  bg={selected.has(i) ? 'purple.50' : 'white'}
                >
                  <HStack align="flex-start" spacing={3}>
                    <Checkbox
                      isChecked={selected.has(i)}
                      onChange={() => toggleSelect(i)}
                      colorScheme="purple"
                      mt={2}
                    />
                    <VStack spacing={2} align="stretch" flex={1}>
                      <Input
                        value={s.title}
                        onChange={(e) => updateSuggestion(i, { title: e.target.value })}
                        placeholder={t('pms.taskTitle')}
                        size="sm"
                        fontWeight="medium"
                      />
                      <Textarea
                        value={s.description}
                        onChange={(e) => updateSuggestion(i, { description: e.target.value })}
                        placeholder={t('pms.taskDescription')}
                        size="sm"
                        rows={2}
                        resize="none"
                      />
                      <HStack spacing={2}>
                        <Select
                          value={s.priority}
                          onChange={(e) => updateSuggestion(i, { priority: e.target.value as AITaskSuggestion['priority'] })}
                          size="sm"
                          flex={1}
                        >
                          <option value="urgent">{t('pms.urgent')}</option>
                          <option value="high">{t('pms.high')}</option>
                          <option value="medium">{t('pms.medium')}</option>
                          <option value="low">{t('pms.low')}</option>
                        </Select>
                        <Select
                          value={s.taskType}
                          onChange={(e) => updateSuggestion(i, { taskType: e.target.value as AITaskSuggestion['taskType'] })}
                          size="sm"
                          flex={1}
                        >
                          <option value="task">{t('pms.taskTypeTask') || 'Task'}</option>
                          <option value="feature">{t('pms.taskTypeFeature') || 'Feature'}</option>
                          <option value="bug">{t('pms.taskTypeBug') || 'Bug'}</option>
                          <option value="improvement">{t('pms.taskTypeImprovement') || 'Improvement'}</option>
                        </Select>
                        <Input
                          type="number"
                          value={s.storyPoints ?? 0}
                          onChange={(e) => updateSuggestion(i, { storyPoints: parseInt(e.target.value) || 0 })}
                          size="sm"
                          w="80px"
                          min={0}
                          max={100}
                          placeholder="SP"
                        />
                        <IconButton
                          aria-label={t('common.delete')}
                          icon={<FiTrash2 />}
                          size="sm"
                          variant="ghost"
                          color="gray.400"
                          _hover={{ color: 'red.500' }}
                          onClick={() => removeSuggestion(i)}
                        />
                      </HStack>
                    </VStack>
                  </HStack>
                </Box>
              ))}
            </VStack>
          )}
        </ModalBody>

        <ModalFooter borderTopWidth="1px">
          <Text fontSize="sm" color="gray.500" mr="auto">
            {t('pms.aiDecomposeSelected', { selected: selected.size, total: suggestions.length })}
          </Text>
          <HStack spacing={2}>
            {!loading && !notConfigured && (
              <Button
                variant="ghost"
                leftIcon={<FiRefreshCw />}
                onClick={loadDecompose}
                isLoading={loading}
                isDisabled={creating}
              >
                {t('pms.aiDecomposeRegenerate')}
              </Button>
            )}
            <Button variant="ghost" onClick={handleClose} isDisabled={creating}>
              {t('common.cancel')}
            </Button>
            <Button
              variant="primary"
              leftIcon={<FiCheck />}
              onClick={handleConfirm}
              isLoading={creating}
              isDisabled={loading || selected.size === 0 || notConfigured || suggestions.length === 0}
            >
              {t('pms.aiDecomposeConfirm')}
            </Button>
          </HStack>
        </ModalFooter>
      </ModalContent>
    </Modal>
  )
}
