'use client'

import {
  Box,
  VStack,
  HStack,
  FormControl,
  FormLabel,
  Input,
  Textarea,
  Code,
  Button,
  Heading,
  Text,
  Icon,
  Divider,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
} from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { api, SSHKey } from '@/lib/api'
import { FiKey, FiTrash2, FiPlus } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { formatRelativeTime } from '@/lib/time'
import { useI18n } from '@/contexts/I18nContext'

interface SshKeysTabProps {
  isActive: boolean
}

export default function SshKeysTab({ isActive }: SshKeysTabProps) {
  const toast = useGithubToast()
  const { t } = useI18n()
  const [saving, setSaving] = useState(false)
  const [sshKeys, setSshKeys] = useState<SSHKey[]>([])
  const [sshKeyLoading, setSshKeyLoading] = useState(false)
  const [newKeyTitle, setNewKeyTitle] = useState('')
  const [newKeyValue, setNewKeyValue] = useState('')
  const [deletingKeyId, setDeletingKeyId] = useState<number | null>(null)

  // SSH Keys: load when the sshkeys tab is active
  useEffect(() => {
    if (!isActive) return
    setSshKeyLoading(true)
    api.listSSHKeys()
      .then((data) => setSshKeys(data || []))
      .catch((err) => {
        toast({ title: t('settings.loadSSHKeysFailed'), description: err.message, status: 'error', duration: 3000 })
      })
      .finally(() => setSshKeyLoading(false))
  }, [isActive])

  const handleAddSSHKey = async () => {
    if (!newKeyTitle.trim() || !newKeyValue.trim()) {
      toast({ title: t('settings.fillKeyFields'), status: 'warning', duration: 2000 })
      return
    }
    setSaving(true)
    try {
      const created = await api.createSSHKey(newKeyTitle.trim(), newKeyValue.trim())
      setSshKeys([created, ...sshKeys])
      setNewKeyTitle('')
      setNewKeyValue('')
      toast({ title: t('settings.keyAdded'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('settings.addSSHKeyFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setSaving(false)
    }
  }

  const handleConfirmDelete = async () => {
    if (deletingKeyId === null) return
    try {
      await api.deleteSSHKey(deletingKeyId)
      setSshKeys(sshKeys.filter((k) => k.sshKeyId !== deletingKeyId))
      toast({ title: t('settings.sshKeyDeleted'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('settings.deleteFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setDeletingKeyId(null)
    }
  }

  return (
    <>
      <VStack spacing={4} align="stretch">
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
          <VStack spacing={4} align="stretch" mb={4}>
            <Heading size="md">{t('settings.sshTitle')}</Heading>
            <Text fontSize="sm" color="myGray.600">
              {t('settings.sshDesc')}
            </Text>
          </VStack>

          {sshKeyLoading ? (
            <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
          ) : sshKeys.length === 0 ? (
            <Text fontSize="sm" color="myGray.500">{t('settings.noSshKeys')}</Text>
          ) : (
            <VStack spacing={0} align="stretch" divider={<Divider />}>
              {sshKeys.map((key) => (
                <HStack key={key.sshKeyId} justify="space-between" py={3}>
                  <VStack align="start" spacing={1} flex={1} minW={0}>
                    <HStack spacing={2}>
                      <Icon as={FiKey} color="myGray.500" />
                      <Text fontSize="sm" fontWeight="medium">{key.title}</Text>
                    </HStack>
                    <Code fontSize="xs" color="myGray.600" noOfLines={1} w="full">
                      {key.publicKey}
                    </Code>
                    <Text fontSize="xs" color="myGray.500">
                      {t('common.addedAt')} {formatRelativeTime(key.registeredDate, t)}
                    </Text>
                  </VStack>
                  <Button
                    size="sm"
                    variant="ghost"
                    colorScheme="red"
                    leftIcon={<Icon as={FiTrash2} />}
                    onClick={() => setDeletingKeyId(key.sshKeyId)}
                  >
                    {t('settings.deleteKey')}
                  </Button>
                </HStack>
              ))}
            </VStack>
          )}
        </Box>

        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
          <VStack spacing={4} align="stretch" mb={4}>
            <Heading size="md">{t('settings.addNewKey')}</Heading>
          </VStack>
          <VStack spacing={4} align="stretch">
            <FormControl>
              <FormLabel>{t('settings.keyTitle')}</FormLabel>
              <Input
                value={newKeyTitle}
                onChange={(e) => setNewKeyTitle(e.target.value)}
                placeholder={t('settings.keyTitlePlaceholder')}
              />
            </FormControl>
            <FormControl>
              <FormLabel>{t('settings.publicKey')}</FormLabel>
              <Textarea
                value={newKeyValue}
                onChange={(e) => setNewKeyValue(e.target.value)}
                placeholder={t('settings.publicKeyPlaceholder')}
                rows={4}
                fontFamily="mono"
                fontSize="sm"
              />
            </FormControl>
            <HStack justify="flex-end">
              <Button
                variant="primary"
                leftIcon={<Icon as={FiPlus} />}
                onClick={handleAddSSHKey}
                isLoading={saving}
              >
                {t('settings.addKey')}
              </Button>
            </HStack>
          </VStack>
        </Box>
      </VStack>

      <Modal isOpen={deletingKeyId !== null} onClose={() => setDeletingKeyId(null)}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('settings.deleteConfirmTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>{t('settings.deleteConfirmDesc')}</Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => setDeletingKeyId(null)}>
              {t('common.cancel')}
            </Button>
            <Button variant="solid" colorScheme="red" onClick={handleConfirmDelete}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </>
  )
}
