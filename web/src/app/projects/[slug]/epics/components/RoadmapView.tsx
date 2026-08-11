'use client'

import { useState, useMemo } from 'react'
import { useRouter } from 'next/navigation'
import {
  Box,
  Button,
  Flex,
  Heading,
  HStack,
  Stack,
  Text,
  Tooltip,
} from '@chakra-ui/react'
import { FiCalendar } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { Epic } from '@/lib/types'

type Scale = 'week' | 'month' | 'quarter'

interface RoadmapViewProps {
  projectSlug: string
  epics: Epic[]
}

// 缩放级别对应的时间轴范围(单位:天,以今天为中心向前后扩展)
const SCALE_RANGE: Record<Scale, { before: number; after: number }> = {
  week: { before: 14, after: 42 },    // 共 8 周(前 2 后 6,让进行中的 Epic 靠左)
  month: { before: 30, after: 90 },   // 共 4 个月
  quarter: { before: 45, after: 135 }, // 共 6 个月(约 2 季度)
}

// 时间轴头部分段数(用于显示日期标签)
const SCALE_SEGMENTS: Record<Scale, number> = {
  week: 8,    // 8 个周标签
  month: 4,   // 4 个月标签
  quarter: 6, // 6 个月标签
}

// 根据缩放级别生成头部时间标签
function generateAxisLabels(axisStart: Date, axisEnd: Date, scale: Scale): Date[] {
  const segments = SCALE_SEGMENTS[scale]
  const totalMs = axisEnd.getTime() - axisStart.getTime()
  const labels: Date[] = []
  for (let i = 0; i < segments; i++) {
    const t = new Date(axisStart.getTime() + (totalMs * i) / segments)
    labels.push(t)
  }
  return labels
}

// 格式化头部日期标签
function formatAxisLabel(date: Date, scale: Scale): string {
  if (scale === 'week') {
    // 周视图:显示 MM-DD
    const m = String(date.getMonth() + 1).padStart(2, '0')
    const d = String(date.getDate()).padStart(2, '0')
    return `${m}-${d}`
  }
  // 月/季度视图:显示 YYYY-MM
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  return `${y}-${m}`
}

// 状态颜色映射(与 EpicsTab 保持一致)
const STATUS_COLOR: Record<string, string> = {
  open: 'gray.400',
  in_progress: 'blue.500',
  done: 'green.500',
  closed: 'green.600',
}

