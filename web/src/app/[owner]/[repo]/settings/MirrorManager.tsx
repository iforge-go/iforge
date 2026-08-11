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
  NumberInput,
  NumberInputField,
  NumberInputStepper,
  NumberIncrementStepper,
  NumberDecrementStepper,
  Switch,
  Select,
  Button,
  Icon,
  Badge,
  Spinner,
} from '@chakra-ui/react'
import { useGithubToast } from '@/app/providers'
import { useEffect, useState, useCallback } from 'react'
import { api, RepositoryMirror } from '@/lib/api'
import { FiRefreshCw, FiTrash2, FiSave, FiPlus } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

interface MirrorManagerProps {
  owner: string
  repoName: string
}

export default function MirrorManager({ owner, repoName }: MirrorManagerProps) {
  const toast = useGithubToast()
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'

  const [mirror, setMirror] = useState<RepositoryMirror | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [showForm, setShowForm] = useState(false)

  const [form, setForm] = useState({
    mirrorUrl: '',
    syncInterval: 60,
    enabled: true,
    syncOnPush: false,
    authentication: 'none',
    username: '',
    password: '',
    sshKey: '',
  })

  const loadMirror = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.getMirror(owner, repoName)
      if (data) {
        setMirror(data)
        setForm({
          mirrorUrl: data.mirrorUrl,
          syncInterval: data.syncInterval,
          enabled: data.enabled,
          syncOnPush: data.syncOnPush,
          authentication: data.authentication || 'none',
          username: data.username || '',
          password: '',
          sshKey: data.sshKey || '',
        })
        setShowForm(false)
      } else {
        setMirror(null)
        setShowForm(true)
      }
    } catch {
      setMirror(null)
      setShowForm(true)
    } finally {
      setLoading(false)
    }
  }, [owner, repoName])

  useEffect(() => {
    loadMirror()
  }, [loadMirror])

  const handleSave = async () => {
    if (!form.mirrorUrl.trim()) {
      toast({ title: t('repo.mirrorUrl'), status: 'warning', duration: 2000 })
      return
    }
    setSaving(true)
    try {
      const payload: Partial<RepositoryMirror> = {
        mirrorUrl: form.mirrorUrl,
        syncInterval: form.syncInterval,
        enabled: form.enabled,
        syncOnPush: form.syncOnPush,
        authentication: form.authentication,
        username: form.authentication === 'basic' ? form.username : null,
        password: form.authentication === 'basic' ? form.password : null,
        sshKey: form.authentication === 'ssh' ? form.sshKey : null,
      }
      if (mirror) {
        const updated = await api.updateMirror(owner, repoName, payload)
        setMirror(updated)
        setShowForm(false)
      } else {
        const created = await api.createMirror(owner, repoName, payload)
        setMirror(created)
        setShowForm(false)
      }
      toast({ title: t('repo.mirrorSaved'), status: 'success', duration: 2000 })
    } catch (err: any) {
      toast({
        title: t('repo.mirrorSaveFailed'),
        description: err.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSaving(false)
    }
  }

  const handleSync = async () => {
    setSyncing(true)
    try {
      await api.syncMirror(owner, repoName)
      toast({ title: t('repo.syncSuccess'), status: 'success', duration: 2000 })
      await loadMirror()
    } catch (err: any) {
      toast({
        title: t('repo.syncFailed'),
        description: err.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSyncing(false)
    }
  }

  const handleConfirmDelete = async () => {
    setDeleting(true)
    try {
      await api.deleteMirror(owner, repoName)
      setMirror(null)
      setForm({
        mirrorUrl: '',
        syncInterval: 60,
        enabled: true,
        syncOnPush: false,
        authentication: 'none',
        username: '',
        password: '',
        sshKey: '',
      })
      setShowForm(true)
      toast({ title: t('repo.mirrorDeleted'), status: 'success', duration: 2000 })
    } catch (err: any) {
      toast({
        title: t('repo.mirrorDeleteFailed'),
        description: err.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setDeleting(false)
    }
  }

  const formatSyncDate = (dateStr?: string) => {
    if (!dateStr) return t('repo.neverSynced')
    const d = new Date(dateStr)
    if (isNaN(d.getTime())) return t('repo.neverSynced')
    return d.toLocaleString(dateLocale)
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
      <Box px={6} pt={6} pb={2}>
        <Heading size="md">{t('repo.mirrorTitle')}</Heading>
        <Text fontSize="sm" color="myGray.600" mt={1}>
          {t('repo.mirrorDesc')}
        </Text>
      </Box>
      <Box p={6}>
        {loading ? (
          <HStack justify="center" py={4}>
            <Spinner size="sm" />
          </HStack>
        ) : mirror && !showForm ? (
          <VStack spacing={4} align="stretch">
            <HStack spacing={3} flexWrap="wrap">
              <Badge colorScheme={mirror.enabled ? 'green' : 'gray'}>
                {mirror.enabled ? t('common.yes') : t('common.no')}
              </Badge>
              <Text fontSize="sm" fontFamily="mono" color="myGray.900">
                {mirror.mirrorUrl}
              </Text>
            </HStack>

            <HStack spacing={6} fontSize="sm" color="myGray.600">
              <HStack spacing={1}>
                <Text fontWeight="medium">{t('repo.syncInterval')}:</Text>
                <Text>{mirror.syncInterval} min</Text>
              </HStack>
              <HStack spacing={1}>
                <Text fontWeight="medium">{t('repo.lastSync')}:</Text>
                <Text>{formatSyncDate(mirror.lastSyncDate)}</Text>
              </HStack>
              <HStack spacing={1}>
                <Text fontWeight="medium">{t('repo.nextSync')}:</Text>
                <Text>{formatSyncDate(mirror.nextSyncDate)}</Text>
              </HStack>
            </HStack>

            {mirror.authentication !== 'none' && (
              <HStack spacing={2} fontSize="sm" color="myGray.600">
                <Text fontWeight="medium">{t('repo.authentication')}:</Text>
                <Badge fontSize="xs">
                  {mirror.authentication === 'basic' ? t('repo.authBasic') : t('repo.authSSH')}
                </Badge>
              </HStack>
            )}

            <HStack spacing={2} justify="flex-end">
              <Button
                size="sm"
                leftIcon={<Icon as={FiRefreshCw} />}
                onClick={handleSync}
                isLoading={syncing}
              >
                {t('repo.syncNow')}
              </Button>
              <Button
                size="sm"
                variant="outline"
                leftIcon={<Icon as={FiSave} />}
                onClick={() => setShowForm(true)}
              >
                {t('common.edit')}
              </Button>
              <Button
                size="sm"
                variant="outline"
                colorScheme="red"
                leftIcon={<Icon as={FiTrash2} />}
                onClick={handleConfirmDelete}
                isLoading={deleting}
              >
                {t('repo.deleteMirror')}
              </Button>
            </HStack>
          </VStack>
        ) : (
          <VStack spacing={4} align="stretch">
            {!mirror && (
              <HStack spacing={2}>
                <Icon as={FiPlus} color="myGray.400" />
                <Text fontSize="sm" color="myGray.500">{t('repo.noMirror')}</Text>
              </HStack>
            )}
            <FormControl isRequired>
              <FormLabel fontSize="sm">{t('repo.mirrorUrl')}</FormLabel>
              <Input
                value={form.mirrorUrl}
                onChange={(e) => setForm({ ...form, mirrorUrl: e.target.value })}
                placeholder={t('repo.mirrorUrlPlaceholder')}
                size="sm"
              />
            </FormControl>
            <HStack spacing={4}>
              <FormControl>
                <FormLabel fontSize="sm">{t('repo.syncInterval')}</FormLabel>
                <NumberInput
                  value={form.syncInterval}
                  onChange={(_, v) => setForm({ ...form, syncInterval: v || 60 })}
                  min={5}
                  max={10080}
                  size="sm"
                >
                  <NumberInputField />
                  <NumberInputStepper>
                    <NumberIncrementStepper />
                    <NumberDecrementStepper />
                  </NumberInputStepper>
                </NumberInput>
              </FormControl>
              <FormControl>
                <FormLabel fontSize="sm">{t('repo.authentication')}</FormLabel>
                <Select
                  value={form.authentication}
                  onChange={(e) => setForm({ ...form, authentication: e.target.value })}
                  size="sm"
                >
                  <option value="none">{t('repo.authNone')}</option>
                  <option value="basic">{t('repo.authBasic')}</option>
                  <option value="ssh">{t('repo.authSSH')}</option>
                </Select>
              </FormControl>
            </HStack>
            {form.authentication === 'basic' && (
              <HStack spacing={4}>
                <FormControl>
                  <FormLabel fontSize="sm">{t('repo.mirrorUsername')}</FormLabel>
                  <Input
                    value={form.username}
                    onChange={(e) => setForm({ ...form, username: e.target.value })}
                    size="sm"
                  />
                </FormControl>
                <FormControl>
                  <FormLabel fontSize="sm">{t('repo.mirrorPassword')}</FormLabel>
                  <Input
                    type="password"
                    value={form.password}
                    onChange={(e) => setForm({ ...form, password: e.target.value })}
                    placeholder={mirror ? '••••••••' : ''}
                    size="sm"
                  />
                </FormControl>
              </HStack>
            )}
            {form.authentication === 'ssh' && (
              <FormControl>
                <FormLabel fontSize="sm">{t('repo.mirrorSshKey')}</FormLabel>
                <Input
                  as="textarea"
                  value={form.sshKey}
                  onChange={(e) => setForm({ ...form, sshKey: e.target.value })}
                  placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                  size="sm"
                  h="80px"
                  fontFamily="mono"
                />
              </FormControl>
            )}
            <HStack spacing={6}>
              <FormControl display="flex" alignItems="center">
                <FormLabel mb={0} fontSize="sm" mr={3}>{t('repo.mirrorEnabled')}</FormLabel>
                <Switch
                  isChecked={form.enabled}
                  onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
                  colorScheme="primary"
                />
              </FormControl>
              <FormControl display="flex" alignItems="center">
                <FormLabel mb={0} fontSize="sm" mr={3}>{t('repo.syncOnPush')}</FormLabel>
                <Switch
                  isChecked={form.syncOnPush}
                  onChange={(e) => setForm({ ...form, syncOnPush: e.target.checked })}
                  colorScheme="primary"
                />
              </FormControl>
            </HStack>
            <HStack justify="flex-end" spacing={2}>
              {mirror && (
                <Button size="sm" variant="ghost" onClick={() => setShowForm(false)}>
                  {t('common.cancel')}
                </Button>
              )}
              <Button
                size="sm"
                variant="primary"
                leftIcon={<Icon as={FiSave} />}
                onClick={handleSave}
                isLoading={saving}
              >
                {mirror ? t('repo.updateMirror') : t('repo.createMirror')}
              </Button>
            </HStack>
          </VStack>
        )}
      </Box>
    </Box>
  )
}
