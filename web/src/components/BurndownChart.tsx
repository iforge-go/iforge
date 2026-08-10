'use client'

import { Box, Text, Flex } from '@chakra-ui/react'
import { useI18n } from '@/contexts/I18nContext'

interface BurndownDataPoint {
  date: string
  expectedPoints: number
  actualPoints: number | null
  isToday?: boolean
}

interface BurndownChartProps {
  data: BurndownDataPoint[]
}

export default function BurndownChart({ data }: BurndownChartProps) {
  const { t } = useI18n()

  if (!data || data.length === 0) {
    return (
      <Box textAlign="center" py={8}>
        <Text color="gray.500">{t('pms.noBurndownData')}</Text>
      </Box>
    )
  }

  const allValues = data.flatMap(d => [
    d.expectedPoints,
    ...(d.actualPoints !== null ? [d.actualPoints] : []),
  ])
  const maxPoints = Math.max(...allValues, 1)

  const chartHeight = 320
  const chartWidth = 700
  const padding = { top: 30, right: 30, bottom: 50, left: 50 }

  const innerWidth = chartWidth - padding.left - padding.right
  const innerHeight = chartHeight - padding.top - padding.bottom

  const xScale = (index: number) => padding.left + (index / (data.length - 1)) * innerWidth
  const yScale = (value: number) => padding.top + innerHeight - (value / maxPoints) * innerHeight

  // 理想线路径（虚线）
  const expectedPath = data.map((d, i) => `${i === 0 ? 'M' : 'L'} ${xScale(i)} ${yScale(d.expectedPoints)}`).join(' ')

  // 实际线路径（实线，跳过 null 值）
  const actualSegments: string[] = []
  let currentSegment = ''
  data.forEach((d, i) => {
    if (d.actualPoints !== null) {
      const cmd = currentSegment === '' ? 'M' : 'L'
      currentSegment += `${cmd} ${xScale(i)} ${yScale(d.actualPoints)} `
    } else {
      if (currentSegment) {
        actualSegments.push(currentSegment.trim())
        currentSegment = ''
      }
    }
  })
  if (currentSegment) actualSegments.push(currentSegment.trim())

  // 今天的位置
  const todayIndex = data.findIndex(d => d.isToday)
  const todayX = todayIndex >= 0 ? xScale(todayIndex) : null

  return (
    <Box w="100%">
      <Box w="100%" overflowX="auto">
        <svg width="100%" viewBox={`0 0 ${chartWidth} ${chartHeight}`} style={{ display: 'block', minWidth: 500 }}>
          {/* Grid lines + Y-axis labels */}
          {[0, 0.25, 0.5, 0.75, 1].map(ratio => {
            const y = padding.top + innerHeight - ratio * innerHeight
            const value = Math.round(maxPoints * ratio)
            return (
              <g key={ratio}>
                <line x1={padding.left} y1={y} x2={chartWidth - padding.right} y2={y} stroke="#e2e8f0" strokeWidth="1" />
                <text x={padding.left - 8} y={y + 4} textAnchor="end" fontSize="11" fill="#718096">
                  {value}
                </text>
              </g>
            )
          })}

          {/* X-axis labels */}
          {data.map((d, i) => {
            const step = Math.max(1, Math.ceil(data.length / 8))
            if (i % step === 0 || i === data.length - 1) {
              const x = xScale(i)
              const date = new Date(d.date)
              return (
                <text key={i} x={x} y={chartHeight - padding.bottom + 20} textAnchor="middle" fontSize="10" fill="#718096">
                  {`${date.getMonth() + 1}/${date.getDate()}`}
                </text>
              )
            }
            return null
          })}

          {/* Today vertical line */}
          {todayX !== null && (
            <g>
              <line
                x1={todayX}
                y1={padding.top}
                x2={todayX}
                y2={padding.top + innerHeight}
                stroke="#f59e0b"
                strokeWidth="1.5"
                strokeDasharray="3,3"
              />
              <text x={todayX} y={padding.top - 8} textAnchor="middle" fontSize="10" fill="#f59e0b" fontWeight="600">
                {t('pms.today')}
              </text>
            </g>
          )}

          {/* Expected line (dashed) */}
          <path d={expectedPath} fill="none" stroke="#94a3b8" strokeWidth="2" strokeDasharray="6,4" />

          {/* Actual line (solid, multiple segments) */}
          {actualSegments.map((segment, i) => (
            <path key={i} d={segment} fill="none" stroke="#3b82f6" strokeWidth="2.5" />
          ))}

          {/* Actual data points */}
          {data.map((d, i) => {
            if (d.actualPoints === null) return null
            return (
              <circle
                key={i}
                cx={xScale(i)}
                cy={yScale(d.actualPoints)}
                r="4"
                fill="#3b82f6"
                stroke="white"
                strokeWidth="1.5"
              />
            )
          })}

          {/* Legend */}
          <g transform={`translate(${padding.left + 10}, ${padding.top + 5})`}>
            <rect x="-5" y="-12" width="180" height="58" fill="white" fillOpacity="0.9" rx="4" />
            <line x1="0" y1="0" x2="24" y2="0" stroke="#94a3b8" strokeWidth="2" strokeDasharray="6,4" />
            <text x="30" y="4" fontSize="11" fill="#4a5568">{t('pms.idealBurndown')}</text>
            <line x1="0" y1="20" x2="24" y2="20" stroke="#3b82f6" strokeWidth="2.5" />
            <circle cx="12" cy="20" r="3.5" fill="#3b82f6" stroke="white" strokeWidth="1" />
            <text x="30" y="24" fontSize="11" fill="#4a5568">{t('pms.actualBurndown')}</text>
            {todayX !== null && (
              <>
                <line x1="0" y1="40" x2="24" y2="40" stroke="#f59e0b" strokeWidth="1.5" strokeDasharray="3,3" />
                <text x="30" y="44" fontSize="11" fill="#4a5568">{t('pms.today')}</text>
              </>
            )}
          </g>
        </svg>
      </Box>

      {/* Summary stats */}
      <Flex justify="center" gap={8} mt={4} flexWrap="wrap">
        <Text fontSize="sm" color="gray.600">
          {t('pms.totalStoryPoints')}: <Text as="span" fontWeight="bold" color="gray.900">{maxPoints}</Text>
        </Text>
        <Text fontSize="sm" color="gray.600">
          {t('pms.remaining')}: <Text as="span" fontWeight="bold" color="blue.500">
            {data.find(d => d.isToday)?.actualPoints ?? data[data.length - 1]?.actualPoints ?? 0}
          </Text>
        </Text>
      </Flex>
    </Box>
  )
}