export default function RoadmapView({ projectSlug, epics }: RoadmapViewProps) {
  const { t } = useI18n()
  const router = useRouter()
  const [scale, setScale] = useState<Scale>('week')

  // 时间轴范围(基于今天)
  const { axisStart, axisEnd, totalMs } = useMemo(() => {
    const range = SCALE_RANGE[scale]
    const now = new Date()
    const start = new Date(now)
    start.setDate(start.getDate() - range.before)
    const end = new Date(now)
    end.setDate(end.getDate() + range.after)
    return { axisStart: start, axisEnd: end, totalMs: end.getTime() - start.getTime() }
  }, [scale])

  // 头部标签
  const axisLabels = useMemo(() => generateAxisLabels(axisStart, axisEnd, scale), [axisStart, axisEnd, scale])

  // 今天在时间轴上的百分比位置
  const todayPercent = useMemo(() => {
    const now = Date.now()
    return Math.max(0, Math.min(100, ((now - axisStart.getTime()) / totalMs) * 100))
  }, [axisStart, totalMs])

  // 分离有日期 / 无日期的 Epic
  const { scheduled, unscheduled } = useMemo(() => {
    const scheduled = epics.filter(e => e.startDate || e.targetDate)
    const unscheduled = epics.filter(e => !e.startDate && !e.targetDate)
    return { scheduled, unscheduled }
  }, [epics])

  // 计算单个 Epic 的条形定位
  const getBarPosition = (epic: Epic) => {
    const startSrc = epic.startDate ? new Date(epic.startDate) : (epic.targetDate ? new Date(new Date(epic.targetDate).getTime() - 14 * 86400000) : new Date())
    const endSrc = epic.targetDate ? new Date(epic.targetDate) : (epic.startDate ? new Date(new Date(epic.startDate).getTime() + 30 * 86400000) : new Date())
    // 限制在时间轴范围内
    const startTs = Math.max(startSrc.getTime(), axisStart.getTime())
    const endTs = Math.min(endSrc.getTime(), axisEnd.getTime())
    const leftPercent = ((startTs - axisStart.getTime()) / totalMs) * 100
    const widthPercent = Math.max(2, ((endTs - startTs) / totalMs) * 100) // 最小 2% 宽度
    return { leftPercent, widthPercent }
  }

  // 头部缩放按钮组
  const scaleButtons: { key: Scale; label: string }[] = [
    { key: 'week', label: t('pms.roadmapScaleWeek') },
    { key: 'month', label: t('pms.roadmapScaleMonth') },
    { key: 'quarter', label: t('pms.roadmapScaleQuarter') },
  ]

  return (
    <Stack spacing={6}>
      {/* 标题 + 缩放切换 */}
      <Flex justify="space-between" align="center">
        <Heading size="lg">{t('pms.epicRoadmap')}</Heading>
        <HStack spacing={2}>
          {scaleButtons.map(btn => (
            <Button
              key={btn.key}
              size="sm"
              variant={scale === btn.key ? 'solid' : 'outline'}
              colorScheme="purple"
              onClick={() => setScale(btn.key)}
            >
              {btn.label}
            </Button>
          ))}
        </HStack>
      </Flex>

      {epics.length === 0 ? (
        <Box bg="white" borderRadius="lg" border="1px solid" borderColor="myGray.200" p={10} textAlign="center">
          <FiCalendar size={40} style={{ margin: '0 auto', color: '#CBD5E0' }} />
          <Text mt={4} color="gray.500">{t('pms.roadmapNoEpics')}</Text>
        </Box>
      ) : (
        <>
          {/* Roadmap 主体 */}
          <Box bg="white" borderRadius="lg" border="1px solid" borderColor="myGray.200" overflow="hidden">
            {/* 时间轴头部 */}
            <Flex borderBottom="1px solid" borderColor="myGray.200" bg="gray.50">
              {/* 左上角空白(对齐标题列) */}
              <Box w="220px" flexShrink={0} borderRight="1px solid" borderColor="myGray.200" px={3} py={2}>
                <Text fontSize="xs" fontWeight="medium" color="gray.600">{t('pms.epicTitle')}</Text>
              </Box>
              {/* 时间轴标签 */}
              <Box position="relative" flex={1} h="36px">
                <Flex h="100%">
                  {axisLabels.map((label, i) => (
                    <Box
                      key={i}
                      flex={1}
                      px={2}
                      py={2}
                      borderRight="1px solid"
                      borderColor="myGray.100"
                      _last={{ borderRight: 'none' }}
                    >
                      <Text fontSize="xs" color="gray.600">{formatAxisLabel(label, scale)}</Text>
                    </Box>
                  ))}
                </Flex>
              </Box>
            </Flex>

            {/* Epic 行列表 */}
            <Box position="relative">
              {scheduled.length === 0 ? (
                <Box px={4} py={6} textAlign="center">
                  <Text color="gray.500" fontSize="sm">{t('pms.roadmapUnscheduled')}: {epics.length}</Text>
                </Box>
              ) : (
                scheduled.map(epic => {
                  const { leftPercent, widthPercent } = getBarPosition(epic)
                  const statusColor = STATUS_COLOR[epic.status] || 'gray.400'
                  const tooltipText = `${epic.title}
${epic.startDate ? new Date(epic.startDate).toLocaleDateString() : '?'} → ${epic.targetDate ? new Date(epic.targetDate).toLocaleDateString() : '?'}
${t('pms.progress')}: ${epic.progressPercent || 0}% (${epic.doneStories || 0}/${epic.totalStories || 0} ${t('pms.stories').toLowerCase()})`
                  return (
                    <Flex
                      key={epic.slug}
                      borderBottom="1px solid"
                      borderColor="myGray.100"
                      _hover={{ bg: 'gray.50' }}
                      h="48px"
                    >
                      {/* 左侧标题列 */}
                      <Box
                        w="220px"
                        flexShrink={0}
                        borderRight="1px solid"
                        borderColor="myGray.200"
                        px={3}
                        py={2}
                        cursor="pointer"
                        onClick={() => router.push(`/projects/${projectSlug}/epics/${epic.slug}`)}
                      >
                        <Text fontSize="sm" fontWeight="medium" color="blue.600" noOfLines={1} _hover={{ color: 'blue.800' }}>
                          {epic.title}
                        </Text>
                        <HStack spacing={1} mt={0.5}>
                          <Box w="6px" h="6px" borderRadius="full" bg={statusColor} />
                          <Text fontSize="2xs" color="gray.500" noOfLines={1}>
                            {epic.ownerName || '—'}
                          </Text>
                        </HStack>
                      </Box>
                      {/* 右侧时间轴条形 */}
                      <Box position="relative" flex={1} py={2}>
                        {/* 网格竖线(对齐头部标签) */}
                        <Flex position="absolute" top={0} left={0} right={0} bottom={0} pointerEvents="none">
                          {axisLabels.map((_, i) => (
                            <Box
                              key={i}
                              flex={1}
                              borderRight="1px solid"
                              borderColor="myGray.50"
                              _last={{ borderRight: 'none' }}
                            />
                          ))}
                        </Flex>
                        {/* 今天竖线 */}
                        {todayPercent > 0 && todayPercent < 100 && (
                          <Box
                            position="absolute"
                            top={0}
                            bottom={0}
                            left={`${todayPercent}%`}
                            borderLeft="2px dashed"
                            borderColor="red.400"
                            zIndex={1}
                            pointerEvents="none"
                          >
                            <Text
                              position="absolute"
                              top="-18px"
                              left="4px"
                              fontSize="2xs"
                              color="red.500"
                              fontWeight="medium"
                              whiteSpace="nowrap"
                            >
                              {t('pms.roadmapToday')}
                            </Text>
                          </Box>
                        )}
                        {/* Epic 条形 */}
                        <Tooltip label={tooltipText} placement="top" hasArrow>
                          <Box
                            position="absolute"
                            top="12px"
                            height="24px"
                            left={`${leftPercent}%`}
                            width={`${widthPercent}%`}
                            bg="purple.100"
                            borderRadius="full"
                            overflow="hidden"
                            cursor="pointer"
                            borderWidth="1px"
                            borderColor="purple.300"
                            _hover={{ opacity: 0.85 }}
                            onClick={() => router.push(`/projects/${projectSlug}/epics/${epic.slug}`)}
                          >
                            {/* 进度叠加 */}
                            <Box
                              position="absolute"
                              top={0}
                              left={0}
                              bottom={0}
                              width={`${epic.progressPercent || 0}%`}
                              bg="purple.500"
                              transition="width 0.2s"
                            />
                            {/* 条形上的标题(如果宽度足够) */}
                            {widthPercent > 10 && (
                              <Text
                                position="relative"
                                zIndex={1}
                                fontSize="2xs"
                                color={epic.progressPercent > 50 ? 'white' : 'purple.700'}
                                fontWeight="medium"
                                px={2}
                                lineHeight="24px"
                                noOfLines={1}
                              >
                                {epic.title}
                              </Text>
                            )}
                          </Box>
                        </Tooltip>
                      </Box>
                    </Flex>
                  )
                })
              )}
            </Box>
          </Box>

          {/* 未排期区 */}
          {unscheduled.length > 0 && (
            <Box bg="white" borderRadius="lg" border="1px solid" borderColor="myGray.200" overflow="hidden">
              <Box px={4} py={2} borderBottom="1px solid" borderColor="myGray.200" bg="gray.50">
                <Text fontSize="sm" fontWeight="medium" color="gray.700">
                  {t('pms.roadmapUnscheduled')} ({unscheduled.length})
                </Text>
              </Box>
              <Flex p={4} flexWrap="wrap" gap={2}>
                {unscheduled.map(epic => (
                  <Box
                    key={epic.slug}
                    bg="gray.50"
                    borderRadius="md"
                    border="1px solid"
                    borderColor="myGray.200"
                    px={3}
                    py={2}
                    cursor="pointer"
                    _hover={{ bg: 'gray.100', borderColor: 'purple.300' }}
                    onClick={() => router.push(`/projects/${projectSlug}/epics/${epic.slug}`)}
                  >
                    <HStack spacing={2}>
                      <Box w="6px" h="6px" borderRadius="full" bg={STATUS_COLOR[epic.status] || 'gray.400'} />
                      <Text fontSize="sm" fontWeight="medium" color="blue.600" noOfLines={1}>
                        {epic.title}
                      </Text>
                      <Text fontSize="2xs" color="gray.500">
                        {epic.progressPercent || 0}%
                      </Text>
                    </HStack>
                  </Box>
                ))}
              </Flex>
            </Box>
          )}
        </>
      )}
    </Stack>
  )
}
