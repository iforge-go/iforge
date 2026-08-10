'use client'

import {
  Box,
  HStack,
  Text,
  Icon,
  Spinner,
  Switch,
  FormControl,
  FormLabel,
} from '@chakra-ui/react'
import { useEffect, useState, useCallback, useRef, memo } from 'react'
import { FiCheckCircle } from 'react-icons/fi'
import { API_BASE, CICDJob, JobLog } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'

interface JobLogsInlineProps {
  owner: string
  repo: string
  job: CICDJob
}

// LogLine 单行日志(用 memo 包裹,避免新增日志时重渲染已有行)
const LogLine = memo(function LogLine({ log }: { log: JobLog }) {
  return (
    <Box
      display="flex"
      alignItems="flex-start"
      px={1}
      _hover={{ bg: 'myGray.800' }}
    >
      <Text
        as="span"
        color="myGray.600"
        flexShrink={0}
        userSelect="none"
        minW="48px"
        textAlign="right"
        pr={3}
        fontSize="xs"
        lineHeight={1.5}
        fontFamily="Consolas, 'SF Mono', Menlo, Monaco, 'Courier New', monospace"
      >
        {log.lineNo}
      </Text>
      <Text
        as="span"
        color={log.stream === 'stderr' ? 'red.300' : 'myGray.100'}
        whiteSpace="pre-wrap"
        wordBreak="break-all"
        flex={1}
        fontSize="xs"
        lineHeight={1.5}
        fontFamily="Consolas, 'SF Mono', Menlo, Monaco, 'Courier New', monospace"
      >
        {log.content}
      </Text>
    </Box>
  )
})

