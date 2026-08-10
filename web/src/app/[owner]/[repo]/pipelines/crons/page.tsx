'use client'

import {
  Box,
  Heading,
  Text,
  VStack,
  HStack,
  Icon,
  Badge,
  Spinner,
  Code,
  IconButton,
  Button,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  FormControl,
  FormLabel,
  FormHelperText,
  Input,
  Select,
  Textarea,
  Switch,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  useDisclosure,
  Alert,
  AlertIcon,
  Tooltip,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useEffect, useState, useCallback } from 'react'
import {
  FiClock,
  FiPlus,
  FiEdit2,
  FiTrash2,
  FiRefreshCw,
  FiCalendar,
  FiGitBranch,
  FiPlay,
  FiGitCommit,
} from 'react-icons/fi'
import { api, CronSchedule } from '@/lib/api'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useI18n } from '@/contexts/I18nContext'
import { useGithubToast } from '@/app/providers'
import { formatRelativeTime } from '@/lib/time'

interface EditForm {
  name: string
  schedule: string
  branch: string
  yamlConfig: string
  enabled: boolean
}

const EMPTY_FORM: EditForm = {
  name: '',
  schedule: '0 2 * * *',
  branch: '',
  yamlConfig: '',
  enabled: true,
}

export default function CronsPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const { branches, userRole } = useRepo()
  const { t } = useI18n()
  const toast = useGithubToast()

  const [crons, setCrons] = useState<CronSchedule[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  const { isOpen: isEditOpen, onOpen: onEditOpen, onClose: onEditClose } = useDisclosure()
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()

  const [form, setForm] = useState<EditForm>(EMPTY_FORM)
  const [isEditing, setIsEditing] = useState(false)
  const [editId, setEditId] = useState<number | null>(null)
  const [saving, setSaving] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<CronSchedule | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [togglingId, setTogglingId] = useState<number | null>(null)

  const canWrite = userRole === 'owner' || userRole === 'member'

  const load = useCallback(async () => {
    try {
      const resp = await api.listCrons(owner, repoName)
      setCrons(resp.items || [])
    } catch (err) {
      console.error('Failed to load crons:', err)
      setCrons([])
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [owner, repoName])

  useEffect(() => {
    load()
  }, [load])

  const handleRefresh = () => {
    setRefreshing(true)
    load()
  }

  const openCreate = () => {
    setForm({
      ...EMPTY_FORM,
      branch: branches[0]?.name || '',
    })
    setIsEditing(false)
    setEditId(null)
    onEditOpen()
  }

  const openEdit = (cron: CronSchedule) => {
    setForm({
      name: cron.name,
      schedule: cron.schedule,
      branch: cron.branch,
      yamlConfig: cron.yamlConfig || '',
      enabled: cron.enabled,
    })
    setIsEditing(true)
    setEditId(cron.id)
    onEditOpen()
  }

  const openDelete = (cron: CronSchedule) => {
    setDeleteTarget(cron)
    onDeleteOpen()
  }

  const handleSave = async () => {
    if (!form.name.trim()) {
      toast({ title: t('cicd.cronNameRequired'), status: 'error', duration: 2000 })
      return
    }
    if (!form.schedule.trim()) {
      toast({ title: t('cicd.cronScheduleRequired'), status: 'error', duration: 2000 })
      return
    }
    if (!form.branch) {
      toast({ title: t('cicd.cronBranchRequired'), status: 'error', duration: 2000 })
      return
    }
    setSaving(true)
    try {
      if (isEditing && editId !== null) {
        await api.updateCron(owner, repoName, editId, {
          schedule: form.schedule.trim(),
          branch: form.branch,
          yamlConfig: form.yamlConfig,
          enabled: form.enabled,
        })
        toast({ title: t('cicd.cronUpdated', { name: form.name }), status: 'success', duration: 2000 })
      } else {
        const created = await api.createCron(owner, repoName, {
          name: form.name.trim(),
          schedule: form.schedule.trim(),
          branch: form.branch,
          yamlConfig: form.yamlConfig,
        })
        toast({ title: t('cicd.cronCreated', { name: created.name }), status: 'success', duration: 2000 })
      }
      onEditClose()
      load()
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 })
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await api.deleteCron(owner, repoName, deleteTarget.id)
      toast({ title: t('cicd.cronDeleted', { name: deleteTarget.name }), status: 'success', duration: 2000 })
      onDeleteClose()
      setDeleteTarget(null)
      load()
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 })
    } finally {
      setDeleting(false)
    }
  }

  const handleToggleEnabled = async (cron: CronSchedule) => {
    setTogglingId(cron.id)
    try {
      await api.updateCron(owner, repoName, cron.id, {
        schedule: cron.schedule,
        branch: cron.branch,
        yamlConfig: cron.yamlConfig || '',
        enabled: !cron.enabled,
      })
      toast({
        title: t('cicd.cronEnabledToggled', {
          name: cron.name,
          state: !cron.enabled ? t('cicd.cronEnabledOn') : t('cicd.cronEnabledOff'),
        }),
        status: 'success',
        duration: 2000,
      })
      load()
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 })
    } finally {
      setTogglingId(null)
    }
  }

  return (
    <VStack spacing={4} align="stretch">
      <HStack justify="space-between">
        <HStack spacing={4}>
          <Heading size="lg" color="myGray.900">
            {t('cicd.crons')}
          </Heading>
          <IconButton
            aria-label="refresh"
            icon={<Icon as={FiRefreshCw} />}
            size="sm"
            variant="ghost"
            onClick={handleRefresh}
            isLoading={refreshing}
          />
        </HStack>
        {canWrite && (
          <Button leftIcon={<Icon as={FiPlus} />} variant="primary" size="sm" onClick={openCreate}>
            {t('cicd.addCron')}
          </Button>
        )}
      </HStack>

      {/* 子导航:Pipelines / Cron Schedules */}
      <HStack spacing={1} borderBottom="2px solid" borderColor="myGray.200">
        <Link href={`/${owner}/${repoName}/pipelines`}>
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<Icon as={FiGitCommit} />}
            fontWeight="medium"
            color="myGray.600"
            borderBottom="2px solid"
            borderColor="transparent"
            mb="-2px"
            _hover={{ bg: 'myGray.50', color: 'myGray.900' }}
          >
            {t('cicd.pipelines')}
          </Button>
        </Link>
        <Link href={`/${owner}/${repoName}/pipelines/crons`}>
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<Icon as={FiCalendar} />}
            fontWeight="semibold"
            color="primary.600"
            borderBottom="2px solid"
            borderColor="primary.600"
            mb="-2px"
            _hover={{ bg: 'myGray.50' }}
          >
            {t('cicd.crons')}
          </Button>
        </Link>
      </HStack>

      {loading ? (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
          <Box py={12}>
            <VStack spacing={4}>
              <Spinner size="md" color="myGray.400" />
              <Text color="myGray.500" fontSize="sm">{t('common.loading')}</Text>
            </VStack>
          </Box>
        </Box>
      ) : crons.length === 0 ? (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
          <Box py={12}>
            <VStack spacing={4}>
              <Icon as={FiCalendar} w={8} h={8} color="myGray.300" />
              <Text color="myGray.500" fontSize="sm">{t('cicd.noCrons')}</Text>
              <Text color="myGray.400" fontSize="xs" maxW="500px" textAlign="center">
                {t('cicd.noCronsHint')}
              </Text>
              {canWrite && (
                <Button leftIcon={<Icon as={FiPlus} />} variant="primary" size="sm" onClick={openCreate}>
                  {t('cicd.addCron')}
                </Button>
              )}
            </VStack>
          </Box>
        </Box>
      ) : (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
          <Table variant="simple" size="sm">
            <Thead bg="myGray.50">
              <Tr>
                <Th>{t('cicd.cronName')}</Th>
                <Th>{t('cicd.cronSchedule')}</Th>
                <Th>{t('cicd.cronBranch')}</Th>
                <Th width="120px">{t('cicd.cronEnabled')}</Th>
                <Th width="160px">{t('cicd.cronNextRun')}</Th>
                <Th width="160px">{t('cicd.cronLastRun')}</Th>
                <Th width="120px">{t('cicd.cronCreator')}</Th>
                <Th width="100px" isNumeric>操作</Th>
              </Tr>
            </Thead>
            <Tbody>
              {crons.map((cron) => (
                <Tr key={cron.id} _hover={{ bg: 'myGray.50' }}>
                  <Td>
                    <HStack spacing={2}>
                      <Icon as={FiClock} color="myGray.500" w={3} h={3} />
                      <Text fontSize="sm" fontWeight="medium" color="myGray.900">{cron.name}</Text>
                    </HStack>
                  </Td>
                  <Td>
                    <Code colorScheme="gray" fontSize="xs" fontFamily="Consolas, 'SF Mono', Menlo, monospace">
                      {cron.schedule}
                    </Code>
                  </Td>
                  <Td>
                    <HStack spacing={1}>
                      <Icon as={FiGitBranch} color="myGray.500" w={3} h={3} />
                      <Text fontSize="sm" color="myGray.700">{cron.branch}</Text>
                    </HStack>
                  </Td>
                  <Td
                    cursor={canWrite ? (togglingId === cron.id ? 'wait' : 'pointer') : 'default'}
                    onClick={canWrite && togglingId !== cron.id ? () => handleToggleEnabled(cron) : undefined}
                    _hover={canWrite ? { bg: 'myGray.50' } : undefined}
                  >
                    {canWrite ? (
                      <Switch
                        size="sm"
                        colorScheme="green"
                        isChecked={cron.enabled}
                        isDisabled={togglingId === cron.id}
                        onChange={() => { /* 由 Td onClick 处理,避免双触发 */ }}
                      />
                    ) : (
                      <Badge colorScheme={cron.enabled ? 'green' : 'gray'} variant="subtle" fontSize="10px">
                        {cron.enabled ? t('cicd.cronEnabled') : t('cicd.cronDisabled')}
                      </Badge>
                    )}
                  </Td>
                  <Td>
                    {cron.nextRunAt ? (
                      <Tooltip label={new Date(cron.nextRunAt).toLocaleString()}>
                        <Text fontSize="xs" color="myGray.600">
                          {formatRelativeTime(cron.nextRunAt, t)}
                        </Text>
                      </Tooltip>
                    ) : (
                      <Text fontSize="xs" color="myGray.400">—</Text>
                    )}
                  </Td>
                  <Td>
                    {cron.lastRunAt ? (
                      <Tooltip label={new Date(cron.lastRunAt).toLocaleString()}>
                        <Text fontSize="xs" color="myGray.600">
                          {formatRelativeTime(cron.lastRunAt, t)}
                        </Text>
                      </Tooltip>
                    ) : (
                      <Text fontSize="xs" color="myGray.400">—</Text>
                    )}
                  </Td>
                  <Td>
                    <Text fontSize="xs" color="myGray.600">{cron.creator}</Text>
                  </Td>
                  <Td isNumeric>
                    {canWrite && (
                      <HStack spacing={1} justify="flex-end">
                        <IconButton
                          aria-label={t('common.edit')}
                          icon={<Icon as={FiEdit2} />}
                          size="xs"
                          variant="ghost"
                          onClick={() => openEdit(cron)}
                        />
                        <IconButton
                          aria-label={t('common.delete')}
                          icon={<Icon as={FiTrash2} />}
                          size="xs"
                          variant="ghost"
                          colorScheme="red"
                          onClick={() => openDelete(cron)}
                        />
                      </HStack>
                    )}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </Box>
      )}

      {/* 新建/编辑 Modal */}
      <Modal isOpen={isEditOpen} onClose={onEditClose} size="lg">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>
            {isEditing ? t('cicd.editCron') : t('cicd.addCron')}
          </ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4} align="stretch">
              <FormControl isRequired isDisabled={isEditing}>
                <FormLabel>{t('cicd.cronName')}</FormLabel>
                <Input
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="nightly-build"
                  fontFamily="Consolas, 'SF Mono', Menlo, monospace"
                />
                {isEditing && (
                  <FormHelperText>{t('cicd.cronName')} 不可修改</FormHelperText>
                )}
              </FormControl>

              <FormControl isRequired>
                <FormLabel>{t('cicd.cronSchedule')}</FormLabel>
                <Input
                  value={form.schedule}
                  onChange={(e) => setForm({ ...form, schedule: e.target.value })}
                  placeholder="0 2 * * *"
                  fontFamily="Consolas, 'SF Mono', Menlo, monospace"
                />
                <FormHelperText>{t('cicd.cronScheduleHint')}</FormHelperText>
              </FormControl>

              <FormControl isRequired>
                <FormLabel>{t('cicd.cronBranch')}</FormLabel>
                <Select
                  value={form.branch}
                  onChange={(e) => setForm({ ...form, branch: e.target.value })}
                  placeholder=""
                >
                  {branches.map((b) => (
                    <option key={b.name} value={b.name}>{b.name}</option>
                  ))}
                </Select>
              </FormControl>

              <FormControl>
                <FormLabel>{t('cicd.cronYamlOverride')}</FormLabel>
                <Textarea
                  value={form.yamlConfig}
                  onChange={(e) => setForm({ ...form, yamlConfig: e.target.value })}
                  placeholder={`stages:\n  - build\nbuild-job:\n  stage: build\n  script:\n    - echo "Hello"`}
                  fontFamily="Consolas, 'SF Mono', Menlo, monospace"
                  fontSize="sm"
                  rows={8}
                />
                <FormHelperText>{t('cicd.cronYamlOverrideHint')}</FormHelperText>
              </FormControl>

              {isEditing && (
                <FormControl display="flex" alignItems="center">
                  <FormLabel mb="0">{t('cicd.cronEnabled')}</FormLabel>
                  <Switch
                    colorScheme="green"
                    isChecked={form.enabled}
                    onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
                  />
                </FormControl>
              )}
            </VStack>
          </ModalBody>
          <ModalFooter>
            <HStack spacing={2}>
              <Button variant="ghost" onClick={onEditClose}>{t('common.cancel')}</Button>
              <Button variant="primary" onClick={handleSave} isLoading={saving}>
                {t('common.save')}
              </Button>
            </HStack>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* 删除确认 Modal */}
      <Modal isOpen={isDeleteOpen} onClose={onDeleteClose} size="sm">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('common.delete')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text fontSize="sm" color="myGray.700">
              {deleteTarget && t('cicd.deleteCronConfirm', { name: deleteTarget.name })}
            </Text>
          </ModalBody>
          <ModalFooter>
            <HStack spacing={2}>
              <Button variant="ghost" onClick={onDeleteClose}>{t('common.cancel')}</Button>
              <Button colorScheme="red" onClick={handleDelete} isLoading={deleting}>
                {t('common.delete')}
              </Button>
            </HStack>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </VStack>
  )
}
