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
  Avatar,
  IconButton,
  Button,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useEffect, useState, useCallback } from 'react'
import {
  FiGitCommit,
  FiGitBranch,
  FiRefreshCw,
  FiXCircle,
  FiClock,
  FiPlay,
  FiCalendar,
} from 'react-icons/fi'
import { api, Pipeline } from '@/lib/api'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useI18n } from '@/contexts/I18nContext'
import { useGithubToast } from '@/app/providers'
import { formatRelativeTime } from '@/lib/time'
import { pipelineStatusColor, pipelineStatusIcon, triggerEventLabel, formatDuration } from './shared'

export default function PipelinesPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const { userRole } = useRepo()
  const { t } = useI18n()
  const toast = useGithubToast()

  const [pipelines, setPipelines] = useState<Pipeline[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  const canWrite = userRole === 'owner' || userRole === 'member'

  const load = useCallback(async () => {
    try {
      const resp = await api.listPipelines(owner, repoName, 1, 50)
      setPipelines(resp.items || [])
    } catch (err) {
      console.error('Failed to load pipelines:', err)
      setPipelines([])
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [owner, repoName])

  const hasActivePipeline = pipelines.some(p => p.status === 'pending' || p.status === 'running')

  useEffect(() => {
    load()
    // 自动刷新:每 5 秒(有 pending/running 时)
    if (!hasActivePipeline) return
    const timer = setInterval(load, 5000)
    return () => clearInterval(timer)
  }, [owner, repoName, load, hasActivePipeline])

  const handleRefresh = () => {
    setRefreshing(true)
    load()
  }

  const handleCancel = async (id: number) => {
    try {
      await api.cancelPipeline(owner, repoName, id)
      toast({ title: t('cicd.cancelSuccess'), status: 'success', duration: 2000 })
      load()
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 })
    }
  }

  const handleRetry = async (id: number) => {
    try {
      await api.retryPipeline(owner, repoName, id)
      toast({ title: t('cicd.retrySuccess'), status: 'success', duration: 2000 })
      load()
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 })
    }
  }

  return (
    <VStack spacing={4} align="stretch">
      <HStack justify="space-between">
        <HStack spacing={4}>
          <Heading size="lg" color="myGray.900">
            {t('cicd.pipelines')}
          </Heading>
          <IconButton
            aria-label="refresh"
            icon={<Icon as={FiRefreshCw} />}
            size="sm"
            variant="ghost"
            onClick={handleRefresh}
            isLoading={refreshing}
          />
        </HStack>
      </HStack>

      {/* 子导航:Pipelines / Cron Schedules */}
      <HStack spacing={1} borderBottom="2px solid" borderColor="myGray.200">
        <Link href={`/${owner}/${repoName}/pipelines`}>
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<Icon as={FiGitCommit} />}
            fontWeight="semibold"
            color="primary.600"
            borderBottom="2px solid"
            borderColor="primary.600"
            mb="-2px"
            _hover={{ bg: 'myGray.50' }}
          >
            {t('cicd.pipelines')}
          </Button>
        </Link>
        <Link href={`/${owner}/${repoName}/pipelines/crons`}>
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<Icon as={FiCalendar} />}
            fontWeight="medium"
            color="myGray.600"
            borderBottom="2px solid"
            borderColor="transparent"
            mb="-2px"
            _hover={{ bg: 'myGray.50', color: 'myGray.900' }}
          >
            {t('cicd.crons')}
          </Button>
        </Link>
      </HStack>

      {loading ? (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
          <Box py={12}>
            <VStack spacing={4}>
              <Spinner size="md" color="myGray.400" />
              <Text color="myGray.500" fontSize="sm">{t('common.loading')}</Text>
            </VStack>
          </Box>
        </Box>
      ) : pipelines.length === 0 ? (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
          <Box py={12}>
            <VStack spacing={4}>
              <Icon as={FiGitCommit} w={8} h={8} color="myGray.300" />
              <Text color="myGray.500" fontSize="sm">{t('cicd.noPipelines')}</Text>
              <Text color="myGray.400" fontSize="xs" maxW="500px" textAlign="center">
                {t('cicd.noPipelinesHint')}
              </Text>
            </VStack>
          </Box>
        </Box>
      ) : (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
          <VStack spacing={0} align="stretch">
            {pipelines.map((pipeline) => {
              const StatusIcon = pipelineStatusIcon(pipeline.status)
              const color = pipelineStatusColor(pipeline.status)
              const isActive = pipeline.status === 'pending' || pipeline.status === 'running'
              const branchName = pipeline.ref.replace(/^refs\/heads\//, '').replace(/^refs\/tags\//, '')
              return (
                <Box
                  key={pipeline.id}
                  py={3}
                  px={4}
                  borderBottom="1px"
                  borderColor="myGray.100"
                  _last={{ borderBottom: 'none' }}
                  _hover={{ bg: 'myGray.50' }}
                >
                  <HStack spacing={3} align="center">
                    <Icon
                      as={StatusIcon}
                      color={color}
                      w={4}
                      h={4}
                      flexShrink={0}
                      className={isActive ? 'spin' : ''}
                    />
                    <Link href={`/${owner}/${repoName}/pipelines/${pipeline.id}`}>
                      <Text fontSize="sm" fontWeight="medium" color="myGray.900" _hover={{ color: 'primary.600' }}>
                        {pipeline.message?.split('\n')[0] || `Pipeline #${pipeline.id}`}
                      </Text>
                    </Link>
                    <HStack spacing={2} fontSize="xs" color="myGray.500">
                      <HStack spacing={1}>
                        <Icon as={FiGitBranch} w={3} h={3} />
                        <Text>{branchName}</Text>
                      </HStack>
                      <Text>·</Text>
                      <Badge colorScheme="gray" variant="subtle" fontSize="10px">
                        {triggerEventLabel(pipeline.triggerEvent, t)}
                      </Badge>
                      <Text>·</Text>
                      <Text>#{pipeline.id}</Text>
                    </HStack>
                    <HStack spacing={2} ml="auto" fontSize="xs" color="myGray.500">
                      <Link href={`/${owner}/${repoName}/commit/${pipeline.commitSha}`}>
                        <Code colorScheme="gray" fontSize="xs" _hover={{ color: 'primary.500' }}>
                          {pipeline.commitSha.substring(0, 7)}
                        </Code>
                      </Link>
                      <HStack spacing={1}>
                        <Avatar size="2xs" name={pipeline.triggeredBy} />
                        <Text>{pipeline.triggeredBy}</Text>
                      </HStack>
                      {pipeline.duration ? (
                        <HStack spacing={1}>
                          <Icon as={FiClock} w={3} h={3} />
                          <Text>{formatDuration(pipeline.duration, t)}</Text>
                        </HStack>
                      ) : null}
                      <Text>{formatRelativeTime(pipeline.createdAt, t)}</Text>
                      {canWrite && isActive && (
                        <IconButton
                          aria-label={t('cicd.cancelPipeline')}
                          icon={<Icon as={FiXCircle} />}
                          size="xs"
                          variant="ghost"
                          colorScheme="red"
                          onClick={(e) => {
                            e.preventDefault()
                            handleCancel(pipeline.id)
                          }}
                        />
                      )}
                      {canWrite && (pipeline.status === 'failed' || pipeline.status === 'canceled' || pipeline.status === 'success') && (
                        <IconButton
                          aria-label={t('cicd.retryPipeline')}
                          icon={<Icon as={FiPlay} />}
                          size="xs"
                          variant="ghost"
                          colorScheme="green"
                          onClick={(e) => {
                            e.preventDefault()
                            handleRetry(pipeline.id)
                          }}
                        />
                      )}
                    </HStack>
                  </HStack>
                </Box>
              )
            })}
          </VStack>
        </Box>
      )}

      <style jsx global>{`
        @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
        .spin { animation: spin 1.5s linear infinite; }
      `}</style>
    </VStack>
  )
}
