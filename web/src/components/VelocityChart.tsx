'use client'

import { Box, Text, Flex } from '@chakra-ui/react'
import { useI18n } from '@/contexts/I18nContext'
import type { VelocityDataPoint } from '@/lib/types'

interface VelocityChartProps {
  data: VelocityDataPoint[]
}

// 速度图（Velocity Chart）：展示最近 N 个已关闭 Sprint 的承诺 vs 完成故事点。
// 对齐 Jira Velocity Report：每个 Sprint 双柱（承诺=灰蓝，完成=绿），横轴按时间排列。
// 参照 BurndownChart.tsx 的 SVG 风格保持视觉一致。
export default function VelocityChart({ data }: VelocityChartProps) {
  const { t } = useI18n()

  if (!data || data.length === 0) {
    return (
      <Box textAlign="center" py={8}>
        <Text color="gray.500">{t('pms.noVelocityData')}</Text>
      </Box>
    )
  }

  const maxValue = Math.max(
    ...data.flatMap((d) => [d.committedPoints, d.completedPoints]),
    1,
  )

  const chartHeight = 320
  const barWidth = 28
  const groupGap = 16
  const innerBarGap = 4
  const groupWidth = barWidth * 2 + innerBarGap
  const chartWidth = Math.max(500, data.length * (groupWidth + groupGap) + groupGap + 80)
  const padding = { top: 30, right: 30, bottom: 60, left: 50 }

  const innerWidth = chartWidth - padding.left - padding.right
  const innerHeight = chartHeight - padding.top - padding.bottom

  const groupSpacing = data.length > 1 ? (innerWidth - data.length * groupWidth) / (data.length - 1) : 0
  const xGroup = (index: number) => padding.left + index * (groupWidth + groupSpacing)

  const yScale = (value: number) => padding.top + innerHeight - (value / maxValue) * innerHeight

  // 平均完成速度（已完成点数的平均值，用于趋势参考线）
  const avgCompleted = Math.round(
    data.reduce((sum, d) => sum + d.completedPoints, 0) / data.length,
  )
  const avgY = yScale(avgCompleted)

  return (
    <Box w="100%">
      <Box w="100%" overflowX="auto">
        <svg width="100%" viewBox={`0 0 ${chartWidth} ${chartHeight}`} style={{ display: 'block', minWidth: 500 }}>
          {/* Grid lines + Y-axis labels */}
          {[0, 0.25, 0.5, 0.75, 1].map((ratio) => {
            const y = padding.top + innerHeight - ratio * innerHeight
            const value = Math.round(maxValue * ratio)
            return (
              <g key={ratio}>
                <line x1={padding.left} y1={y} x2={chartWidth - padding.right} y2={y} stroke="#e2e8f0" strokeWidth="1" />
                <text x={padding.left - 8} y={y + 4} textAnchor="end" fontSize="11" fill="#718096">
                  {value}
                </text>
              </g>
            )
          })}

          {/* Average completed line (trend reference, dashed green) */}
          {avgCompleted > 0 && (
            <g>
              <line
                x1={padding.left}
                y1={avgY}
                x2={chartWidth - padding.right}
                y2={avgY}
                stroke="#22c55e"
                strokeWidth="1.5"
                strokeDasharray="5,4"
                opacity="0.6"
              />
              <text x={chartWidth - padding.right} y={avgY - 5} textAnchor="end" fontSize="10" fill="#22c55e" fontWeight="600">
                {t('pms.avgVelocity')}: {avgCompleted}
              </text>
            </g>
          )}

          {/* Dual bars per sprint */}
          {data.map((d, i) => {
            const groupX = xGroup(i)
            const committedHeight = (d.committedPoints / maxValue) * innerHeight
            const completedHeight = (d.completedPoints / maxValue) * innerHeight
            return (
              <g key={i}>
                {/* Committed bar (gray-blue) */}
                <rect
                  x={groupX}
                  y={padding.top + innerHeight - committedHeight}
                  width={barWidth}
                  height={committedHeight}
                  fill="#94a3b8"
                  rx="3"
                />
                <text x={groupX + barWidth / 2} y={padding.top + innerHeight - committedHeight - 5} textAnchor="middle" fontSize="10" fill="#64748b" fontWeight="600">
                  {d.committedPoints}
                </text>

                {/* Completed bar (green) */}
                <rect
                  x={groupX + barWidth + innerBarGap}
                  y={padding.top + innerHeight - completedHeight}
                  width={barWidth}
                  height={completedHeight}
                  fill="#22c55e"
                  rx="3"
                />
                <text x={groupX + barWidth + innerBarGap + barWidth / 2} y={padding.top + innerHeight - completedHeight - 5} textAnchor="middle" fontSize="10" fill="#16a34a" fontWeight="600">
                  {d.completedPoints}
                </text>

                {/* X-axis label (sprint title, truncated) */}
                <text
                  x={groupX + groupWidth / 2}
                  y={chartHeight - padding.bottom + 18}
                  textAnchor="middle"
                  fontSize="10"
                  fill="#718096"
                >
                  {d.title.length > 12 ? d.title.slice(0, 11) + '…' : d.title}
                </text>
                {d.completedDate && (
                  <text
                    x={groupX + groupWidth / 2}
                    y={chartHeight - padding.bottom + 32}
                    textAnchor="middle"
                    fontSize="9"
                    fill="#94a3b8"
                  >
                    {new Date(d.completedDate).toLocaleDateString()}
                  </text>
                )}
              </g>
            )
          })}

          {/* Legend */}
          <g transform={`translate(${padding.left + 10}, ${padding.top + 5})`}>
            <rect x="-5" y="-12" width="200" height="58" fill="white" fillOpacity="0.9" rx="4" />
            <rect x="0" y="-5" width="14" height="10" fill="#94a3b8" rx="2" />
            <text x="20" y="4" fontSize="11" fill="#4a5568">{t('pms.committedPoints')}</text>
            <rect x="0" y="15" width="14" height="10" fill="#22c55e" rx="2" />
            <text x="20" y="24" fontSize="11" fill="#4a5568">{t('pms.completedPoints')}</text>
          </g>
        </svg>
      </Box>

      {/* Summary stats */}
      <Flex justify="center" gap={8} mt={4} flexWrap="wrap">
        <Text fontSize="sm" color="gray.600">
          {t('pms.sprintsClosed')}: <Text as="span" fontWeight="bold" color="gray.900">{data.length}</Text>
        </Text>
        <Text fontSize="sm" color="gray.600">
          {t('pms.avgVelocity')}: <Text as="span" fontWeight="bold" color="green.500">{avgCompleted}</Text>
        </Text>
      </Flex>
    </Box>
  )
}
