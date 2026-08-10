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
  Spinner,
  Flex,
  Box,
  Stack,
  Badge,
} from '@chakra-ui/react'
import { FiRefreshCw, FiCheck, FiZap, FiClock } from 'react-icons/fi'
import { api, AIOptimizeResult, AIOptimizeTaskResult, AIOptimizeSprintResult } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'

type OptimizeType = 'story' | 'task' | 'sprint'
// 三种结果的联合类型：组件内部用字段存在性做类型窄化
type OptimizeResult = AIOptimizeResult | AIOptimizeTaskResult | AIOptimizeSprintResult
// 根据 type 映射到对应的结果类型，让调用方的 onApply 拿到精确类型
type ResultForType<T extends OptimizeType> =
  T extends 'task' ? AIOptimizeTaskResult :
  T extends 'sprint' ? AIOptimizeSprintResult :
  AIOptimizeResult

interface AIOptimizeModalProps<T extends OptimizeType = 'story'> {
  isOpen: boolean
  onClose: () => void
  projectSlug: string
  /** 优化目标类型：'story' 用户故事 / 'task' 任务 / 'sprint' 迭代。默认 'story' 保持向后兼容 */
  type?: T
  /** 当前草稿内容：AI 会基于这些内容进行优化 */
  draft: {
    title: string
    description?: string
    /** 仅 story 模式使用；task/sprint 模式可省略 */
    acceptanceCriteria?: string
    /** 仅 sprint 模式使用 */
    goal?: string
  }
  /** 用户点击「应用优化结果」时回调，把优化后的字段回填到表单 */
  onApply: (result: ResultForType<T>) => void
}

// 类型守卫：区分 story / task / sprint 结果，用于条件渲染对应字段
function isStoryResult(r: OptimizeResult): r is AIOptimizeResult {
  return 'acceptanceCriteria' in r
}
function isTaskResult(r: OptimizeResult): r is AIOptimizeTaskResult {
  return 'taskType' in r
}
function isSprintResult(r: OptimizeResult): r is AIOptimizeSprintResult {
  return 'goal' in r
}

// 格式化对话耗时：< 60s 显示一位小数（3.2s），>= 60s 显示分秒（1m 5s）
function formatDuration(ms: number): string {
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  const minutes = Math.floor(ms / 60000)
  const seconds = Math.floor((ms % 60000) / 1000)
  return `${minutes}m ${seconds}s`
}

