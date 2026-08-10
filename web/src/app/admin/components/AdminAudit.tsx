'use client'

import {
  Box, Heading, Text, Button, VStack, HStack, Table, Thead, Tbody, Tr, Th, Td,
  Badge, Icon, Input, Select, FormControl, Modal, ModalOverlay, ModalContent,
  ModalHeader, ModalBody, ModalCloseButton, Code, Spinner,
} from '@chakra-ui/react'
import { useEffect, useState, useCallback } from 'react'
import { api } from '@/lib/api'
import { AuditLog } from '@/lib/types'
import { FiChevronLeft, FiChevronRight, FiFilter, FiEye } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

// 常见动作选项（用于过滤器下拉）
const ACTION_OPTIONS = [
  { value: '', labelKey: 'admin.auditAllActions' },
  { value: 'repository.create', label: 'repository.create' },
  { value: 'repository.update', label: 'repository.update' },
  { value: 'repository.delete', label: 'repository.delete' },
  { value: 'repository.transfer', label: 'repository.transfer' },
  { value: 'repository.rename', label: 'repository.rename' },
  { value: 'repository.archive', label: 'repository.archive' },
  { value: 'collaborator.add', label: 'collaborator.add' },
  { value: 'collaborator.update', label: 'collaborator.update' },
  { value: 'collaborator.remove', label: 'collaborator.remove' },
  { value: 'organization.create', label: 'organization.create' },
  { value: 'organization.delete', label: 'organization.delete' },
  { value: 'organization.member.add', label: 'organization.member.add' },
  { value: 'organization.member.remove', label: 'organization.member.remove' },
  { value: 'user.create', label: 'user.create' },
  { value: 'user.delete', label: 'user.delete' },
  { value: 'user.update', label: 'user.update' },
  { value: 'project.member.add', label: 'project.member.add' },
  { value: 'project.member.remove', label: 'project.member.remove' },
  { value: 'project.member.update', label: 'project.member.update' },
  { value: 'project.delete', label: 'project.delete' },
]

const RESOURCE_OPTIONS = [
  { value: '', labelKey: 'admin.auditAllResources' },
  { value: 'repository', label: 'repository' },
  { value: 'collaborator', label: 'collaborator' },
  { value: 'organization', label: 'organization' },
  { value: 'user', label: 'user' },
  { value: 'project', label: 'project' },
  { value: 'system', label: 'system' },
]

// 资源类型 -> Badge 颜色
const RESOURCE_COLORS: Record<string, string> = {
  repository: 'blue',
  collaborator: 'cyan',
  organization: 'purple',
  user: 'green',
  project: 'orange',
  system: 'gray',
}

