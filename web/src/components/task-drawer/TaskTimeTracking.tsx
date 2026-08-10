'use client'

import { useState } from 'react'
import {
  Avatar,
  Badge,
  Box,
  Button,
  Heading,
  HStack,
  Icon,
  Input,
  Progress,
  Text,
  VStack,
  IconButton,
  useToast,
} from '@chakra-ui/react'
import { FiClock, FiTrash2 } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import { TaskWorkLog } from '@/lib/types'

interface TaskTimeTrackingProps {
  estimatedHours: number | null
  actualHours: number
  workLogs: TaskWorkLog[]
  canEdit: boolean
  onAddLog: (hours: number, description: string) => void
  onDeleteLog: (logId: number) => void
  addingLog: boolean
}

export default function TaskTimeTracking({
  estimatedHours,
  actualHours,
  workLogs,
  canEdit,
  onAddLog,
  onDeleteLog,
  addingLog,
}: TaskTimeTrackingProps) {
  const { t } = useI18n()
  const { user: currentUser } = useCurrentUser()
  const toast = useToast()
  const [hoursInput, setHoursInput] = useState('')
  const [descInput, setDescInput] = useState('')

  // 进度计算：实际 / 预估
  const hasEstimate = estimatedHours != null && estimatedHours > 0
  const progressPercent = hasEstimate ? Math.min((actualHours / estimatedHours!) * 100, 100) : 0
  const isOvertime = hasEstimate && actualHours > estimatedHours!
  const remaining = hasEstimate ? Math.max(estimatedHours! - actualHours, 0) : null
  const overtime = hasEstimate ? Math.max(actualHours - estimatedHours!, 0) : 0

  function handleSubmit() {
    const hours = parseFloat(hoursInput)
    if (isNaN(hours) || hours <= 0) {
      toast({ title: t('pms.addWorkLogFailed'), description: t('pms.hoursMustBePositive'), status: 'warning', duration: 3000 })
      return
    }
    onAddLog(hours, descInput.trim())
    setHoursInput('')
    setDescInput('')
  }

  return (
    <Box>
      <HStack mb={3}>
        <Icon as={FiClock} />
        <Heading size="sm">{t('pms.timeTracking')}</Heading>
      </HStack>

      {/* 摘要行：预估 / 实际 / 剩余或超时 */}
      <HStack spacing={4} mb={2} fontSize="sm" flexWrap="wrap">
        <HStack spacing={1}>
          <Text color="gray.500">{t('pms.estimatedHours')}:</Text>
          <Text fontWeight="medium">{estimatedHours ?? 0}h</Text>
        </HStack>
        <HStack spacing={1}>
          <Text color="gray.500">{t('pms.actualHours')}:</Text>
          <Text fontWeight="medium" color={isOvertime ? 'red.500' : 'inherit'}>{actualHours}h</Text>
        </HStack>
        {hasEstimate && (
          <HStack spacing={1}>
            {isOvertime ? (
              <>
                <Text color="gray.500">{t('pms.overtime')}:</Text>
                <Text fontWeight="medium" color="red.500">+{overtime}h</Text>
              </>
            ) : (
              <>
                <Text color="gray.500">{t('pms.remainingHours')}:</Text>
                <Text fontWeight="medium">{remaining}h</Text>
              </>
            )}
          </HStack>
        )}
      </HStack>

      {/* 进度条 */}
      {hasEstimate && (
        <Progress
          value={progressPercent}
          size="sm"
          colorScheme={isOvertime ? 'red' : 'blue'}
          borderRadius="md"
          mb={4}
        />
      )}

      {/* 工时日志历史 */}
      <VStack align="stretch" spacing={2} mb={4}>
        {workLogs.length === 0 ? (
          <Text color="gray.500" textAlign="center" py={2} fontSize="sm">{t('pms.noWorkLogs')}</Text>
        ) : (
          workLogs.map((log) => (
            <Box key={log.logId} p={3} bg="gray.50" borderRadius="md">
              <HStack justify="space-between" mb={1}>
                <HStack spacing={2}>
                  <Avatar size="xs" name={log.userName} />
                  <Text fontWeight="medium" fontSize="sm">{log.userName}</Text>
                  <Badge color="blue.500" bg="blue.50" fontSize="xs" px={2} py={0.5} borderRadius="full">
                    {log.hours}h
                  </Badge>
                </HStack>
                <HStack spacing={2}>
                  <Text fontSize="xs" color="gray.500">
                    {new Date(log.createdAt).toLocaleString()}
                  </Text>
                  {canEdit && log.userName === currentUser?.userName && (
                    <IconButton
                      aria-label={t('common.delete')}
                      icon={<FiTrash2 />}
                      size="xs"
                      variant="ghost"
                      colorScheme="red"
                      onClick={() => onDeleteLog(log.logId)}
                    />
                  )}
                </HStack>
              </HStack>
              {log.description && (
                <Text whiteSpace="pre-wrap" fontSize="sm" color="gray.700" ml={10}>
                  {log.description}
                </Text>
              )}
            </Box>
          ))
        )}
      </VStack>

      {/* 添加工时日志表单 */}
      {canEdit && (
        <HStack spacing={2} align="flex-start">
          <Box>
            <Input
              type="number"
              step="0.5"
              min="0"
              value={hoursInput}
              onChange={(e) => setHoursInput(e.target.value)}
              placeholder={t('pms.hours')}
              size="sm"
              w="90px"
            />
          </Box>
          <Input
            value={descInput}
            onChange={(e) => setDescInput(e.target.value)}
            placeholder={t('pms.descriptionPlaceholder')}
            size="sm"
            flex={1}
          />
          <Button
            colorScheme="blue"
            size="sm"
            onClick={handleSubmit}
            isLoading={addingLog}
            isDisabled={!hoursInput.trim()}
          >
            {t('pms.addWorkLog')}
          </Button>
        </HStack>
      )}
    </Box>
  )
}