export default function AIOptimizeModal<T extends OptimizeType = 'story'>({
  isOpen,
  onClose,
  projectSlug,
  type,
  draft,
  onApply,
}: AIOptimizeModalProps<T>) {
  // type 是泛型 T（可能为 undefined）；解析为具体的 OptimizeType 供内部逻辑使用
  const optimizeType: OptimizeType = type ?? 'story'
  const router = useRouter()
  const { t, locale } = useI18n()
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<OptimizeResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  // 'not_configured' 特殊错误：显示跳转链接
  const [notConfigured, setNotConfigured] = useState(false)
  // SSE 流式文本：实时展示 AI 的最终 JSON 输出
  const [streamingText, setStreamingText] = useState('')
  // 发给 AI 的完整提示（系统消息 + 详情），让用户看到需求是怎么提的
  const [promptText, setPromptText] = useState('')
  // AI 的思考过程（reasoning_content，DeepSeek 等模型支持）
  const [thinkingText, setThinkingText] = useState('')
  // 本次 AI 对话耗时（毫秒），结果返回后计算
  const [elapsedMs, setElapsedMs] = useState<number | null>(null)
  // 结果出来后的视图切换：'process' 对话过程回看（默认）/ 'result' 优化结果表单
  const [viewTab, setViewTab] = useState<'result' | 'process'>('process')
  const streamRef = useRef<HTMLDivElement>(null)
  const processScrollRef = useRef<HTMLDivElement>(null)

  const loadOptimize = async () => {
    setLoading(true)
    setError(null)
    setNotConfigured(false)
    setResult(null)
    setStreamingText('')
    setPromptText('')
    setThinkingText('')
    setElapsedMs(null)
    setViewTab('process')
    // 记录请求发起时间，用于计算对话耗时
    const startedAt = Date.now()
    try {
      // onDelta: AI 的最终 JSON 输出
      const onDelta = (delta: string) => {
        setStreamingText((prev) => {
          const next = prev + delta
          autoScroll()
          return next
        })
      }
      // onThinking: AI 的思考过程（reasoning_content）
      const onThinking = (thinking: string) => {
        setThinkingText((prev) => {
          const next = prev + thinking
          autoScroll()
          return next
        })
      }
      // onPrompt: 发给 AI 的完整提示
      const onPrompt = (prompt: string) => {
        setPromptText(prompt)
        autoScroll()
      }

      // 根据类型调用对应 API
      const res =
        optimizeType === 'task'
          ? await api.aiOptimizeTask(projectSlug, draft, locale, onDelta, onThinking, onPrompt)
          : optimizeType === 'sprint'
            ? await api.aiOptimizeSprint(projectSlug, draft, locale, onDelta, onThinking, onPrompt)
            : await api.aiOptimizeUserStory(projectSlug, draft, locale, onDelta, onThinking, onPrompt)
      setResult(res)
      // 记录本次对话耗时（从请求发起到响应返回）
      setElapsedMs(Date.now() - startedAt)
      // 结果返回后自动切换到「优化结果」tab
      setViewTab('result')
    } catch (e: any) {
      const status = e?.status
      const msg: string = e?.message || ''
      if (status === 412 || msg.includes('ai_not_configured') || msg.includes('not_configured')) {
        setNotConfigured(true)
      } else {
        setError(msg || 'Failed to optimize')
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

  // 打开时自动调用 AI 优化
  useEffect(() => {
    if (isOpen) {
      loadOptimize()
    }
  }, [isOpen]) // eslint-disable-line react-hooks/exhaustive-deps

  const updateResult = (patch: Partial<OptimizeResult>) => {
    setResult((prev) => (prev ? ({ ...prev, ...patch } as OptimizeResult) : prev))
  }

  const handleApply = () => {
    if (!result) return
    // 内部 result 是联合类型；根据 type prop 转换为调用方期望的精确类型
    onApply(result as ResultForType<T>)
    handleClose()
  }

  const handleClose = () => {
    // 重置状态以便下次打开是干净的
    setResult(null)
    setError(null)
    setNotConfigured(false)
    setStreamingText('')
    setPromptText('')
    setThinkingText('')
    setElapsedMs(null)
    setViewTab('process')
    onClose()
  }

  // 切换到「对话过程」tab 时，自动滚到底部展示最新内容
  useEffect(() => {
    if (viewTab === 'process' && streamRef.current) {
      requestAnimationFrame(() => {
        if (streamRef.current) {
          streamRef.current.scrollTop = streamRef.current.scrollHeight
        }
      })
    }
  }, [viewTab])

  // 根据类型选择 i18n 文案
  const titleKey =
    optimizeType === 'task' ? 'pms.aiOptimizeTaskTitle'
      : optimizeType === 'sprint' ? 'pms.aiOptimizeSprintTitle'
        : 'pms.aiOptimizeTitle'
  const subtitleKey =
    optimizeType === 'task' ? 'pms.aiOptimizeTaskSubtitle'
      : optimizeType === 'sprint' ? 'pms.aiOptimizeSprintSubtitle'
        : 'pms.aiOptimizeSubtitle'
  const loadingKey =
    optimizeType === 'task' ? 'pms.aiOptimizeTaskLoading'
      : optimizeType === 'sprint' ? 'pms.aiOptimizeSprintLoading'
        : 'pms.aiOptimizeLoading'

  // 思考过程内容块（prompt + thinking + delta），在加载中和结果回看时复用
  const processBlocks = (
    <>
      {/* 1. 发送给 AI 的请求（系统消息 + 详情） */}
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
    </>
  )

  return (
    <Modal isOpen={isOpen} onClose={handleClose} size="2xl" blockScrollOnMount={false}>
      <ModalOverlay />
      <ModalContent>
        <ModalHeader borderBottomWidth="1px">
          <Text fontWeight="bold">{t(titleKey)}</Text>
          <Text fontSize="sm" fontWeight="normal" color="gray.500" mt={1}>
            {t(subtitleKey)}
          </Text>
        </ModalHeader>
        <ModalCloseButton />

        <ModalBody py={6}>
          {loading ? (
            <Stack spacing={4}>
              <HStack spacing={3}>
                <Spinner size="sm" color="purple.500" />
                <Text fontSize="sm" color="gray.600">
                  {t(loadingKey)}
                </Text>
              </HStack>
              {/* 可滚动容器：复用 processBlocks 展示实时流式内容 */}
              <Box ref={streamRef} maxH="420px" overflowY="auto" pr={1}>
                {processBlocks}
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
              <Button leftIcon={<FiRefreshCw />} variant="outline" onClick={loadOptimize}>
                {t('pms.aiDecomposeRegenerate')}
              </Button>
            </Flex>
          ) : result ? (
            <VStack spacing={4} align="stretch">
              {/* Tab 切换：思考过程 / 优化结果 */}
              <HStack spacing={0} borderBottom="1px solid" borderColor="gray.200">
                <Button
                  variant="unstyled"
                  fontSize="sm"
                  fontWeight={viewTab === 'process' ? 'semibold' : 'normal'}
                  color={viewTab === 'process' ? 'purple.600' : 'gray.500'}
                  borderBottom="2px solid"
                  borderColor={viewTab === 'process' ? 'purple.500' : 'transparent'}
                  px={4}
                  py={2}
                  mb="-1px"
                  onClick={() => setViewTab('process')}
                >
                  {t('pms.aiOptimizeTabProcess')}
                </Button>
                <Button
                  variant="unstyled"
                  fontSize="sm"
                  fontWeight={viewTab === 'result' ? 'semibold' : 'normal'}
                  color={viewTab === 'result' ? 'purple.600' : 'gray.500'}
                  borderBottom="2px solid"
                  borderColor={viewTab === 'result' ? 'purple.500' : 'transparent'}
                  px={4}
                  py={2}
                  mb="-1px"
                  onClick={() => setViewTab('result')}
                >
                  {t('pms.aiOptimizeTabResult')}
                </Button>
              </HStack>

              {viewTab === 'process' ? (
                /* 对话过程回看：复用 processBlocks，只读展示 */
                <>
                <Box ref={streamRef} maxH="380px" overflowY="auto" pr={1}>
                  {processBlocks}
                </Box>
                {/* 本次对话统计：token 用量 + 耗时（仅在「对话过程」tab 显示） */}
                {(result.usage || elapsedMs !== null) && (
                  <HStack
                    spacing={{ base: 2, md: 4 }}
                    p={2}
                    bg="gray.50"
                    borderWidth="1px"
                    borderColor="gray.200"
                    borderRadius="md"
                    fontSize="xs"
                    color="gray.600"
                    flexWrap="wrap"
                  >
                    {result.usage && (
                      <>
                        <HStack spacing={1}>
                          <FiZap color="purple.500" />
                          <Text fontWeight="bold" color="purple.600">
                            {t('pms.aiTokenUsage')}
                          </Text>
                        </HStack>
                        <Text>
                          {t('pms.aiTokenPrompt')}:{' '}
                          <Text as="span" fontWeight="bold" color="blue.600">
                            {result.usage.prompt_tokens ?? 0}
                          </Text>
                        </Text>
                        <Text>
                          {t('pms.aiTokenCompletion')}:{' '}
                          <Text as="span" fontWeight="bold" color="green.600">
                            {result.usage.completion_tokens ?? 0}
                          </Text>
                        </Text>
                        <Text>
                          {t('pms.aiTokenTotal')}:{' '}
                          <Text as="span" fontWeight="bold" color="purple.600">
                            {result.usage.total_tokens ?? 0}
                          </Text>
                        </Text>
                      </>
                    )}
                    {elapsedMs !== null && (
                      <Text>
                        <HStack as="span" spacing={1} display="inline-flex" verticalAlign="middle">
                          <FiClock color="orange.500" />
                          <Text as="span" fontWeight="bold" color="orange.600">
                            {t('pms.aiDuration')}
                          </Text>
                        </HStack>
                        {' '}
                        <Text as="span" fontWeight="bold" color="orange.600">
                          {formatDuration(elapsedMs)}
                        </Text>
                      </Text>
                    )}
                  </HStack>
                )}
                </>
              ) : (
              <>
              {/* AI 的优化说明 */}
              {result.summary && (
                <Box
                  p={3}
                  bg="purple.50"
                  borderWidth="1px"
                  borderColor="purple.100"
                  borderRadius="md"
                >
                  <Text fontSize="xs" fontWeight="bold" color="purple.600" mb={1}>
                    ✨ {t('pms.aiOptimizeSummary')}
                  </Text>
                  <Text fontSize="sm" color="gray.700" whiteSpace="pre-wrap">
                    {result.summary}
                  </Text>
                </Box>
              )}

              {/* 优化后的标题 */}
              <Box>
                <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={1}>
                  {optimizeType === 'task' ? t('pms.taskTitle')
                    : optimizeType === 'sprint' ? t('pms.sprintTitle')
                      : t('pms.userStoryTitle')}{' '}
                  <Text as="span" color="purple.500" fontSize="xs">· AI</Text>
                </Text>
                <Input
                  value={result.title}
                  onChange={(e) => updateResult({ title: e.target.value })}
                  placeholder={optimizeType === 'task' ? t('pms.taskTitle')
                    : optimizeType === 'sprint' ? t('pms.sprintTitle')
                      : t('pms.userStoryTitle')}
                  size="sm"
                />
              </Box>

              {/* 优化后的描述 */}
              <Box>
                <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={1}>
                  {optimizeType === 'task' ? t('pms.taskDescription')
                    : optimizeType === 'sprint' ? t('pms.sprintDescription')
                      : t('pms.userStoryDescription')}{' '}
                  <Text as="span" color="purple.500" fontSize="xs">· AI</Text>
                </Text>
                <Textarea
                  value={result.description}
                  onChange={(e) => updateResult({ description: e.target.value })}
                  placeholder={optimizeType === 'task' ? t('pms.taskDescription')
                    : optimizeType === 'sprint' ? t('pms.sprintDescription')
                      : t('pms.userStoryDescription')}
                  size="sm"
                  rows={4}
                  resize="vertical"
                />
              </Box>

              {/* 类型相关字段：story 验收标准 / task 任务类型 / sprint 迭代目标 */}
              {isStoryResult(result) ? (
                <Box>
                  <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={1}>
                    {t('pms.acceptanceCriteria')} <Text as="span" color="purple.500" fontSize="xs">· AI</Text>
                  </Text>
                  <Textarea
                    value={result.acceptanceCriteria}
                    onChange={(e) => updateResult({ acceptanceCriteria: e.target.value } as Partial<AIOptimizeResult>)}
                    placeholder={t('pms.acceptanceCriteria')}
                    size="sm"
                    rows={3}
                    resize="vertical"
                  />
                </Box>
              ) : isTaskResult(result) ? (
                <Box>
                  <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={1}>
                    {t('pms.taskType')} <Text as="span" color="purple.500" fontSize="xs">· AI 建议</Text>
                  </Text>
                  <Select
                    value={result.taskType}
                    onChange={(e) => updateResult({ taskType: e.target.value as AIOptimizeTaskResult['taskType'] } as Partial<AIOptimizeTaskResult>)}
                    size="sm"
                  >
                    <option value="task">{t('pms.taskTypeTask')}</option>
                    <option value="feature">{t('pms.taskTypeFeature')}</option>
                    <option value="bug">{t('pms.taskTypeBug')}</option>
                    <option value="improvement">{t('pms.taskTypeImprovement')}</option>
                  </Select>
                </Box>
              ) : isSprintResult(result) ? (
                <Box>
                  <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={1}>
                    {t('pms.sprintGoal')} <Text as="span" color="purple.500" fontSize="xs">· AI</Text>
                  </Text>
                  <Textarea
                    value={result.goal}
                    onChange={(e) => updateResult({ goal: e.target.value } as Partial<AIOptimizeSprintResult>)}
                    placeholder={t('pms.sprintGoal')}
                    size="sm"
                    rows={3}
                    resize="vertical"
                  />
                </Box>
              ) : null}

              {/* 建议的优先级 + 故事点（sprint 无此字段） */}
              {!isSprintResult(result) && (
              <HStack spacing={3} align="flex-start">
                <Box flex={1}>
                  <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={1}>
                    {t('common.priority')} <Text as="span" color="purple.500" fontSize="xs">· AI 建议</Text>
                  </Text>
                  <Select
                    value={result.priority}
                    onChange={(e) => updateResult({ priority: e.target.value as AIOptimizeResult['priority'] })}
                    size="sm"
                  >
                    <option value="urgent">{t('pms.urgent')}</option>
                    <option value="high">{t('pms.high')}</option>
                    <option value="medium">{t('pms.medium')}</option>
                    <option value="low">{t('pms.low')}</option>
                  </Select>
                </Box>
                <Box flex={1}>
                  <Text fontSize="sm" fontWeight="medium" color="gray.700" mb={1}>
                    {t('pms.storyPoints')} <Text as="span" color="purple.500" fontSize="xs">· AI 建议</Text>
                  </Text>
                  <Select
                    value={String(result.storyPoints)}
                    onChange={(e) => updateResult({ storyPoints: parseInt(e.target.value) || 0 })}
                    size="sm"
                  >
                    {[0, 1, 2, 3, 5, 8, 13, 21].map((sp) => (
                      <option key={sp} value={String(sp)}>{sp}</option>
                    ))}
                  </Select>
                </Box>
              </HStack>
              )}

              {/* AI 建议快捷提示 */}
              <HStack spacing={2} color="gray.500" fontSize="xs">
                <Badge colorScheme="purple" variant="subtle">
                  {t('pms.aiOptimizeBadge')}
                </Badge>
                <Text>{t('pms.aiOptimizeEditHint')}</Text>
              </HStack>
              </>
              )}
            </VStack>
          ) : null}
        </ModalBody>

        <ModalFooter borderTopWidth="1px">
          <HStack spacing={2}>
            {!loading && !notConfigured && (
              <Button
                variant="ghost"
                leftIcon={<FiRefreshCw />}
                onClick={loadOptimize}
                isDisabled={loading}
              >
                {t('pms.aiDecomposeRegenerate')}
              </Button>
            )}
            <Button variant="ghost" onClick={handleClose}>
              {t('common.cancel')}
            </Button>
            <Button
              variant="primary"
              leftIcon={<FiCheck />}
              onClick={handleApply}
              isDisabled={loading || !result || notConfigured}
            >
              {t('pms.aiOptimizeApply')}
            </Button>
          </HStack>
        </ModalFooter>
      </ModalContent>
    </Modal>
  )
}
