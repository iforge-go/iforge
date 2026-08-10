'use client'

import { Box, Text, HStack, VStack } from '@chakra-ui/react'
import { useEffect, useState, useRef } from 'react'
import { api, ContributionDay } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useSiteSettings } from '@/contexts/SiteSettingsContext'

interface ContributionGraphProps {
  username: string
}

function getColor(count: number, max: number): string {
  if (count === 0) return 'myGray.100'
  const ratio = count / Math.max(max, 1)
  if (ratio < 0.25) return 'primary.200'
  if (ratio < 0.5) return 'primary.400'
  if (ratio < 0.75) return 'primary.600'
  return 'primary.800'
}

const NUM_WEEKS = 53

export function ContributionGraph({ username }: ContributionGraphProps) {
  const { t } = useI18n()
  const { settings } = useSiteSettings()
  const [contributions, setContributions] = useState<ContributionDay[]>([])
  const [loading, setLoading] = useState(true)
  const wrapperRef = useRef<HTMLDivElement>(null)
  const [cellSize, setCellSize] = useState(13)

  useEffect(() => {
    api.getUserContributions(username)
      .then((data) => setContributions(data || []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [username])

  useEffect(() => {
    if (!wrapperRef.current) return
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const w = entry.contentRect.width
        const labelW = 22
        const gap = 3
        const available = w - labelW
        const size = Math.floor((available - (NUM_WEEKS - 1) * gap) / NUM_WEEKS)
        setCellSize(Math.max(Math.min(size, 14), 8))
      }
    })
    observer.observe(wrapperRef.current)
    return () => observer.disconnect()
  }, [])

  if (loading) return null

  const countMap = new Map<string, number>()
  contributions.forEach(c => countMap.set(c.date, c.count))

  const totalContributions = contributions.reduce((sum, c) => sum + c.count, 0)

  const tz = settings?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone

  // Get today's date string in the configured timezone
  const todayStr = new Intl.DateTimeFormat('en-CA', { timeZone: tz, year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date())

  // Start from 364 days before today, aligned to Sunday
  const todayParts = new Intl.DateTimeFormat('en-US', {
    timeZone: tz, year: 'numeric', month: 'numeric', day: 'numeric',
  }).formatToParts(new Date())
  const getPart = (type: string) => todayParts.find(p => p.type === type)?.value
  const todayY = Number(getPart('year'))
  const todayM = Number(getPart('month')) - 1
  const todayD = Number(getPart('day'))

  const startDate = new Date(Date.UTC(todayY, todayM, todayD - 364))
  startDate.setUTCDate(startDate.getUTCDate() - startDate.getUTCDay())

  const weeks: { date: Date; count: number }[][] = []
  const maxCount = Math.max(...contributions.map(c => c.count), 1)

  const fmtDate = (d: Date) => {
    return new Intl.DateTimeFormat('en-CA', { timeZone: tz, year: 'numeric', month: '2-digit', day: '2-digit' }).format(d)
  }

  for (let week = 0; week < 53; week++) {
    const days: { date: Date; count: number }[] = []
    for (let day = 0; day < 7; day++) {
      const date = new Date(startDate.getTime() + (week * 7 + day) * 86400000)
      const dateStr = fmtDate(date)
      if (dateStr > todayStr) break
      days.push({ date, count: countMap.get(dateStr) || 0 })
    }
    if (days.length > 0) weeks.push(days)
  }

  const dayLabels = ['', t('contribution.mon'), '', t('contribution.wed'), '', t('contribution.fri'), '']

  const monthNames = [t('contribution.jan'), t('contribution.feb'), t('contribution.mar'), t('contribution.apr'), t('contribution.may'), t('contribution.jun'), t('contribution.jul'), t('contribution.aug'), t('contribution.sep'), t('contribution.oct'), t('contribution.nov'), t('contribution.dec')]
  const monthPositions: { label: string; weekIdx: number }[] = []
  let lastMonth = -1

  weeks.forEach((week, weekIdx) => {
    if (week.length > 0) {
      const month = week[0].date.getMonth()
      if (month !== lastMonth) {
        monthPositions.push({ label: monthNames[month], weekIdx })
        lastMonth = month
      }
    }
  })

  const gap = 3
  const labelW = 22

  return (
    <VStack spacing={2} align="stretch">
      <Text fontSize="sm" fontWeight="medium" color="myGray.700">
        {t('contribution.contributions', { count: totalContributions })}
      </Text>

      <Box ref={wrapperRef} overflow="hidden">
        {/* Month labels */}
        <Box ml={`${labelW + gap}px`} mb="2px">
          <Box display="flex" gap={`${gap}px`}>
            {weeks.map((week, weekIdx) => {
              const monthLabel = monthPositions.find(m => m.weekIdx === weekIdx)
              return (
                <Box key={weekIdx} w={`${cellSize}px`} flexShrink={0}>
                  {monthLabel && (
                    <Text fontSize="10px" color="myGray.500" whiteSpace="nowrap">
                      {monthLabel.label}
                    </Text>
                  )}
                </Box>
              )
            })}
          </Box>
        </Box>

        {/* Day labels + grid */}
        <Box display="flex" alignItems="flex-start">
          <VStack spacing={`${gap}px`} mr={`${gap}px`} mt="0">
            {dayLabels.map((label, i) => (
              <Box key={i} h={`${cellSize}px`} fontSize="10px" color="myGray.500" lineHeight={`${cellSize}px`} minW={`${labelW}px`}>
                {label}
              </Box>
            ))}
          </VStack>

          <Box display="flex" gap={`${gap}px`}>
            {weeks.map((week, weekIdx) => (
              <VStack key={weekIdx} spacing={`${gap}px`}>
                {week.map((day, dayIdx) => {
                  const dateStr = fmtDate(day.date)
                  const tooltipText = t('contribution.contributionsTooltip', { count: day.count, date: dateStr })
                  return (
                    <Box
                      key={dayIdx}
                      w={`${cellSize}px`}
                      h={`${cellSize}px`}
                      borderRadius="2px"
                      bg={getColor(day.count, maxCount)}
                      cursor="pointer"
                      transition="all 0.1s"
                      _hover={{ opacity: 0.8 }}
                      title={tooltipText}
                    />
                  )
                })}
              </VStack>
            ))}
          </Box>
        </Box>

        {/* Legend */}
        <HStack spacing={1} justify="flex-end" mt="4px">
          <Text fontSize="10px" color="myGray.500">{t('contribution.less')}</Text>
          {['myGray.100', 'primary.200', 'primary.400', 'primary.600', 'primary.800'].map((color) => (
            <Box key={color} w="11px" h="11px" borderRadius="2px" bg={color} />
          ))}
          <Text fontSize="10px" color="myGray.500">{t('contribution.more')}</Text>
        </HStack>
      </Box>
    </VStack>
  )
}