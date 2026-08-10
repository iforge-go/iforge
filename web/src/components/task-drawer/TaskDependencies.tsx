'use client'

import { useState, useEffect, useCallback } from 'react'
import {
  Box,
  HStack,
  VStack,
  Text,
  Button,
  IconButton,
  Icon,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  Input,
  InputGroup,
  InputLeftElement,
  Spinner,
  Badge,
  useToast,
} from '@chakra-ui/react'
import { FiLink, FiPlus, FiX, FiSearch, FiAlertCircle } from 'react-icons/fi'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { DependencyTask } from '@/lib/types'

interface TaskDependenciesProps {
  projectSlug: string
  taskId: number
  canEdit: boolean
}

// 判断状态是否为完成态(对齐后端 isClosedStatusSlug)
function isClosedStatus(status: string): boolean {
  return ['done', 'closed', 'completed', 'archived'].includes(status)
}

// 判断依赖的前置条件是否未满足(用于显示"阻塞中"badge)
// FS/FF: B 未完成; SS/SF: B 未开始
function isPrerequisiteUnmet(depType: string | undefined, status: string): boolean {
  const done = isClosedStatus(status)
  const started = done || ['in_progress', 'active', 'doing', 'review'].includes(status)
  switch (depType) {
    case 'fs': case 'ff': return !done
    case 'ss': case 'sf': return !started
    default: return !done
  }
}

// 依赖类型短标签
function depTypeLabel(depType: string | undefined): string {
  return depType?.toUpperCase() || 'FS'
}

