'use client'

import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import {
  Box,
  Card,
  CardBody,
  CardHeader,
  Flex,
  Heading,
  HStack,
  Spinner,
  Stack,
  Text,
  Badge,
  VStack,
} from '@chakra-ui/react'
import { FiBarChart2, FiRepeat, FiArrowRight } from 'react-icons/fi'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import VelocityChart from '@/components/VelocityChart'
import { useProject } from '../ProjectContext'
import type { VelocityDataPoint } from '@/lib/types'

// 项目报告页（对齐 Jira Reports tab）
// 上方：速度图（跨 Sprint 承诺 vs 完成故事点）
// 下方：已关闭 Sprint 报告列表（点击进入 Sprint 详情页报告 tab）
export default function ReportsPage() {
  const params = useParams()
  const projectSlug = params.slug as string
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()
  const { project, loading: projectLoading } = useProject()
  const [velocityData, setVelocityData] = useState<VelocityDataPoint[]>([])
  const [sprints, setSprints] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!user || !projectSlug) return
    Promise.all([
      api.getVelocity(projectSlug).then(setVelocityData).catch(() => setVelocityData([])),
      api.getSprints(projectSlug).then(setSprints).catch(() => setSprints([])),
    ]).finally(() => setLoading(false))
  }, [user, projectSlug])

  if (authLoading) return null

  if (!user) {
    return (
      <Box py={20}>
        <Text textAlign="center">{t('common.pleaseLogin')}</Text>
      </Box>
    )
  }

  if (projectLoading || loading) {
    return (
      <Flex justify="center" align="center" minH="60vh">
        <Spinner size="xl" color="blue.500" />
      </Flex>
    )
  }

  if (!project) return null

  // 已关闭 Sprint 按 completedDate DESC 排列
  const closedSprints = sprints
    .filter((s) => s.status === 'closed')
    .sort((a, b) => {
      const da = a.completedDate ? new Date(a.completedDate).getTime() : 0
      const db = b.completedDate ? new Date(b.completedDate).getTime() : 0
      return db - da
    })

  // 速度图数据 map（sprintSlug → dataPoint）用于列表展示承诺/完成点数
  const velocityMap = new Map(velocityData.map((v) => [v.sprintSlug, v]))

  return (
    <Stack spacing={6}>
      {/* 速度图 */}
      <Card bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" overflow="hidden">
        <CardHeader>
          <HStack spacing={2}>
            <FiBarChart2 color="#3182ce" />
            <Heading size="sm">{t('pms.velocityChart')}</Heading>
          </HStack>
        </CardHeader>
        <CardBody>
          <VelocityChart data={velocityData} />
        </CardBody>
      </Card>

      {/* Sprint 报告列表 */}
      <Card bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" overflow="hidden">
        <CardHeader>
          <HStack spacing={2}>
            <FiRepeat color="#3182ce" />
            <Heading size="sm">{t('pms.sprintReports')}</Heading>
          </HStack>
        </CardHeader>
        <CardBody>
          {closedSprints.length === 0 ? (
            <Text color="gray.400" fontSize="sm" textAlign="center" py={8}>
              {t('pms.noVelocityData')}
            </Text>
          ) : (
            <VStack spacing={3} align="stretch">
              {closedSprints.map((sprint) => {
                const vd = velocityMap.get(sprint.slug)
                return (
                  <Flex
                    key={sprint.slug}
                    align="center"
                    justify="space-between"
                    p={3}
                    borderRadius="md"
                    border="1px solid #EDF2F7"
                    _hover={{ bg: 'gray.50', borderColor: 'blue.200' }}
                    cursor="pointer"
                    as={Link}
                    href={`/projects/${projectSlug}/sprint/${sprint.slug}`}
                  >
                    <HStack spacing={3} flex={1}>
                      <Box w="8px" h="8px" borderRadius="full" bg="gray.400" flexShrink={0} />
                      <VStack align="start" spacing={0} flex={1}>
                        <Text fontSize="sm" fontWeight="medium" color="gray.800" noOfLines={1}>
                          {sprint.title}
                        </Text>
                        <Text fontSize="xs" color="gray.500">
                          {sprint.completedDate
                            ? new Date(sprint.completedDate).toLocaleDateString('zh-CN', { year: 'numeric', month: 'numeric', day: 'numeric' })
                            : t('pms.noDatesSet')}
                        </Text>
                      </VStack>
                    </HStack>
                    <HStack spacing={3}>
                      {vd && (
                        <HStack spacing={2} fontSize="xs">
                          <Badge colorScheme="gray" variant="subtle">
                            {t('pms.committedPoints')}: {vd.committedPoints}
                          </Badge>
                          <Badge colorScheme="green" variant="subtle">
                            {t('common.statusDone')}: {vd.completedPoints}
                          </Badge>
                        </HStack>
                      )}
                      <FiArrowRight color="#94a3b8" />
                    </HStack>
                  </Flex>
                )
              })}
            </VStack>
          )}
        </CardBody>
      </Card>
    </Stack>
  )
}
