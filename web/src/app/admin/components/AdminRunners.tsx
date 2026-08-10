'use client'

import {
  Box,
  VStack,
  HStack,
  Text,
  Icon,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Badge,
  Spinner,
  IconButton,
  Button,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
  useDisclosure,
  useToast,
} from '@chakra-ui/react'
import { FiCpu, FiRefreshCw, FiTrash2 } from 'react-icons/fi'
import { useEffect, useState, useCallback, useRef } from 'react'
import { api } from '@/lib/api'
import { Runner } from '@/lib/types'
import { useI18n } from '@/contexts/I18nContext'
import { formatRelativeTime } from '@/lib/time'

export default function AdminRunners() {
  const { t } = useI18n()
  const toast = useToast()
  const [runners, setRunners] = useState<Runner[]>([])
  const [loading, setLoading] = useState(true)
  const [deletingRunner, setDeletingRunner] = useState<Runner | null>(null)
  const [deleting, setDeleting] = useState(false)
  const { isOpen, onOpen, onClose } = useDisclosure()
  const cancelRef = useRef<any>(null)

  const loadRunners = useCallback(async () => {
    try {
      const data = await api.listRunners()
      setRunners(data.items || [])
    } catch (error) {
      console.error('Failed to load runners:', error)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadRunners()
    // 每 15 秒刷新一次（心跳间隔 30 秒，stale 检测 30 秒）
    const timer = setInterval(loadRunners, 15000)
    return () => clearInterval(timer)
  }, [loadRunners])

  const statusColor = (status: string) => {
    switch (status) {
      case 'online': return 'green'
      case 'busy': return 'blue'
      case 'offline': return 'red'
      default: return 'gray'
    }
  }

  const handleDeleteClick = (runner: Runner) => {
    setDeletingRunner(runner)
    onOpen()
  }

  const handleDeleteConfirm = async () => {
    if (!deletingRunner) return
    setDeleting(true)
    try {
      await api.deleteRunner(deletingRunner.id)
      toast({ status: 'success', description: t('admin.runnerDeleted') })
      await loadRunners()
    } catch (error: any) {
      toast({ status: 'error', description: error.message })
    } finally {
      setDeleting(false)
      onClose()
      setDeletingRunner(null)
    }
  }

  if (loading) {
    return (
      <VStack spacing={4} py={8}>
        <Spinner size="xl" />
        <Text color="myGray.500">{t('common.loading')}</Text>
      </VStack>
    )
  }

  return (
    <VStack spacing={4} align="stretch">
      <HStack justify="space-between">
        <HStack spacing={2}>
          <Icon as={FiCpu} color="primary.500" />
          <Text fontSize="sm" color="myGray.600">
            {runners.length} {t('admin.runners')}
          </Text>
        </HStack>
        <HStack
          spacing={1}
          cursor="pointer"
          onClick={loadRunners}
          color="myGray.500"
          _hover={{ color: 'primary.500' }}
        >
          <Icon as={FiRefreshCw} w={3.5} h={3.5} />
          <Text fontSize="sm">{t('common.refresh')}</Text>
        </HStack>
      </HStack>

      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
        <Table variant="simple" size="sm">
          <Thead bg="myGray.50">
            <Tr>
              <Th color="myGray.600">{t('admin.runnerName')}</Th>
              <Th color="myGray.600" width="100px">{t('admin.runnerStatus')}</Th>
              <Th color="myGray.600" width="80px">{t('admin.runnerType')}</Th>
              <Th color="myGray.600" width="120px">{t('admin.lastHeartbeat')}</Th>
              <Th color="myGray.600" width="80px">{t('admin.runnerVersion')}</Th>
              <Th color="myGray.600" width="60px">{t('admin.runnerBuiltin')}</Th>
              <Th color="myGray.600" width="60px">{t('admin.actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {runners.length === 0 ? (
              <Tr>
                <Td colSpan={7}>
                  <Text color="myGray.400" textAlign="center" py={4}>
                    {t('admin.noRunners')}
                  </Text>
                </Td>
              </Tr>
            ) : (
              runners.map((runner) => (
                <Tr key={runner.id} _hover={{ bg: 'myGray.50' }}>
                  <Td>
                    <HStack spacing={2}>
                      <Icon as={FiCpu} w={3.5} h={3.5} color="myGray.400" />
                      <VStack spacing={0} align="flex-start">
                        <Text fontSize="sm" fontWeight="medium" color="myGray.800">
                          {runner.name}
                        </Text>
                        {runner.description && (
                          <Text fontSize="xs" color="myGray.400">
                            {runner.description}
                          </Text>
                        )}
                      </VStack>
                    </HStack>
                  </Td>
                  <Td>
                    <Badge colorScheme={statusColor(runner.status)} variant="subtle">
                      {runner.status}
                    </Badge>
                  </Td>
                  <Td>
                    <Text fontSize="sm" color="myGray.600">{runner.type}</Text>
                  </Td>
                  <Td>
                    {runner.lastHeartbeat ? (
                      <Text fontSize="sm" color="myGray.500" title={runner.lastHeartbeat}>
                        {formatRelativeTime(runner.lastHeartbeat, t)}
                      </Text>
                    ) : (
                      <Text fontSize="sm" color="myGray.300">—</Text>
                    )}
                  </Td>
                  <Td>
                    <Text fontSize="xs" color="myGray.500" fontFamily="mono">
                      {runner.version || '—'}
                    </Text>
                  </Td>
                  <Td>
                    {runner.isBuiltin ? (
                      <Badge colorScheme="gray" variant="subtle" fontSize="xs">
                        {t('admin.yes')}
                      </Badge>
                    ) : (
                      <Text fontSize="sm" color="myGray.300">—</Text>
                    )}
                  </Td>
                  <Td>
                    {!runner.isBuiltin && (
                      <IconButton
                        aria-label={t('common.delete')}
                        icon={<Icon as={FiTrash2} />}
                        size="sm"
                        variant="ghost"
                        colorScheme="red"
                        onClick={() => handleDeleteClick(runner)}
                      />
                    )}
                  </Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>
      </Box>

      <AlertDialog
        isOpen={isOpen}
        leastDestructiveRef={cancelRef}
        onClose={onClose}
        blockScrollOnMount={false}
      >
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              {t('admin.deleteRunner')}
            </AlertDialogHeader>
            <AlertDialogBody>
              {t('admin.deleteRunnerConfirm', { name: deletingRunner?.name || '' })}
            </AlertDialogBody>
            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={onClose}>
                {t('common.cancel')}
              </Button>
              <Button colorScheme="red" onClick={handleDeleteConfirm} ml={3} isLoading={deleting}>
                {t('common.delete')}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>
    </VStack>
  )
}
