'use client'

import {
  Box,
  Heading,
  Text,
  VStack,
  HStack,
  FormControl,
  FormLabel,
  Input,
  Select,
  Switch,
  Button,
  Icon,
  Badge,
  useToast,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  Spinner,
} from '@chakra-ui/react'
import { useGithubToast } from '@/app/providers'
import { useEffect, useState } from 'react'
import { api, CustomField } from '@/lib/api'
import { FiPlus, FiEdit2, FiTrash2 } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

interface CustomFieldManagerProps {
  owner: string
  repoName: string
}

const FIELD_TYPES = ['text', 'number', 'date', 'boolean', 'select']

export default function CustomFieldManager({ owner, repoName }: CustomFieldManagerProps) {
  const toast = useGithubToast()
  const { t } = useI18n()

  const [fields, setFields] = useState<CustomField[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [deletingId, setDeletingId] = useState<number | null>(null)

  const [form, setForm] = useState({
    fieldName: '',
    fieldType: 'text',
    constraints: '',
    enableForIssues: true,
    enableForMergeRequests: false,
  })

  const loadData = async () => {
    setLoading(true)
    try {
      const list = await api.listCustomFields(owner, repoName).catch(() => [])
      setFields(list || [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [owner, repoName])

  const resetForm = () => {
    setForm({ fieldName: '', fieldType: 'text', constraints: '', enableForIssues: true, enableForMergeRequests: false })
    setEditingId(null)
    setShowForm(false)
  }

  const handleEdit = (f: CustomField) => {
    setForm({
      fieldName: f.fieldName,
      fieldType: f.fieldType,
      constraints: f.constraints || '',
      enableForIssues: f.enableForIssues,
      enableForMergeRequests: f.enableForMergeRequests,
    })
    setEditingId(f.fieldId)
    setShowForm(true)
  }

  const handleSave = async () => {
    if (!form.fieldName.trim() || !form.fieldType) {
      toast({ title: t('repo.fillFieldForm'), status: 'warning', duration: 2000 })
      return
    }
    setSaving(true)
    try {
      const data = {
        fieldName: form.fieldName,
        fieldType: form.fieldType,
        constraints: form.constraints || null,
        enableForIssues: form.enableForIssues,
        enableForMergeRequests: form.enableForMergeRequests,
      }
      if (editingId !== null) {
        await api.updateCustomField(owner, repoName, editingId, {
          fieldName: data.fieldName,
          fieldType: data.fieldType,
          constraints: data.constraints || undefined,
          enableForIssues: data.enableForIssues,
          enableForMergeRequests: data.enableForMergeRequests,
        })
        toast({ title: t('repo.customFieldUpdated'), status: 'success', duration: 2000 })
      } else {
        await api.createCustomField(owner, repoName, data)
        toast({ title: t('repo.customFieldCreated'), status: 'success', duration: 2000 })
      }
      resetForm()
      await loadData()
    } catch (err: any) {
      toast({
        title: editingId !== null ? t('repo.customFieldUpdateFailed') : t('repo.customFieldCreateFailed'),
        description: err.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSaving(false)
    }
  }

  const handleConfirmDelete = async () => {
    if (deletingId === null) return
    try {
      await api.deleteCustomField(owner, repoName, deletingId)
      toast({ title: t('repo.customFieldDeleted'), status: 'success', duration: 2000 })
      await loadData()
    } catch (err: any) {
      toast({ title: t('repo.customFieldDeleteFailed'), description: err.message, status: 'error', duration: 3000 })
    } finally {
      setDeletingId(null)
    }
  }

  const typeLabel = (type: string) => {
    const map: Record<string, string> = {
      text: t('repo.fieldTypeText'),
      number: t('repo.fieldTypeNumber'),
      date: t('repo.fieldTypeDate'),
      boolean: t('repo.fieldTypeBoolean'),
      select: t('repo.fieldTypeSelect'),
    }
    return map[type] || type
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
      <Box px={6} pt={6} pb={2}>
        <HStack justify="space-between">
          <VStack align="start" spacing={1}>
            <Heading size="md">{t('repo.customFieldsTitle')}</Heading>
            <Text fontSize="sm" color="myGray.600">{t('repo.customFieldsDesc')}</Text>
          </VStack>
          {!showForm && (
            <Button size="sm" leftIcon={<Icon as={FiPlus} />} onClick={() => { resetForm(); setShowForm(true) }}>
              {t('common.add')}
            </Button>
          )}
        </HStack>
      </Box>
      <Box p={6}>
        {loading ? (
          <HStack justify="center" py={4}><Spinner size="sm" /></HStack>
        ) : showForm ? (
          <VStack spacing={4} align="stretch">
            <HStack spacing={4}>
              <FormControl isRequired>
                <FormLabel fontSize="sm">{t('repo.fieldName')}</FormLabel>
                <Input
                  value={form.fieldName}
                  onChange={(e) => setForm({ ...form, fieldName: e.target.value })}
                  placeholder={t('repo.fieldNamePlaceholder')}
                  size="sm"
                />
              </FormControl>
              <FormControl isRequired>
                <FormLabel fontSize="sm">{t('repo.fieldType')}</FormLabel>
                <Select
                  value={form.fieldType}
                  onChange={(e) => setForm({ ...form, fieldType: e.target.value })}
                  size="sm"
                >
                  {FIELD_TYPES.map((type) => (
                    <option key={type} value={type}>{typeLabel(type)}</option>
                  ))}
                </Select>
              </FormControl>
            </HStack>
            <FormControl>
              <FormLabel fontSize="sm">{t('repo.fieldConstraints')}</FormLabel>
              <Input
                value={form.constraints}
                onChange={(e) => setForm({ ...form, constraints: e.target.value })}
                placeholder={t('repo.fieldConstraintsPlaceholder')}
                size="sm"
              />
            </FormControl>
            <HStack spacing={6}>
              <FormControl display="flex" alignItems="center">
                <FormLabel mb={0} fontSize="sm" mr={3}>{t('repo.enableForIssues')}</FormLabel>
                <Switch
                  isChecked={form.enableForIssues}
                  onChange={(e) => setForm({ ...form, enableForIssues: e.target.checked })}
                  colorScheme="primary"
                />
              </FormControl>
              <FormControl display="flex" alignItems="center">
                <FormLabel mb={0} fontSize="sm" mr={3}>{t('repo.enableForMergeRequests')}</FormLabel>
                <Switch
                  isChecked={form.enableForMergeRequests}
                  onChange={(e) => setForm({ ...form, enableForMergeRequests: e.target.checked })}
                  colorScheme="primary"
                />
              </FormControl>
            </HStack>
            <HStack spacing={2} justify="flex-end">
              <Button size="sm" variant="ghost" onClick={resetForm}>{t('common.cancel')}</Button>
              <Button size="sm" variant="primary" onClick={handleSave} isLoading={saving}>
                {editingId !== null ? t('common.update') : t('common.create')}
              </Button>
            </HStack>
          </VStack>
        ) : fields.length === 0 ? (
          <Text fontSize="sm" color="myGray.500">{t('repo.noCustomFields')}</Text>
        ) : (
          <VStack spacing={2} align="stretch">
            {fields.map((f) => (
              <HStack
                key={f.fieldId}
                spacing={3}
                p={3}
                borderWidth="1px"
                borderRadius="md"
                borderColor="myGray.200"
                _hover={{ bg: 'myGray.50' }}
              >
                <VStack align="start" spacing={1} flex={1} minW={0}>
                  <HStack spacing={2}>
                    <Text fontSize="sm" fontWeight="medium" color="myGray.900">{f.fieldName}</Text>
                    <Badge fontSize="xs" colorScheme="blue">{typeLabel(f.fieldType)}</Badge>
                    {f.enableForIssues && <Badge fontSize="xs" colorScheme="green">Issue</Badge>}
                    {f.enableForMergeRequests && <Badge fontSize="xs" colorScheme="purple">MR</Badge>}
                  </HStack>
                  {f.constraints && (
                    <Text fontSize="xs" color="myGray.500" noOfLines={1}>{f.constraints}</Text>
                  )}
                </VStack>
                <HStack spacing={1}>
                  <Button size="xs" variant="ghost" leftIcon={<Icon as={FiEdit2} />} onClick={() => handleEdit(f)}>
                    {t('common.edit')}
                  </Button>
                  <Button size="xs" variant="ghost" colorScheme="red" leftIcon={<Icon as={FiTrash2} />} onClick={() => setDeletingId(f.fieldId)}>
                    {t('common.delete')}
                  </Button>
                </HStack>
              </HStack>
            ))}
          </VStack>
        )}
      </Box>

      <Modal isOpen={deletingId !== null} onClose={() => setDeletingId(null)}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('repo.deleteCustomField')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody><Text>{t('repo.deleteCustomFieldConfirm')}</Text></ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => setDeletingId(null)}>{t('common.cancel')}</Button>
            <Button variant="solid" colorScheme="red" onClick={handleConfirmDelete}>{t('common.delete')}</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  )
}
