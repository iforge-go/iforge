'use client'

import { Box, Grid, Text, Button, Flex, Tooltip } from '@chakra-ui/react'
import { FiChevronLeft, FiChevronRight } from 'react-icons/fi'
import { useState } from 'react'

interface Sprint {
  id: number
  slug: string
  title: string
  startDate: string | null
  endDate: string | null
  status: string
}

interface SprintCalendarProps {
  sprints: Sprint[]
  onSprintClick?: (sprint: Sprint) => void
}

export default function SprintCalendar({ sprints, onSprintClick }: SprintCalendarProps) {
  const [currentDate, setCurrentDate] = useState(new Date())

  const year = currentDate.getFullYear()
  const month = currentDate.getMonth()

  const firstDayOfMonth = new Date(year, month, 1)
  const lastDayOfMonth = new Date(year, month + 1, 0)
  const daysInMonth = lastDayOfMonth.getDate()
  const startingDayOfWeek = firstDayOfMonth.getDay()

  const monthNames = ['一月', '二月', '三月', '四月', '五月', '六月',
    '七月', '八月', '九月', '十月', '十一月', '十二月']

  const dayNames = ['日', '一', '二', '三', '四', '五', '六']

  const getSprintsForDate = (day: number) => {
    const date = new Date(year, month, day)
    return sprints.filter(sprint => {
      if (!sprint.startDate || !sprint.endDate) return false
      const startDate = new Date(sprint.startDate)
      const endDate = new Date(sprint.endDate)
      return date >= startDate && date <= endDate
    })
  }

  const isSprintStart = (day: number, sprint: Sprint) => {
    if (!sprint.startDate) return false
    const date = new Date(year, month, day)
    const startDate = new Date(sprint.startDate)
    return date.toDateString() === startDate.toDateString()
  }

  const isSprintEnd = (day: number, sprint: Sprint) => {
    if (!sprint.endDate) return false
    const date = new Date(year, month, day)
    const endDate = new Date(sprint.endDate)
    return date.toDateString() === endDate.toDateString()
  }

  const previousMonth = () => {
    setCurrentDate(new Date(year, month - 1, 1))
  }

  const nextMonth = () => {
    setCurrentDate(new Date(year, month + 1, 1))
  }

  const today = new Date()
  const isToday = (day: number) => {
    return year === today.getFullYear() &&
      month === today.getMonth() &&
      day === today.getDate()
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'open': return 'gray.400'
      case 'active': return 'blue.400'
      case 'closed': return 'green.400'
      default: return 'gray.400'
    }
  }

  const days = []
  for (let i = 0; i < startingDayOfWeek; i++) {
    days.push(<Box key={`empty-${i}`} />)
  }

  for (let day = 1; day <= daysInMonth; day++) {
    const daySprints = getSprintsForDate(day)
    const isTodayDate = isToday(day)

    days.push(
      <Box
        key={day}
        p="2"
        minH="80px"
        border="1px solid"
        borderColor={isTodayDate ? 'blue.300' : 'gray.200'}
        borderRadius="md"
        bg={isTodayDate ? 'blue.50' : 'white'}
        position="relative"
      >
        <Text
          fontSize="sm"
          fontWeight={isTodayDate ? 'bold' : 'normal'}
          color={isTodayDate ? 'blue.600' : 'gray.700'}
          mb="1"
        >
          {day}
        </Text>
        <Box>
          {daySprints.slice(0, 2).map((sprint) => {
            const isStart = isSprintStart(day, sprint)
            const isEnd = isSprintEnd(day, sprint)
            return (
              <Tooltip
                key={sprint.slug}
                label={`${sprint.title} (${sprint.status})`}
                fontSize="xs"
              >
                <Box
                  fontSize="xs"
                  px="1"
                  py="0.5"
                  mb="0.5"
                  borderRadius="sm"
                  bg={getStatusColor(sprint.status)}
                  color="white"
                  cursor="pointer"
                  onClick={() => onSprintClick?.(sprint)}
                  overflow="hidden"
                  whiteSpace="nowrap"
                  textOverflow="ellipsis"
                  borderLeftRadius={isStart ? 'md' : 'sm'}
                  borderRightRadius={isEnd ? 'md' : 'sm'}
                  _hover={{ opacity: 0.8 }}
                >
                  {sprint.title}
                </Box>
              </Tooltip>
            )
          })}
          {daySprints.length > 2 && (
            <Text fontSize="xs" color="gray.500" pl="1">
              +{daySprints.length - 2} 更多
            </Text>
          )}
        </Box>
      </Box>
    )
  }

  return (
    <Box>
      <Flex justify="space-between" align="center" mb="4">
        <Button onClick={previousMonth} variant="outline" size="sm">
          <FiChevronLeft />
        </Button>
        <Text fontSize="xl" fontWeight="bold">
          {monthNames[month]} {year}
        </Text>
        <Button onClick={nextMonth} variant="outline" size="sm">
          <FiChevronRight />
        </Button>
      </Flex>

      <Grid templateColumns="repeat(7, 1fr)" gap="2" mb="2">
        {dayNames.map((dayName) => (
          <Box
            key={dayName}
            p="2"
            textAlign="center"
            fontWeight="bold"
            fontSize="sm"
            color="gray.600"
            bg="gray.50"
            borderRadius="md"
          >
            {dayName}
          </Box>
        ))}
      </Grid>

      <Grid templateColumns="repeat(7, 1fr)" gap="2">
        {days}
      </Grid>
    </Box>
  )
}
