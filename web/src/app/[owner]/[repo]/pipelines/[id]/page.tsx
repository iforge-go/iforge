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
  Button,
  Code,
  Avatar,
  IconButton,
  Divider,
  Collapse,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  useDisclosure,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useEffect, useState, useCallback, useRef } from 'react'
import {
  FiArrowLeft,
  FiGitBranch,
  FiXCircle,
  FiPlay,
  FiClock,
  FiChevronDown,
  FiChevronRight,
  FiDownload,
  FiPackage,
} from 'react-icons/fi'
import { api, Pipeline, CICDJob, Artifact } from '@/lib/api'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useI18n } from '@/contexts/I18nContext'
import { useGithubToast } from '@/app/providers'
import { formatRelativeTime } from '@/lib/time'
import {
  pipelineStatusColor,
  pipelineStatusIcon,
  statusLabel,
  triggerEventIcon,
  triggerEventLabel,
  formatDuration,
  formatBytes,
} from '../shared'
import { JobLogsInline } from '../JobLogsInline'

export default function PipelineDetailPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const pipelineId = Number(params.id)
  const { userRole } = useRepo()
  const { t } = useI18n()
  const toast = useGithubToast()
  const { isOpen: yamlOpen, onToggle: toggleYaml } = useDisclosure()
  // 内联展开 job 日志:点击 job 行切换展开/折叠
  const [expandedJobIds, setExpandedJobIds] = useState<Set<number>>(new Set())
  // 追踪"自动展开"的 job(由系统展开,非用户手动展开)
  // 自动展开的 job 完成后会自动折叠;用户手动展开的 job 完成后保持展开
  // 用 ref 而非 state,因为只用于逻辑判断,不需要触发重渲染
  const autoExpandedJobIdsRef = useRef<Set<number>>(new Set())
  const toggleJobExpanded = useCallback((jobId: number) => {
    setExpandedJobIds(prev => {
      const next = new Set(prev)
      if (next.has(jobId)) {
        next.delete(jobId)
      } else {
        next.add(jobId)
      }
      return next
    })
    // 用户手动操作的 job 不再视为"自动展开",完成后不会自动折叠
    autoExpandedJobIdsRef.current.delete(jobId)
  }, [])

  const [pipeline, setPipeline] = useState<Pipeline | null>(null)
  const [jobs, setJobs] = useState<CICDJob[]>([])
  const [artifacts, setArtifacts] = useState<Artifact[]>([])
  const [loading, setLoading] = useState(true)
  // 追踪上一次的 job 状态,用于检测 job 何时变为 running 以自动展开日志
  const prevJobStatusRef = useRef<Map<number, string>>(new Map())

  const canWrite = userRole === 'owner' || userRole === 'member'

  const load = useCallback(async () => {
    try {
      const [p, jobsResp, artifactsResp] = await Promise.all([
        api.getPipeline(owner, repoName, pipelineId),
        api.listJobs(owner, repoName, pipelineId),
        api.listArtifacts(owner, repoName, pipelineId),
      ])
      setPipeline(p)
      const newJobs = jobsResp.items || []
      setJobs(newJobs)
      setArtifacts(artifactsResp.items || [])

      // 自动展开/折叠 job 日志:
      // - 首次加载:展开所有 running 的 job
      // - 后续轮询:展开新变为 running 的 job(状态从非 running 变为 running)
      // - job 从 running 变为完成状态(success/failed/canceled)时,自动展开的 job 自动折叠
      //   (用户手动展开的 job 完成后保持展开,尊重用户操作)
      const prevStatus = prevJobStatusRef.current
      const isFirstLoad = prevStatus.size === 0
      const toExpand: number[] = []
      const toCollapse: number[] = []
      for (const job of newJobs) {
        const oldStatus = prevStatus.get(job.id)
        if (job.status === 'running') {
          if (isFirstLoad || (oldStatus && oldStatus !== 'running')) {
            toExpand.push(job.id)
          }
        }
        // job 从 running 变为完成状态:需要自动折叠(仅限自动展开的 job)
        if (oldStatus === 'running' && job.status !== 'running' && job.status !== 'pending') {
          toCollapse.push(job.id)
        }
        prevStatus.set(job.id, job.status)
      }
      if (toExpand.length > 0) {
        setExpandedJobIds(prev => {
          const next = new Set(prev)
          toExpand.forEach(id => next.add(id))
          return next
        })
        toExpand.forEach(id => autoExpandedJobIdsRef.current.add(id))
      }
      if (toCollapse.length > 0) {
        // 只折叠自动展开的 job,保留用户手动展开的 job
        const collapseIds = toCollapse.filter(id => autoExpandedJobIdsRef.current.has(id))
        if (collapseIds.length > 0) {
          setExpandedJobIds(prev => {
            const next = new Set(prev)
            collapseIds.forEach(id => next.delete(id))
            return next
          })
          collapseIds.forEach(id => autoExpandedJobIdsRef.current.delete(id))
        }
      }
    } catch (err) {
      console.error('Failed to load pipeline:', err)
    } finally {
      setLoading(false)
    }
  }, [owner, repoName, pipelineId])

  // 初始加载(只在组件挂载或 owner/repo/pipelineId 变化时调用)
  useEffect(() => {
    load()
  }, [load])

  // 轮询:pipeline 活跃时定期刷新(只依赖 pipeline.status,不依赖 job 状态)
  // 避免每次 job 状态变化都重建 setInterval 和额外调用 load() 导致页面闪烁
  useEffect(() => {
    if (!pipeline) return
    const isActive = pipeline.status === 'pending' || pipeline.status === 'running'
    if (!isActive) return
    const timer = setInterval(load, 3000)
    return () => clearInterval(timer)
  }, [pipeline?.status, load])

  const handleCancel = async () => {
    try {
      await api.cancelPipeline(owner, repoName, pipelineId)
      toast({ title: t('cicd.cancelSuccess'), status: 'success', duration: 2000 })
      load()
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 })
    }
  }

  const [retrying, setRetrying] = useState(false)
  const handleRetry = async () => {
    setRetrying(true)
    try {
      await api.retryPipeline(owner, repoName, pipelineId)
      toast({ title: t('cicd.retrySuccess'), status: 'success', duration: 2000 })
      load()
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 })
    } finally {
      setRetrying(false)
    }
  }

  const handleTriggerJob = async (jobId: number) => {
    try {
      await api.triggerManualJob(owner, repoName, jobId)
      toast({ title: t('cicd.manualJobTriggered'), status: 'success', duration: 2000 })
      load()
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 })
    }
  }

  if (loading) {
    return (
      <VStack spacing={4} align="stretch">
        <HStack>
          <Spinner size="md" color="myGray.400" />
          <Text color="myGray.500" fontSize="sm">{t('common.loading')}</Text>
        </HStack>
      </VStack>
    )
  }

  if (!pipeline) {
    return (
      <VStack spacing={4} align="stretch">
        <Text color="myGray.500">{t('common.notFound')}</Text>
      </VStack>
    )
  }

  const StatusIcon = pipelineStatusIcon(pipeline.status)
  const statusColor = pipelineStatusColor(pipeline.status)
  const TriggerIcon = triggerEventIcon(pipeline.triggerEvent)
  const branchName = pipeline.ref.replace(/^refs\/heads\//, '').replace(/^refs\/tags\//, '')
  const isActive = pipeline.status === 'pending' || pipeline.status === 'running'

  // 按 stage 分组
  const stagesMap = new Map<string, CICDJob[]>()
  for (const job of jobs) {
    const arr = stagesMap.get(job.stageName) || []
    arr.push(job)
    stagesMap.set(job.stageName, arr)
  }
  const stages = Array.from(stagesMap.entries())

  return (
    <VStack spacing={4} align="stretch">
      {/* 顶部:返回按钮 + 标题 */}
      <HStack spacing={3}>
        <Link href={`/${owner}/${repoName}/pipelines`}>
          <IconButton aria-label="back" icon={<Icon as={FiArrowLeft} />} size="sm" variant="ghost" />
        </Link>
        <Heading size="lg" color="myGray.900">
          {t('cicd.pipelineId', { id: pipeline.id })}
        </Heading>
        <Icon as={StatusIcon} color={statusColor} w={5} h={5} className={isActive ? 'spin' : ''} />
        <Badge colorScheme={pipeline.status === 'success' ? 'green' : pipeline.status === 'failed' ? 'red' : pipeline.status === 'running' ? 'blue' : 'gray'} variant="subtle">
          {statusLabel(pipeline.status, t)}
        </Badge>
        <HStack spacing={2} ml="auto">
          {canWrite && isActive && (
            <Button leftIcon={<Icon as={FiXCircle} />} size="sm" variant="whiteBase" colorScheme="red" onClick={handleCancel}>
              {t('cicd.cancelPipeline')}
            </Button>
          )}
          {canWrite && (pipeline.status === 'failed' || pipeline.status === 'canceled' || pipeline.status === 'success') && (
            <Button leftIcon={<Icon as={FiPlay} />} size="sm" variant="primary" onClick={handleRetry} isLoading={retrying} loadingText={t('cicd.retryPipeline')}>
              {t('cicd.retryPipeline')}
            </Button>
          )}
        </HStack>
      </HStack>

      {/* 信息卡片 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
        <VStack spacing={2} align="stretch" px={4} py={3}>
          <HStack spacing={4} flexWrap="wrap" fontSize="sm">
            <HStack spacing={1}>
              <Icon as={TriggerIcon} color="myGray.500" />
              <Text color="myGray.600">{triggerEventLabel(pipeline.triggerEvent, t)}</Text>
            </HStack>
            <HStack spacing={1}>
              <Icon as={FiGitBranch} color="myGray.500" />
              <Link href={`/${owner}/${repoName}/commits/${branchName}`}>
                <Text color="primary.600" _hover={{ textDecoration: 'underline' }}>{branchName}</Text>
              </Link>
            </HStack>
            <HStack spacing={1}>
              <Icon as={FiClock} color="myGray.500" />
              <Text color="myGray.600">{formatRelativeTime(pipeline.createdAt, t)}</Text>
            </HStack>
            {pipeline.duration ? (
              <HStack spacing={1}>
                <Icon as={FiClock} color="myGray.500" />
                <Text color="myGray.600">{formatDuration(pipeline.duration, t)}</Text>
              </HStack>
            ) : null}
          </HStack>
          <HStack spacing={4} flexWrap="wrap" fontSize="sm">
            <HStack spacing={1}>
              <Text color="myGray.500">{t('cicd.commit')}:</Text>
              <Link href={`/${owner}/${repoName}/commit/${pipeline.commitSha}`}>
                <Code colorScheme="gray" fontSize="xs" _hover={{ color: 'primary.500' }}>
                  {pipeline.commitSha.substring(0, 7)}
                </Code>
              </Link>
              <Text color="myGray.700" ml={2}>{pipeline.message?.split('\n')[0]}</Text>
            </HStack>
            <HStack spacing={1}>
              <Text color="myGray.500">{t('cicd.triggeredBy')}:</Text>
              <Avatar size="2xs" name={pipeline.triggeredBy} />
              <Text color="myGray.700">{pipeline.triggeredBy}</Text>
            </HStack>
          </HStack>
        </VStack>
      </Box>

      {/* Jobs 按 stage 分组 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
        <Box px={4} py={2} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
          <Text fontSize="sm" fontWeight="medium" color="myGray.700">
            {t('cicd.jobs')} ({jobs.length})
          </Text>
        </Box>
        {jobs.length === 0 ? (
          <Box py={8}>
            <VStack spacing={2}>
              <Text color="myGray.500" fontSize="sm">{t('cicd.noJobs')}</Text>
            </VStack>
          </Box>
        ) : (
          <VStack spacing={0} align="stretch">
            {stages.map(([stageName, stageJobs], stageIdx) => (
              <Box key={stageName}>
                {stageIdx > 0 && <Divider borderColor="myGray.200" />}
                <Box px={4} py={2} bg="myGray.50">
                  <HStack spacing={2}>
                    <Text fontSize="xs" fontWeight="semibold" color="myGray.600" textTransform="uppercase" letterSpacing="wide">
                      {t('cicd.stage')}: {stageName}
                    </Text>
                    <Badge colorScheme="gray" fontSize="10px">{stageJobs.length}</Badge>
                  </HStack>
                </Box>
                <VStack spacing={0} align="stretch">
                  {stageJobs.map((job) => {
                    const JobStatusIcon = pipelineStatusIcon(job.status)
                    const jobColor = pipelineStatusColor(job.status)
                    const jobActive = job.status === 'running' || job.status === 'pending'
                    const expanded = expandedJobIds.has(job.id)
                    return (
                      <Box key={job.id} borderBottom="1px" borderColor="myGray.100">
                        <Box
                          px={4}
                          py={3}
                          cursor="pointer"
                          _hover={{ bg: 'myGray.50' }}
                          onClick={() => toggleJobExpanded(job.id)}
                        >
                          <HStack spacing={3} align="center">
                            <Icon
                              as={FiChevronRight}
                              color="myGray.500"
                              w={4}
                              h={4}
                              flexShrink={0}
                              sx={{
                                transition: 'transform 0.2s ease',
                                transform: expanded ? 'rotate(90deg)' : 'rotate(0deg)',
                              }}
                            />
                            <Icon as={JobStatusIcon} color={jobColor} w={4} h={4} flexShrink={0} className={jobActive ? 'spin' : ''} />
                            <Text fontSize="sm" fontWeight="medium" color="myGray.900">
                              {job.jobName}
                            </Text>
                            <Badge colorScheme={job.status === 'success' ? 'green' : job.status === 'failed' ? 'red' : job.status === 'running' ? 'blue' : 'gray'} variant="subtle" fontSize="10px">
                              {statusLabel(job.status, t)}
                            </Badge>
                            {job.environment && (
                              <Badge colorScheme="purple" variant="subtle" fontSize="10px">
                                {job.environment}
                              </Badge>
                            )}
                            {job.image && (
                              <Code colorScheme="gray" fontSize="xs">{job.image}</Code>
                            )}
                            <HStack spacing={2} ml="auto" fontSize="xs" color="myGray.500">
                              {job.startedAt && (
                                <HStack spacing={1}>
                                  <Icon as={FiClock} w={3} h={3} />
                                  <Text>
                                    {job.finishedAt
                                      ? formatDuration(new Date(job.finishedAt).getTime() - new Date(job.startedAt).getTime(), t)
                                      : formatDuration(Date.now() - new Date(job.startedAt).getTime(), t)}
                                  </Text>
                                </HStack>
                              )}
                              {job.exitCode !== null && job.exitCode !== undefined && (
                                <Text color={job.exitCode === 0 ? 'green.600' : 'red.600'}>
                                  {t('cicd.exitCode', { code: job.exitCode })}
                                </Text>
                              )}
                              {canWrite && job.when === 'manual' && (job.status === 'pending' || job.status === 'failed' || job.status === 'canceled') && (
                                <Button
                                  size="xs"
                                  variant="ghost"
                                  colorScheme="green"
                                  leftIcon={<Icon as={FiPlay} />}
                                  onClick={(e) => {
                                    e.stopPropagation()
                                    handleTriggerJob(job.id)
                                  }}
                                >
                                  {t('cicd.manualJobTrigger')}
                                </Button>
                              )}
                            </HStack>
                          </HStack>
                          {job.failureReason && (
                            <Text fontSize="xs" color="red.600" mt={1} ml={11}>
                              {t('cicd.failureReason')}: {job.failureReason}
                            </Text>
                          )}
                        </Box>
                        {expanded && (
                          <Box
                            sx={{
                              animation: 'fadeInDown 0.2s ease-out',
                              '@keyframes fadeInDown': {
                                '0%': { opacity: 0, transform: 'translateY(-8px)' },
                                '100%': { opacity: 1, transform: 'translateY(0)' },
                              },
                            }}
                          >
                            <JobLogsInline owner={owner} repo={repoName} job={job} />
                          </Box>
                        )}
                      </Box>
                    )
                  })}
                </VStack>
              </Box>
            ))}
          </VStack>
        )}
      </Box>

      {/* Artifacts 构建产物 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
        <Box px={4} py={2} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
          <HStack spacing={2}>
            <Icon as={FiPackage} color="myGray.500" w={4} h={4} />
            <Text fontSize="sm" fontWeight="medium" color="myGray.700">
              {t('cicd.artifacts')} ({artifacts.length})
            </Text>
          </HStack>
        </Box>
        {artifacts.length === 0 ? (
          <Box py={6}>
            <VStack spacing={2}>
              <Text color="myGray.500" fontSize="sm">{t('cicd.noArtifacts')}</Text>
            </VStack>
          </Box>
        ) : (
          <Table size="sm" variant="simple">
            <Thead bg="myGray.50">
              <Tr>
                <Th color="myGray.600" fontSize="xs" textTransform="none" fontWeight="semibold">{t('cicd.artifactName')}</Th>
                <Th color="myGray.600" fontSize="xs" textTransform="none" fontWeight="semibold" isNumeric>{t('cicd.artifactSize')}</Th>
                <Th color="myGray.600" fontSize="xs" textTransform="none" fontWeight="semibold">{t('cicd.artifactSourceJob')}</Th>
                <Th color="myGray.600" fontSize="xs" textTransform="none" fontWeight="semibold">{t('cicd.artifactCreated')}</Th>
                <Th color="myGray.600" fontSize="xs" textTransform="none" fontWeight="semibold" width="80px"></Th>
              </Tr>
            </Thead>
            <Tbody>
              {artifacts.map((art) => {
                const sourceJob = jobs.find(j => j.id === art.jobId)
                const expired = art.expiresAt && new Date(art.expiresAt).getTime() < Date.now()
                return (
                  <Tr key={art.id} _hover={{ bg: 'myGray.50' }}>
                    <Td fontSize="sm" color="myGray.900">
                      <HStack spacing={2}>
                        <Icon as={FiPackage} color="myGray.500" w={3} h={3} />
                        <Text>{art.name}</Text>
                        {expired && (
                          <Badge colorScheme="red" variant="subtle" fontSize="10px">{t('cicd.artifactExpired')}</Badge>
                        )}
                      </HStack>
                    </Td>
                    <Td fontSize="sm" color="myGray.600" isNumeric>{formatBytes(art.size)}</Td>
                    <Td fontSize="sm" color="myGray.600">
                      {sourceJob ? (
                        <Text>{sourceJob.jobName}</Text>
                      ) : (
                        <Text color="myGray.400">#{art.jobId}</Text>
                      )}
                    </Td>
                    <Td fontSize="sm" color="myGray.600">{formatRelativeTime(art.createdAt, t)}</Td>
                    <Td fontSize="sm">
                      <IconButton
                        aria-label={t('cicd.artifactDownload')}
                        icon={<Icon as={FiDownload} />}
                        size="xs"
                        variant="ghost"
                        colorScheme="blue"
                        isDisabled={!!expired}
                        onClick={() => {
                          api.downloadArtifact(owner, repoName, art.id, art.name)
                        }}
                      />
                    </Td>
                  </Tr>
                )
              })}
            </Tbody>
          </Table>
        )}
      </Box>

      {/* YAML 配置(折叠) */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
        <Box px={4} py={2} borderBottom="1px" borderColor="myGray.200" bg="myGray.50" cursor="pointer" onClick={toggleYaml}>
          <HStack spacing={2}>
            <Icon as={yamlOpen ? FiChevronDown : FiChevronRight} color="myGray.500" />
            <Text fontSize="sm" fontWeight="medium" color="myGray.700">
              {t('cicd.yamlConfig')}
            </Text>
          </HStack>
        </Box>
        <Collapse in={yamlOpen}>
          <Box px={4} py={3} bg="myGray.50" overflowX="auto">
            <Code display="block" whiteSpace="pre" p={3} bg="myGray.900" color="myGray.50" fontSize="xs" borderRadius="md">
              {pipeline.yamlConfig}
            </Code>
          </Box>
        </Collapse>
      </Box>

      <style jsx global>{`
        @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
        .spin { animation: spin 1.5s linear infinite; }
      `}</style>
    </VStack>
  )
}
