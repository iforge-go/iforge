'use client'

import {
  Box,
  Text,
  HStack,
  VStack,
  Icon,
  Badge,
  Spinner,
  Link,
  Code,
} from '@chakra-ui/react'
import NextLink from 'next/link'
import { useEffect, useState } from 'react'
import {
  FiCheckCircle,
  FiXCircle,
  FiClock,
  FiAlertCircle,
  FiExternalLink,
  FiChevronRight,
} from 'react-icons/fi'
import { api, CombinedCommitStatus } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { formatRelativeTime } from '@/lib/time'

interface MrChecksProps {
  owner: string
  repoName: string
  // 源分支信息（用于 cross-repo MR 时从源仓库获取 checks）
  sourceOwner: string
  sourceRepo: string
  // 源分支 HEAD commit SHA（pipeline 写 CommitStatus 时用的就是这个 SHA）
  headCommitSha: string | null
}

function statusIcon(state: string) {
  switch (state) {
    case 'success': return FiCheckCircle
    case 'failure':
    case 'error': return FiXCircle
    case 'pending': return FiClock
    default: return FiAlertCircle
  }
}

function statusColor(state: string): string {
  switch (state) {
    case 'success': return 'green.500'
    case 'failure':
    case 'error': return 'red.500'
    case 'pending': return 'yellow.500'
    default: return 'myGray.400'
  }
}

function statusLabel(state: string, t: (k: string) => string): string {
  switch (state) {
    case 'success': return t('cicd.statusSuccess')
    case 'failure': return t('cicd.statusFailed')
    case 'error': return t('cicd.statusFailed')
    case 'pending': return t('cicd.statusPending')
    default: return state
  }
}

export function MrChecks({ owner, repoName, sourceOwner, sourceRepo, headCommitSha }: MrChecksProps) {
  const { t } = useI18n()
  const [status, setStatus] = useState<CombinedCommitStatus | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!headCommitSha) {
      setLoading(false)
      return
    }
    setLoading(true)
    // cross-repo MR：checks 由源仓库的 pipeline 写入,需从源仓库读取
    const targetOwner = sourceOwner || owner
    const targetRepo = sourceRepo || repoName
    api.getCombinedCommitStatus(targetOwner, targetRepo, headCommitSha)
      .then((data) => setStatus(data || null))
      .catch(() => setStatus(null))
      .finally(() => setLoading(false))
  }, [owner, repoName, sourceOwner, sourceRepo, headCommitSha])

  if (loading) {
    return (
      <Box px={4} py={8} textAlign="center">
        <HStack spacing={2} justify="center">
          <Spinner size="sm" color="myGray.400" />
          <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
        </HStack>
      </Box>
    )
  }

  if (!headCommitSha) {
    return (
      <Box px={4} py={8} textAlign="center">
        <Icon as={FiAlertCircle} w={8} h={8} color="myGray.300" mb={2} />
        <Text fontSize="sm" color="myGray.500">{t('repo.noChecks')}</Text>
      </Box>
    )
  }

  if (!status || !status.statuses || status.statuses.length === 0) {
    return (
      <Box px={4} py={8} textAlign="center">
        <Icon as={FiChevronRight} w={8} h={8} color="myGray.300" mb={2} />
        <Text fontSize="sm" color="myGray.500">{t('repo.noChecks')}</Text>
        <Text fontSize="xs" color="myGray.400" mt={1}>
          {t('cicd.noPipelinesHint')}
        </Text>
      </Box>
    )
  }

  return (
    <Box>
      {/* 顶部汇总条 */}
      <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200" bg={
        status.state === 'success' ? 'green.50' :
        status.state === 'failure' || status.state === 'error' ? 'red.50' :
        status.state === 'pending' ? 'yellow.50' : 'myGray.50'
      }>
        <HStack spacing={2}>
          <Icon
            as={statusIcon(status.state)}
            color={statusColor(status.state)}
            w={4}
            h={4}
          />
          <Text fontSize="sm" fontWeight="medium" color={
            status.state === 'success' ? 'green.700' :
            status.state === 'failure' || status.state === 'error' ? 'red.700' :
            status.state === 'pending' ? 'yellow.700' : 'myGray.700'
          }>
            {statusLabel(status.state, t)}
          </Text>
          <Text fontSize="xs" color="myGray.500">
            · {status.statuses.length} {t('cicd.jobs')}
          </Text>
        </HStack>
      </Box>

      {/* 每个 check 一行 */}
      <VStack spacing={0} align="stretch">
        {status.statuses.map((cs, idx) => {
          const Icon_ = statusIcon(cs.state)
          const color = statusColor(cs.state)
          // 解析 context："ci/iforge:build" → "build"
          const jobName = cs.context.startsWith('ci/iforge:')
            ? cs.context.substring('ci/iforge:'.length)
            : cs.context
          return (
            <Box
              key={`${cs.context}-${idx}`}
              px={4}
              py={3}
              borderBottom="1px"
              borderColor="myGray.100"
              _last={{ borderBottom: 'none' }}
              _hover={{ bg: 'myGray.50' }}
            >
              <HStack spacing={3} align="center">
                <Icon as={Icon_} color={color} w={4} h={4} flexShrink={0} />
                <VStack align="start" spacing={0} flex={1} minW={0}>
                  <HStack spacing={2}>
                    <Text fontSize="sm" fontWeight="medium" color="myGray.900">
                      {jobName}
                    </Text>
                    <Badge colorScheme={
                      cs.state === 'success' ? 'green' :
                      cs.state === 'failure' || cs.state === 'error' ? 'red' :
                      cs.state === 'pending' ? 'yellow' : 'gray'
                    } variant="subtle" fontSize="10px">
                      {statusLabel(cs.state, t)}
                    </Badge>
                  </HStack>
                  <HStack spacing={2} fontSize="xs" color="myGray.500" mt={1}>
                    <Text>{cs.context}</Text>
                    {cs.description && (
                      <>
                        <Text>·</Text>
                        <Text>{cs.description}</Text>
                      </>
                    )}
                    <Text>·</Text>
                    <Text>{formatRelativeTime(cs.updatedDate, t)}</Text>
                  </HStack>
                </VStack>
                {cs.targetUrl && (
                  <Link as={NextLink} href={cs.targetUrl} _hover={{ textDecoration: 'none' }}>
                    <HStack spacing={1} fontSize="xs" color="primary.600" _hover={{ color: 'primary.700' }}>
                      <Text>{t('cicd.viewLogs')}</Text>
                      <Icon as={FiExternalLink} w={3} h={3} />
                    </HStack>
                  </Link>
                )}
              </HStack>
            </Box>
          )
        })}
      </VStack>
    </Box>
  )
}