// JobLogsInline 在 pipeline 详情页内联展示 job 日志
// 展开 时挂载并加载日志,折叠时卸载(自动停止 SSE)
//
// 日志通过 SSE(Server-Sent Events)流式推送,200ms 间隔实时接收新日志行,
// 避免 2 秒轮询导致的批量加载抖动。
export function JobLogsInline({ owner, repo, job }: JobLogsInlineProps) {
  const { t } = useI18n()
  const [logs, setLogs] = useState<JobLog[]>([])
  const [loading, setLoading] = useState(true)
  const [streaming, setStreaming] = useState(true)
  // 已完成的 job 从顶部开始显示(不自动滚动);运行中的 job 自动滚到底部(实时查看最新输出)
  const jobIsActive = job.status === 'running' || job.status === 'pending'
  const [autoScroll, setAutoScroll] = useState(jobIsActive)
  const logContainerRef = useRef<HTMLDivElement>(null)
  const eventSourceRef = useRef<EventSource | null>(null)
  const autoScrollRef = useRef(autoScroll)
  autoScrollRef.current = autoScroll

  // sseKey:用于强制重新创建 EventSource(例如重试流水线时,job 从终态变为活跃态)
  // 递增 sseKey 会触发下面的 SSE useEffect 重新执行
  const [sseKey, setSseKey] = useState(0)
  const prevJobStatusRef = useRef(job.status)

  // 检测 job 从终态变为活跃态(重试场景):清空旧日志,重新连接 SSE
  useEffect(() => {
    const prevStatus = prevJobStatusRef.current
    const wasTerminal = prevStatus === 'success' || prevStatus === 'failed' || prevStatus === 'canceled'
    const isActive = job.status === 'running' || job.status === 'pending'
    if (wasTerminal && isActive) {
      // 重试场景:重置状态,触发 SSE 重连
      setLogs([])
      setLoading(true)
      setStreaming(true)
      setAutoScroll(true) // 重试后默认自动滚动到底部(查看新日志)
      setSseKey(k => k + 1)
    }
    prevJobStatusRef.current = job.status
  }, [job.status])

  // 自动滚动到日志底部:只滚动日志容器内部,不触发整个页面跳动
  const scrollToBottom = useCallback(() => {
    if (autoScrollRef.current && logContainerRef.current) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight
    }
  }, [])

  // SSE 流式接收日志(依赖 sseKey,重试时重新创建连接)
  useEffect(() => {
    // 构建 SSE URL
    const url = `${API_BASE}/repos/${owner}/${repo}/jobs/${job.id}/logs/stream?from_line=0`
    // withCredentials: true — 跨域请求需要发送 session cookie(API 在 8081,前端在 3001)
    const es = new EventSource(url, { withCredentials: true })
    eventSourceRef.current = es

    es.addEventListener('log', (e) => {
      try {
        const log: JobLog = JSON.parse(e.data)
        setLogs(prev => [...prev, log])
        // 收到新日志后,如果自动滚动开启,滚动到底部
        if (autoScrollRef.current) {
          // 用 requestAnimationFrame 避免阻塞渲染
          requestAnimationFrame(scrollToBottom)
        }
        setLoading(false)
      } catch (err) {
        console.error('Failed to parse log event:', err)
      }
    })

    es.addEventListener('done', () => {
      setStreaming(false)
      es.close()
    })

    es.onerror = () => {
      // SSE 错误(客户端断开或服务端关闭)
      setStreaming(false)
      es.close()
      setLoading(false)
    }

    return () => {
      es.close()
      eventSourceRef.current = null
    }
    // 依赖 sseKey:重试流水线时递增 sseKey,触发 EventSource 重新创建
  }, [owner, repo, job.id, sseKey, scrollToBottom])

  // 日志变化后,如果自动滚动开启,滚动到底部
  useEffect(() => {
    if (autoScroll && logs.length > 0) {
      requestAnimationFrame(scrollToBottom)
    }
  }, [logs, autoScroll, scrollToBottom])

  const handleScroll = () => {
    if (!logContainerRef.current) return
    const { scrollTop, scrollHeight, clientHeight } = logContainerRef.current
    const atBottom = scrollHeight - scrollTop - clientHeight < 50
    setAutoScroll(atBottom)
  }

  const isActive = job.status === 'running' || job.status === 'pending'

  return (
    <Box bg="myGray.900" borderTop="1px" borderColor="myGray.700">
      {/* 日志标题栏 */}
      <HStack px={4} py={2} bg="myGray.800" justify="space-between">
        <HStack spacing={2}>
          <Text color="myGray.300" fontSize="xs" fontWeight="medium">
            {t('cicd.logs')}
          </Text>
          {isActive && streaming && (
            <HStack spacing={1}>
              <Spinner size="xs" color="blue.400" />
              <Text color="blue.300" fontSize="xs">{t('cicd.logsStreaming')}</Text>
            </HStack>
          )}
          {(!isActive || !streaming) && logs.length > 0 && (
            <HStack spacing={1}>
              <Icon as={FiCheckCircle} color="green.400" w={3} h={3} />
              <Text color="green.300" fontSize="xs">{t('cicd.logsComplete')}</Text>
            </HStack>
          )}
        </HStack>
        <FormControl display="flex" alignItems="center" w="auto">
          <FormLabel htmlFor={`auto-scroll-${job.id}`} mb="0" fontSize="xs" color="myGray.300" cursor="pointer" mr={2}>
            {t('cicd.logsAutoScroll')}
          </FormLabel>
          <Switch
            id={`auto-scroll-${job.id}`}
            size="sm"
            isChecked={autoScroll}
            onChange={(e) => setAutoScroll(e.target.checked)}
            colorScheme="blue"
          />
        </FormControl>
      </HStack>
      {/* 日志内容 */}
      <Box
        ref={logContainerRef}
        onScroll={handleScroll}
        maxH="400px"
        minH="120px"
        overflowY="auto"
        p={3}
        sx={{
          '&::-webkit-scrollbar': { width: '8px' },
          '&::-webkit-scrollbar-track': { bg: 'myGray.800' },
          '&::-webkit-scrollbar-thumb': { bg: 'myGray.600', borderRadius: '4px' },
        }}
      >
        {loading ? (
          <HStack spacing={2} justify="center" h="60px">
            <Spinner size="sm" color="blue.400" />
            <Text color="myGray.400" fontSize="xs" fontFamily="Consolas, 'SF Mono', Menlo, Monaco, 'Courier New', monospace">
              {t('cicd.loadingLogs')}
            </Text>
          </HStack>
        ) : logs.length === 0 ? (
          <Text color="myGray.400" fontSize="xs" fontFamily="Consolas, 'SF Mono', Menlo, Monaco, 'Courier New', monospace">
            {t('cicd.noLogs')}
          </Text>
        ) : (
          <Box>
            {logs.map((log) => (
              <LogLine key={log.id} log={log} />
            ))}
          </Box>
        )}
      </Box>
    </Box>
  )
}