export default function TaskDependencies({ projectSlug, taskId, canEdit }: TaskDependenciesProps) {
  const { t } = useI18n()
  const toast = useToast()

  const [dependencies, setDependencies] = useState<DependencyTask[]>([])
  const [blocks, setBlocks] = useState<DependencyTask[]>([])
  const [loading, setLoading] = useState(false)

  // Modal state
  const [isOpen, setIsOpen] = useState(false)
  const [searchKeyword, setSearchKeyword] = useState('')
  const [searchResults, setSearchResults] = useState<any[]>([])
  const [searchLoading, setSearchLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [selectedDepType, setSelectedDepType] = useState('fs')

  const loadData = useCallback(async () => {
    if (!projectSlug || !taskId) return
    setLoading(true)
    try {
      const [depRes, blockRes] = await Promise.all([
        api.getTaskDependencies(projectSlug, taskId),
        api.getTaskBlocks(projectSlug, taskId),
      ])
      setDependencies(depRes.dependencies || [])
      setBlocks(blockRes.blocks || [])
    } catch {
      // 静默失败，避免drawer打开时报错刷屏
    } finally {
      setLoading(false)
    }
  }, [projectSlug, taskId])

  useEffect(() => {
    loadData()
  }, [loadData])

  // 搜索任务(300ms 防抖),排除自身和已依赖的任务
  useEffect(() => {
    if (!searchKeyword.trim()) {
      setSearchResults([])
      return
    }
    setSearchLoading(true)
    const timer = setTimeout(async () => {
      try {
        const result = await api.getTasks(projectSlug, { q: searchKeyword, limit: 20 })
        const existingIds = new Set([taskId, ...dependencies.map((d) => d.taskId)])
        setSearchResults((result || []).filter((t: any) => !existingIds.has(t.taskId)))
      } catch {
        setSearchResults([])
      } finally {
        setSearchLoading(false)
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [searchKeyword, projectSlug, taskId, dependencies])

  const handleAdd = async (dependsOnTaskId: number) => {
    setSubmitting(true)
    try {
      await api.addTaskDependency(projectSlug, taskId, dependsOnTaskId, selectedDepType)
      toast({ title: t('pms.dependencyAddedSuccess'), status: 'success', duration: 2000 })
      await loadData()
      setSearchKeyword('')
      setSearchResults([])
      setSelectedDepType('fs')
    } catch (error: any) {
      const status = (error as any).status
      let msg = error.message
      if (status === 409) msg = t('pms.dependencyAlreadyExists')
      else if (status === 422) msg = t('pms.circularDependency')
      else if (status === 400) msg = t('pms.selfDependencyNotAllowed')
      toast({ title: msg, status: 'error', duration: 3000 })
    } finally {
      setSubmitting(false)
    }
  }

  const handleRemove = async (dependsOnTaskId: number) => {
    try {
      await api.removeTaskDependency(projectSlug, taskId, dependsOnTaskId)
      toast({ title: t('pms.dependencyRemovedSuccess'), status: 'success', duration: 2000 })
      await loadData()
    } catch (error: any) {
      toast({ title: error.message, status: 'error', duration: 3000 })
    }
  }

  const openModal = () => {
    setIsOpen(true)
    setSearchKeyword('')
    setSearchResults([])
  }

  return (
    <Box>
      <HStack justify="space-between" mb={2}>
        <HStack spacing={2}>
          <Icon as={FiLink} color="gray.500" />
          <Text fontSize="sm" fontWeight="medium" color="gray.700">
            {t('pms.dependencies')}
          </Text>
          {dependencies.length > 0 && (
            <Text fontSize="xs" color="gray.500">({dependencies.length})</Text>
          )}
        </HStack>
        {canEdit && (
          <Button
            size="xs"
            variant="ghost"
            colorScheme="blue"
            leftIcon={<Icon as={FiPlus} />}
            onClick={openModal}
          >
            {t('pms.addDependency')}
          </Button>
        )}
      </HStack>

      {loading ? (
        <Spinner size="xs" />
      ) : dependencies.length === 0 ? (
        <Text fontSize="sm" color="gray.400">{t('pms.noDependencies')}</Text>
      ) : (
        <VStack align="stretch" spacing={1}>
          {dependencies.map((d) => {
            const unmet = isPrerequisiteUnmet(d.dependencyType, d.status)
            return (
              <HStack
                key={d.taskId}
                spacing={2}
                bg="gray.50"
                borderRadius="md"
                px={2}
                py={1.5}
              >
                <Text fontSize="xs" color="gray.500" flexShrink={0}>#{d.taskId}</Text>
                <Badge fontSize="2xs" colorScheme="blue" variant="subtle">{depTypeLabel(d.dependencyType)}</Badge>
                <Text fontSize="xs" color="gray.700" flex={1} minW={0} isTruncated>{d.title}</Text>
                {unmet && (
                  <Badge fontSize="2xs" colorScheme="orange" variant="subtle">{t('pms.blocking')}</Badge>
                )}
                {canEdit && (
                  <IconButton
                    aria-label={t('pms.removeDependency')}
                    icon={<Icon as={FiX} />}
                    size="xs"
                    variant="ghost"
                    colorScheme="gray"
                    minW="20px"
                    w="20px"
                    h="20px"
                    onClick={() => handleRemove(d.taskId)}
                  />
                )}
              </HStack>
            )
          })}
        </VStack>
      )}

      {/* 阻塞了谁(只读) */}
      {blocks.length > 0 && (
        <Box mt={3}>
          <HStack spacing={2} mb={2}>
            <Icon as={FiAlertCircle} color="orange.400" />
            <Text fontSize="sm" fontWeight="medium" color="gray.700">
              {t('pms.blocks')}
            </Text>
            <Text fontSize="xs" color="gray.500">({blocks.length})</Text>
          </HStack>
          <Box bg="orange.50" borderRadius="md" p={2}>
            <VStack align="stretch" spacing={1}>
              {blocks.map((b) => (
                <HStack key={b.taskId} spacing={2} fontSize="xs">
                  <Text color="gray.500" flexShrink={0}>#{b.taskId}</Text>
                  <Badge fontSize="2xs" colorScheme="blue" variant="subtle">{depTypeLabel(b.dependencyType)}</Badge>
                  <Text color="gray.700" flex={1} minW={0} isTruncated>{b.title}</Text>
                </HStack>
              ))}
            </VStack>
          </Box>
        </Box>
      )}

      {/* 添加依赖弹窗 */}
      <Modal isOpen={isOpen} onClose={() => setIsOpen(false)} size="md">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('pms.addDependency')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack align="stretch" spacing={3}>
              {/* 依赖类型选择器 */}
              <HStack spacing={2} wrap="wrap">
                <Text fontSize="xs" color="gray.500" flexShrink={0}>{t('pms.dependencyType')}</Text>
                <HStack spacing={1}>
                  {(['fs', 'ss', 'ff', 'sf'] as const).map((tp) => (
                    <Button
                      key={tp}
                      size="xs"
                      variant={selectedDepType === tp ? 'solid' : 'outline'}
                      colorScheme="blue"
                      onClick={() => setSelectedDepType(tp)}
                    >
                      {t(`pms.depType${tp.toUpperCase()}`)}
                    </Button>
                  ))}
                </HStack>
              </HStack>
              <Text fontSize="2xs" color="gray.400">{t(`pms.depTypeDesc${selectedDepType.toUpperCase()}`)}</Text>
              <InputGroup>
                <InputLeftElement pointerEvents="none">
                  <Icon as={FiSearch} color="gray.400" />
                </InputLeftElement>
                <Input
                  size="sm"
                  placeholder={t('pms.searchTaskPlaceholder')}
                  value={searchKeyword}
                  onChange={(e) => setSearchKeyword(e.target.value)}
                  autoFocus
                />
              </InputGroup>
              {searchLoading ? (
                <HStack spacing={2}><Spinner size="xs" /><Text fontSize="sm" color="gray.500">{t('common.loading')}</Text></HStack>
              ) : searchResults.length === 0 ? (
                searchKeyword.trim() && <Text fontSize="sm" color="gray.500">{t('pms.noTasksFound')}</Text>
              ) : (
                <VStack align="stretch" spacing={1} maxH="300px" overflowY="auto">
                  {searchResults.map((task: any) => (
                    <HStack
                      key={task.taskId}
                      spacing={2}
                      px={2}
                      py={1.5}
                      borderRadius="md"
                      bg="gray.50"
                      cursor="pointer"
                      _hover={{ bg: 'blue.50' }}
                      onClick={() => !submitting && handleAdd(task.taskId)}
                    >
                      <Text fontSize="xs" color="gray.500" flexShrink={0}>#{task.taskId}</Text>
                      <Text fontSize="xs" color="gray.700" flex={1} minW={0} isTruncated>{task.title}</Text>
                    </HStack>
                  ))}
                </VStack>
              )}
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="outline" mr={3} onClick={() => setIsOpen(false)}>
              {t('common.cancel')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  )
}
