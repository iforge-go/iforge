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
  Textarea,
  Button,
  Icon,
  Badge,
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
import { useEffect, useState, useCallback } from 'react'
import { api, Priority } from '@/lib/api'
import { FiPlus, FiEdit2, FiTrash2, FiStar } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

interface PriorityManagerProps {
  owner: string
  repoName: string
}

const PRESET_COLORS = [
  '#dc2626', '#ea580c', '#d97706', '#ca8a04', '#65a30d',
  '#16a34a', '#0891b2', '#0284c7', '#2563eb', '#4f46e5',
  '#7c3aed', '#c026d3', '#db2777', '#e11d48', '#6b7280',
]

export default function PriorityManager({ owner, repoName }: PriorityManagerProps) {
  const toast = useGithubToast()
  const { t } = useI18n()

  const [priorities, setPriorities] = useState<Priority[]>([])
  const [defaultId, setDefaultId] = useState<number | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [deletingId, setDeletingId] = useState<number | null>(null)

  const [form, setForm] = useState({
    priorityName: '',
    description: '',
    color: PRESET_COLORS[0],
  })

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const [list, def] = await Promise.all([
        api.listPriorities(owner, repoName).catch(() => []),
        api.getDefaultPriority(owner, repoName)
          .then((res) => res.priorityId)
          .catch(() => null),
      ])
      setPriorities(list || [])
      setDefaultId(def)
    } finally {
      setLoading(false)
    }
  }, [owner, repoName])

  useEffect(() => {
    loadData()
  }, [loadData])

  const resetForm = () => {
    setForm({ priorityName: '', description: '', color: PRESET_COLORS[0] })
    setEditingId(null)
    setShowForm(false)
  }

  const handleEdit = (p: Priority) => {
    setForm({
      priorityName: p.priorityName,
      description: p.description || '',
      color: p.color,
    })
    setEditingId(p.priorityId)
    setShowForm(true)
  }

  const handleSave = async () => {
    if (!form.priorityName.trim() || !form.color) {
      toast({ title: t('repo.fillPriorityFields'), status: 'warning', duration: 2000 })
      return
    }
    setSaving(true)
    try {
      if (editingId !== null) {
        await api.updatePriority(owner, repoName, editingId, {
          priorityName: form.priorityName,
          description: form.description,
          color: form.color,
        })
        toast({ title: t('repo.priorityUpdated'), status: 'success', duration: 2000 })
      } else {
        await api.createPriority(owner, repoName, {
          priorityName: form.priorityName,
          description: form.description,
          color: form.color,
        })
        toast({ title: t('repo.priorityCreated'), status: 'success', duration: 2000 })
      }
      resetForm()
      await loadData()
    } catch (err: any) {
      toast({
        title: editingId !== null ? t('repo.priorityUpdateFailed') : t('repo.priorityCreateFailed'),
        description: err.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSaving(false)
    }
  }

  const handleSetDefault = async (id: number) => {
    try {
      await api.setDefaultPriority(owner, repoName, id)
      setDefaultId(id)
      toast({ title: t('repo.priorityUpdated'), status: 'success', duration: 2000 })
    } catch (err: any) {
      toast({ title: t('repo.priorityUpdateFailed'), description: err.message, status: 'error', duration: 3000 })
    }
  }

  const handleConfirmDelete = async () => {
    if (deletingId === null) return
    try {
      await api.deletePriority(owner, repoName, deletingId)
      toast({ title: t('repo.priorityDeleted'), status: 'success', duration: 2000 })
      if (defaultId === deletingId) setDefaultId(null)
      await loadData()
    } catch (err: any) {
      toast({ title: t('repo.priorityDeleteFailed'), description: err.message, status: 'error', duration: 3000 })
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
      <Box px={6} pt={6} pb={2}>
        <HStack justify="space-between">
          <VStack align="start" spacing={1}>
            <Heading size="md">{t('repo.prioritiesTitle')}</Heading>
            <Text fontSize="sm" color="myGray.600">
              {t('repo.prioritiesDesc')}
            </Text>
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
            <FormControl isRequired>
              <FormLabel fontSize="sm">{t('repo.priorityName')}</FormLabel>
              <Input
                value={form.priorityName}
                onChange={(e) => setForm({ ...form, priorityName: e.target.value })}
                placeholder={t('repo.priorityNamePlaceholder')}
                size="sm"
              />
            </FormControl>
            <FormControl>
              <FormLabel fontSize="sm">{t('repo.priorityDescription')}</FormLabel>
              <Textarea
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
                size="sm"
                rows={2}
              />
            </FormControl>
            <FormControl isRequired>
              <FormLabel fontSize="sm">{t('repo.priorityColor')}</FormLabel>
              <HStack spacing={2} flexWrap="wrap">
                {PRESET_COLORS.map((color) => (
                  <Box
                    key={color}
                    w={7}
                    h={7}
                    borderRadius="md"
                    bg={color}
                    cursor="pointer"
                    borderWidth="2px"
                    borderColor={form.color === color ? 'myGray.900' : 'transparent'}
                    onClick={() => setForm({ ...form, color })}
                    _hover={{ transform: 'scale(1.1)' }}
                    transition="transform 0.1s"
                  />
                ))}
                <Input
                  type="color"
                  value={form.color}
                  onChange={(e) => setForm({ ...form, color: e.target.value })}
                  w={10}
                  h={7}
                  p={0}
                  border="none"
                />
              </HStack>
            </FormControl>
            <HStack spacing={2} justify="flex-end">
              <Button size="sm" variant="ghost" onClick={resetForm}>
                {t('common.cancel')}
              </Button>
              <Button size="sm" variant="primary" onClick={handleSave} isLoading={saving}>
                {editingId !== null ? t('common.update') : t('common.create')}
              </Button>
            </HStack>
          </VStack>
        ) : priorities.length === 0 ? (
          <Text fontSize="sm" color="myGray.500">{t('repo.noPriorities')}</Text>
        ) : (
          <VStack spacing={2} align="stretch">
            {priorities.map((p) => (
              <HStack
                key={p.priorityId}
                spacing={3}
                p={3}
                borderWidth="1px"
                borderRadius="md"
                borderColor="myGray.200"
                _hover={{ bg: 'myGray.50' }}
              >
                <Box w={4} h={4} borderRadius="sm" bg={p.color} flexShrink={0} />
                <VStack align="start" spacing={0} flex={1} minW={0}>
                  <HStack spacing={2}>
                    <Text fontSize="sm" fontWeight="medium" color="myGray.900">
                      {p.priorityName}
                    </Text>
                    {defaultId === p.priorityId && (
                      <Badge fontSize="xs" colorScheme="yellow" variant="subtle">
                        <HStack spacing={1}>
                          <Icon as={FiStar} w={2.5} h={2.5} />
                          <Text>{t('repo.priorityDefault')}</Text>
                        </HStack>
                      </Badge>
                    )}
                  </HStack>
                  {p.description && (
                    <Text fontSize="xs" color="myGray.500" noOfLines={1}>
                      {p.description}
                    </Text>
                  )}
                </VStack>
                <HStack spacing={1}>
                  {defaultId !== p.priorityId && (
                    <Button
                      size="xs"
                      variant="ghost"
                      onClick={() => handleSetDefault(p.priorityId)}
                    >
                      {t('repo.setAsDefault')}
                    </Button>
                  )}
                  <Button
                    size="xs"
                    variant="ghost"
                    leftIcon={<Icon as={FiEdit2} />}
                    onClick={() => handleEdit(p)}
                  >
                    {t('common.edit')}
                  </Button>
                  <Button
                    size="xs"
                    variant="ghost"
                    colorScheme="red"
                    leftIcon={<Icon as={FiTrash2} />}
                    onClick={() => setDeletingId(p.priorityId)}
                  >
                    {t('common.delete')}
                  </Button>
                </HStack>
              </HStack>
            ))}
          </VStack>
        )}
      </Box>

      {/* Delete confirmation */}
      <Modal isOpen={deletingId !== null} onClose={() => setDeletingId(null)}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('repo.deletePriority')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>{t('repo.deletePriorityConfirm')}</Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => setDeletingId(null)}>
              {t('common.cancel')}
            </Button>
            <Button variant="solid" colorScheme="red" onClick={handleConfirmDelete}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  )
}