export default function AdminAudit() {
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'

  const [logs, setLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [pageSize] = useState(20)

  // 过滤器
  const [filterActor, setFilterActor] = useState('')
  const [filterAction, setFilterAction] = useState('')
  const [filterResourceType, setFilterResourceType] = useState('')

  // 详情抽屉
  const [detailLog, setDetailLog] = useState<AuditLog | null>(null)

  const loadLogs = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.listAuditLogs({
        actor: filterActor.trim() || undefined,
        action: filterAction || undefined,
        resourceType: filterResourceType || undefined,
        page,
        pageSize,
      })
      setLogs(data.logs || [])
      setTotal(data.total || 0)
    } catch (error) {
      console.error('Failed to load audit logs:', error)
    } finally {
      setLoading(false)
    }
  }, [filterActor, filterAction, filterResourceType, page, pageSize])

  useEffect(() => {
    loadLogs()
  }, [loadLogs])

  const handleFilter = () => {
    setPage(1)
    loadLogs()
  }

  const totalPages = Math.ceil(total / pageSize)

  const formatDateTime = (iso: string) => {
    try {
      return new Date(iso).toLocaleString(dateLocale, {
        year: 'numeric', month: '2-digit', day: '2-digit',
        hour: '2-digit', minute: '2-digit', second: '2-digit',
      })
    } catch {
      return iso
    }
  }

  // 解析 detail JSON 用于详情展示
  const renderDetail = (detailStr: string) => {
    if (!detailStr) return <Text color="myGray.500">{t('admin.auditNoDetail')}</Text>
    try {
      const obj = JSON.parse(detailStr)
      return (
        <VStack align="stretch" spacing={2}>
          {Object.entries(obj).map(([k, v]) => (
            <HStack key={k} align="flex-start" spacing={3}>
              <Text fontSize="sm" color="myGray.600" minW="120px" fontWeight="medium">
                {k}:
              </Text>
              <Code fontSize="sm" whiteSpace="pre-wrap" wordBreak="break-all">
                {typeof v === 'string' ? v : JSON.stringify(v)}
              </Code>
            </HStack>
          ))}
        </VStack>
      )
    } catch {
      return <Code fontSize="sm" whiteSpace="pre-wrap" wordBreak="break-all">{detailStr}</Code>
    }
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
      <HStack justify="space-between" mb={4}>
        <Heading size="md">{t('admin.auditLogs')}</Heading>
        <Text fontSize="sm" color="myGray.600">
          {t('admin.auditTotal', { total })}
        </Text>
      </HStack>

      {/* 过滤器 */}
      <HStack spacing={3} mb={4} align="flex-end" flexWrap="wrap">
        <FormControl width="200px" flexShrink={0}>
          <FormLabelSmall>{t('admin.auditActor')}</FormLabelSmall>
          <Input
            size="sm"
            placeholder={t('admin.auditActorPlaceholder')}
            value={filterActor}
            onChange={(e) => setFilterActor(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleFilter()}
          />
        </FormControl>
        <FormControl width="200px" flexShrink={0}>
          <FormLabelSmall>{t('admin.auditAction')}</FormLabelSmall>
          <Select
            size="sm"
            value={filterAction}
            onChange={(e) => { setFilterAction(e.target.value); setPage(1) }}
          >
            {ACTION_OPTIONS.map((opt) => (
              <option key={opt.value || 'all'} value={opt.value}>
                {opt.labelKey ? t(opt.labelKey) : opt.label}
              </option>
            ))}
          </Select>
        </FormControl>
        <FormControl width="200px" flexShrink={0}>
          <FormLabelSmall>{t('admin.auditResourceType')}</FormLabelSmall>
          <Select
            size="sm"
            value={filterResourceType}
            onChange={(e) => { setFilterResourceType(e.target.value); setPage(1) }}
          >
            {RESOURCE_OPTIONS.map((opt) => (
              <option key={opt.value || 'all'} value={opt.value}>
                {opt.labelKey ? t(opt.labelKey) : opt.label}
              </option>
            ))}
          </Select>
        </FormControl>
        <Button
          size="sm"
          variant="primary"
          leftIcon={<Icon as={FiFilter} />}
          onClick={handleFilter}
          flexShrink={0}
        >
          {t('common.filter')}
        </Button>
      </HStack>

      {/* 表格 */}
      {loading ? (
        <HStack justify="center" py={8}>
          <Spinner size="sm" />
          <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
        </HStack>
      ) : (
        <Table variant="simple" borderColor="myGray.200" size="sm">
          <Thead>
            <Tr bg="myGray.100">
              <Th>{t('admin.auditTime')}</Th>
              <Th>{t('admin.auditActor')}</Th>
              <Th>{t('admin.auditIP')}</Th>
              <Th>{t('admin.auditAction')}</Th>
              <Th>{t('admin.auditResource')}</Th>
              <Th>{t('admin.auditStatus')}</Th>
              <Th width="60px"></Th>
            </Tr>
          </Thead>
          <Tbody>
            {logs.map((log) => (
              <Tr key={log.id}>
                <Td>
                  <Text fontSize="sm" color="myGray.600">
                    {formatDateTime(log.createdAt)}
                  </Text>
                </Td>
                <Td>
                  <Text fontSize="sm" fontWeight="medium">
                    {log.actorUserName || '-'}
                  </Text>
                </Td>
                <Td>
                  <Text fontSize="sm" fontFamily="mono" color="myGray.600">
                    {log.actorIP || '-'}
                  </Text>
                </Td>
                <Td>
                  <Code fontSize="xs">{log.action}</Code>
                </Td>
                <Td>
                  <HStack spacing={2}>
                    <Badge colorScheme={RESOURCE_COLORS[log.resourceType] || 'gray'} fontSize="xs">
                      {log.resourceType}
                    </Badge>
                    <Text fontSize="sm" fontFamily="mono" color="myGray.600">
                      {log.resourceId}
                    </Text>
                  </HStack>
                </Td>
                <Td>
                  {log.success ? (
                    <Badge colorScheme="green">{t('admin.auditSuccess')}</Badge>
                  ) : (
                    <Badge colorScheme="red">{t('admin.auditFailed')}</Badge>
                  )}
                </Td>
                <Td>
                  <Button
                    size="xs"
                    variant="ghost"
                    leftIcon={<Icon as={FiEye} />}
                    onClick={() => setDetailLog(log)}
                  >
                    {t('common.detail')}
                  </Button>
                </Td>
              </Tr>
            ))}
            {logs.length === 0 && (
              <Tr>
                <Td colSpan={7}>
                  <Text textAlign="center" color="myGray.500" py={4}>
                    {t('admin.auditNoLogs')}
                  </Text>
                </Td>
              </Tr>
            )}
          </Tbody>
        </Table>
      )}

      {/* 分页 */}
      {totalPages > 1 && (
        <HStack justify="center" spacing={4} mt={6}>
          <Button
            size="sm"
            variant="whiteBase"
            leftIcon={<Icon as={FiChevronLeft} />}
            onClick={() => setPage(Math.max(1, page - 1))}
            isDisabled={page === 1}
          >
            {t('common.prevPage')}
          </Button>
          <Text fontSize="sm" color="myGray.600">
            {t('common.pageInfo', { page, totalPages })}
          </Text>
          <Button
            size="sm"
            variant="whiteBase"
            rightIcon={<Icon as={FiChevronRight} />}
            onClick={() => setPage(Math.min(totalPages, page + 1))}
            isDisabled={page === totalPages}
          >
            {t('common.nextPage')}
          </Button>
        </HStack>
      )}

      {/* 详情 Modal */}
      <Modal isOpen={detailLog !== null} onClose={() => setDetailLog(null)} size="lg" blockScrollOnMount={false}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('admin.auditDetail')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody pb={6}>
            {detailLog && (
              <VStack align="stretch" spacing={4}>
                <HStack justify="space-between">
                  <HStack spacing={2}>
                    <Badge colorScheme={RESOURCE_COLORS[detailLog.resourceType] || 'gray'}>
                      {detailLog.resourceType}
                    </Badge>
                    <Code fontSize="sm">{detailLog.action}</Code>
                  </HStack>
                  {detailLog.success ? (
                    <Badge colorScheme="green">{t('admin.auditSuccess')}</Badge>
                  ) : (
                    <Badge colorScheme="red">{t('admin.auditFailed')}</Badge>
                  )}
                </HStack>

                <DetailRow label={t('admin.auditTime')} value={formatDateTime(detailLog.createdAt)} />
                <DetailRow label={t('admin.auditActor')} value={detailLog.actorUserName || '-'} />
                <DetailRow label={t('admin.auditIP')} value={detailLog.actorIP || '-'} mono />
                <DetailRow label={t('admin.auditResource')} value={detailLog.resourceId} mono />

                <Box>
                  <Text fontSize="sm" color="myGray.600" mb={2} fontWeight="medium">
                    {t('admin.auditDetailLabel')}
                  </Text>
                  <Box p={3} bg="myGray.50" borderRadius="md" borderWidth="1px" borderColor="myGray.200">
                    {renderDetail(detailLog.detail)}
                  </Box>
                </Box>
              </VStack>
            )}
          </ModalBody>
        </ModalContent>
      </Modal>
    </Box>
  )
}

// 小标题辅助组件
function FormLabelSmall({ children }: { children: React.ReactNode }) {
  return <Text fontSize="xs" color="myGray.600" mb={1}>{children}</Text>
}

function DetailRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <HStack spacing={3} align="flex-start">
      <Text fontSize="sm" color="myGray.600" minW="100px" fontWeight="medium">
        {label}:
      </Text>
      {mono ? (
        <Text fontSize="sm" fontFamily="mono" wordBreak="break-all">
          {value}
        </Text>
      ) : (
        <Text fontSize="sm" wordBreak="break-all">
          {value}
        </Text>
      )}
    </HStack>
  )
}
