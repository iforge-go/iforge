'use client'

import {
  Box,
  Heading,
  Text,
  VStack,
  HStack,
  Icon,
  Badge,
  Spinner,
  Code,
  IconButton,
  Switch,
  FormControl,
  FormLabel,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useEffect, useState, useCallback, useRef } from 'react'
import {
  FiArrowLeft,
  FiClock,
  FiCheckCircle,
} from 'react-icons/fi'
import { api, CICDJob, JobLog } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { formatRelativeTime } from '@/lib/time'
import {
  pipelineStatusColor,
  pipelineStatusIcon,
  statusLabel,
  formatDuration,
} from '../../pipelines/shared'

const POLL_INTERVAL = 2000

export default function JobLogsPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const jobId = Number(params.jobId)
  const { t } = useI18n()

  const [job, setJob] = useState<CICDJob | null>(null)
  const [logs, setLogs] = useState<JobLog[]>([])
  const [loading, setLoading] = useState(true)
  const [autoScroll, setAutoScroll] = useState(true)
  const logEndRef = useRef<HTMLDivElement>(null)
  const logContainerRef = useRef<HTMLDivElement>(null)
  const lastLineNoRef = useRef<number>(0)
  // 防止初始加载与轮询并发执行导致重复追加相同日志
  const loadingLogsRef = useRef<boolean>(false)

  const loadJob = useCallback(async () => {
    try {
      const j = await api.getJob(owner, repoName, jobId)
      setJob(j)
    } catch (err) {
      console.error('Failed to load job:', err)
    }
  }, [owner, repoName, jobId])

  const loadLogs = useCallback(async () => {
    // 竞态保护:上一次加载未完成时跳过,避免重复获取同一批日志
    if (loadingLogsRef.current) return
    loadingLogsRef.current = true
    try {
      // 增量加载:从最后一条日志的 lineNo + 1 开始
      const fromLine = lastLineNoRef.current
      const resp = await api.getJobLogs(owner, repoName, jobId, fromLine, 5000)
      const newItems = resp.items
      if (newItems && newItems.length > 0) {
        // 按 id 去重,防止极端情况下重复追加
        setLogs(prev => {
          const existingIds = new Set(prev.map(l => l.id))
          const deduped = newItems.filter(l => !existingIds.has(l.id))
          return deduped.length > 0 ? [...prev, ...deduped] : prev
        })
        lastLineNoRef.current = newItems[newItems.length - 1].lineNo + 1
      }
    } catch (err) {
      console.error('Failed to load logs:', err)
    } finally {
      loadingLogsRef.current = false
      setLoading(false)
    }
  }, [owner, repoName, jobId])

  // 初始加载
  useEffect(() => {
    loadJob()
    loadLogs()
  }, [owner, repoName, jobId])

  // 轮询:当 job 还在 pending/running 时持续刷新
  useEffect(() => {
    if (!job) return
    const isActive = job.status === 'pending' || job.status === 'running'
    if (!isActive) return
    const timer = setInterval(() => {
      loadJob()
      loadLogs()
    }, POLL_INTERVAL)
    return () => clearInterval(timer)
  }, [job?.status, loadJob, loadLogs])

  // 自动滚动到底部
  useEffect(() => {
    if (autoScroll && logEndRef.current) {
      logEndRef.current.scrollIntoView({ behavior: 'smooth', block: 'end' })
    }
  }, [logs, autoScroll])

  const handleScroll = () => {
    if (!logContainerRef.current) return
    const { scrollTop, scrollHeight, clientHeight } = logContainerRef.current
    const atBottom = scrollHeight - scrollTop - clientHeight < 50
    setAutoScroll(atBottom)
  }

  if (loading && !job) {
    return (
      <VStack spacing={4} align="stretch">
        <HStack>
          <Spinner size="md" color="myGray.400" />
          <Text color="myGray.500" fontSize="sm">{t('cicd.loadingLogs')}</Text>
        </HStack>
      </VStack>
    )
  }

  if (!job) {
    return (
      <VStack spacing={4} align="stretch">
        <Text color="myGray.500">{t('common.notFound')}</Text>
      </VStack>
    )
  }

  const StatusIcon = pipelineStatusIcon(job.status)
  const statusColor = pipelineStatusColor(job.status)
  const isActive = job.status === 'running' || job.status === 'pending'
  const jobDuration = job.startedAt
    ? (job.finishedAt
        ? new Date(job.finishedAt).getTime() - new Date(job.startedAt).getTime()
        : Date.now() - new Date(job.startedAt).getTime())
    : null

  return (
    <VStack spacing={4} align="stretch">
      {/* 顶部:返回 + 标题 + 状态 */}
      <HStack spacing={3}>
        <Link href={`/${owner}/${repoName}/pipelines/${job.pipelineId}`}>
          <IconButton aria-label="back" icon={<Icon as={FiArrowLeft} />} size="sm" variant="ghost" />
        </Link>
        <Heading size="lg" color="myGray.900">
          {job.jobName}
        </Heading>
        <Icon as={StatusIcon} color={statusColor} w={5} h={5} className={isActive ? 'spin' : ''} />
        <Badge colorScheme={job.status === 'success' ? 'green' : job.status === 'failed' ? 'red' : job.status === 'running' ? 'blue' : 'gray'} variant="subtle">
          {statusLabel(job.status, t)}
        </Badge>
      </HStack>

      {/* Job 元信息卡片 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
        <VStack spacing={2} align="stretch" px={4} py={3} fontSize="sm">
          <HStack spacing={4} flexWrap="wrap">
            <HStack spacing={1}>
              <Text color="myGray.500">{t('cicd.stage')}:</Text>
              <Text color="myGray.700" fontWeight="medium">{job.stageName}</Text>
            </HStack>
            <HStack spacing={1}>
              <Text color="myGray.500">{t('cicd.commit')}:</Text>
              <Link href={`/${owner}/${repoName}/pipelines/${job.pipelineId}`}>
                <Code colorScheme="gray" fontSize="xs" _hover={{ color: 'primary.500' }}>
                  #{job.pipelineId}
                </Code>
              </Link>
            </HStack>
            {job.image && (
              <HStack spacing={1}>
                <Text color="myGray.500">image:</Text>
                <Code colorScheme="gray" fontSize="xs">{job.image}</Code>
              </HStack>
            )}
            {job.environment && (
              <HStack spacing={1}>
                <Text color="myGray.500">{t('cicd.environment')}:</Text>
                <Badge colorScheme="purple" variant="subtle" fontSize="10px">{job.environment}</Badge>
              </HStack>
            )}
          </HStack>
          <HStack spacing={4} flexWrap="wrap">
            {job.startedAt && (
              <HStack spacing={1}>
                <Icon as={FiClock} color="myGray.500" />
                <Text color="myGray.600">
                  {jobDuration !== null ? formatDuration(jobDuration, t) : null}
                </Text>
                <Text color="myGray.500" fontSize="xs">· {formatRelativeTime(job.startedAt, t)}</Text>
              </HStack>
            )}
            {job.exitCode !== null && job.exitCode !== undefined && (
              <Text color={job.exitCode === 0 ? 'green.600' : 'red.600'} fontSize="xs">
                {t('cicd.exitCode', { code: job.exitCode })}
              </Text>
            )}
            {job.failureReason && (
              <Text color="red.600" fontSize="xs">
                {t('cicd.failureReason')}: {job.failureReason}
              </Text>
            )}
          </HStack>
        </VStack>
      </Box>

      {/* 日志区域 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="myGray.900" borderColor="myGray.700">
        <HStack px={3} py={2} bg="myGray.800" borderBottom="1px" borderColor="myGray.700" justify="space-between">
          <HStack spacing={2}>
            <Text color="myGray.300" fontSize="sm" fontWeight="medium">
              {t('cicd.logs')}
            </Text>
            {isActive && (
              <HStack spacing={1}>
                <Spinner size="xs" color="blue.400" />
                <Text color="blue.300" fontSize="xs">{t('cicd.logsStreaming')}</Text>
              </HStack>
            )}
            {!isActive && logs.length > 0 && (
              <HStack spacing={1}>
                <Icon as={FiCheckCircle} color="green.400" w={3} h={3} />
                <Text color="green.300" fontSize="xs">{t('cicd.logsComplete')}</Text>
              </HStack>
            )}
          </HStack>
          <FormControl display="flex" alignItems="center" w="auto">
            <FormLabel htmlFor="auto-scroll" mb="0" fontSize="xs" color="myGray.300" cursor="pointer">
              {t('cicd.logsAutoScroll')}
            </FormLabel>
            <Switch id="auto-scroll" size="sm" isChecked={autoScroll} onChange={(e) => setAutoScroll(e.target.checked)} colorScheme="blue" />
          </FormControl>
        </HStack>
        <Box
          ref={logContainerRef}
          onScroll={handleScroll}
          maxH="600px"
          overflowY="auto"
          p={3}
          fontFamily="Consolas, 'SF Mono', Menlo, Monaco, 'Courier New', monospace"
          fontSize="xs"
          lineHeight={1.5}
          sx={{
            '&::-webkit-scrollbar': { width: '8px' },
            '&::-webkit-scrollbar-track': { bg: 'myGray.800' },
            '&::-webkit-scrollbar-thumb': { bg: 'myGray.600', borderRadius: '4px' },
          }}
        >
          {logs.length === 0 ? (
            <Text color="myGray.400">{t('cicd.noLogs')}</Text>
          ) : (
            <VStack spacing={0} align="stretch">
              {logs.map((log) => (
                <HStack key={log.id} spacing={3} align="start" _hover={{ bg: 'myGray.800' }} px={1}>
                  <Text color="myGray.600" flexShrink={0} userSelect="none" minW="40px" textAlign="right">
                    {log.lineNo}
                  </Text>
                  <Text
                    color={log.stream === 'stderr' ? 'red.300' : 'myGray.100'}
                    whiteSpace="pre-wrap"
                    wordBreak="break-all"
                    flex={1}
                  >
                    {log.content}
                  </Text>
                </HStack>
              ))}
            </VStack>
          )}
          <div ref={logEndRef} />
        </Box>
      </Box>

      <style jsx global>{`
        @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
        .spin { animation: spin 1.5s linear infinite; }
      `}</style>
    </VStack>
  )
}
