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
import { api, GPGKey } from '@/lib/api'
import { FiAward, FiTrash2, FiPlus } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { formatRelativeTime } from '@/lib/time'
import { useI18n } from '@/contexts/I18nContext'

interface GpgKeysTabProps {
  isActive: boolean
}

export default function GpgKeysTab({ isActive }: GpgKeysTabProps) {
  const toast = useGithubToast()
  const { t } = useI18n()
  const [saving, setSaving] = useState(false)

  // GPG Keys state
  const [gpgKeys, setGpgKeys] = useState<GPGKey[]>([])
  const [gpgKeyLoading, setGpgKeyLoading] = useState(false)
  const [newGpgTitle, setNewGpgTitle] = useState('')
  const [newGpgKey, setNewGpgKey] = useState('')
  const [deletingGpgKeyId, setDeletingGpgKeyId] = useState<number | null>(null)

  // GPG Keys: load when the gpgkeys tab is active
  useEffect(() => {
    if (!isActive) return
    setGpgKeyLoading(true)
    api.listGPGKeys()
      .then((data) => setGpgKeys(data || []))
      .catch((err) => {
        toast({ title: t('settings.loadGpgKeysFailed'), description: err.message, status: 'error', duration: 3000 })
      })
      .finally(() => setGpgKeyLoading(false))
  }, [isActive, t, toast])

  const handleAddGPGKey = async () => {
    if (!newGpgTitle.trim() || !newGpgKey.trim()) {
      toast({ title: t('settings.fillGpgFields'), status: 'warning', duration: 2000 })
      return
    }
    setSaving(true)
    try {
      const created = await api.createGPGKey(newGpgTitle.trim(), newGpgKey.trim())
      setGpgKeys([created, ...gpgKeys])
      setNewGpgTitle('')
      setNewGpgKey('')
      toast({ title: t('settings.gpgKeyAdded'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('settings.addGpgKeyFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setSaving(false)
    }
  }

  const handleConfirmDeleteGpgKey = async () => {
    if (deletingGpgKeyId === null) return
    try {
      await api.deleteGPGKey(deletingGpgKeyId)
      setGpgKeys(gpgKeys.filter((k) => k.keyId !== deletingGpgKeyId))
      toast({ title: t('settings.gpgKeyDeleted'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('settings.deleteGpgKeyFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setDeletingGpgKeyId(null)
    }
  }

  return (
    <>
      <VStack spacing={4} align="stretch">
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
          <VStack spacing={4} align="stretch" mb={4}>
            <Heading size="md">{t('settings.gpgKeysTitle')}</Heading>
            <Text fontSize="sm" color="myGray.600">
              {t('settings.gpgKeysDesc')}
            </Text>
          </VStack>

          {gpgKeyLoading ? (
            <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
          ) : gpgKeys.length === 0 ? (
            <Text fontSize="sm" color="myGray.500">{t('settings.noGpgKeys')}</Text>
          ) : (
            <VStack spacing={0} align="stretch" divider={<Divider />}>
              {gpgKeys.map((key) => (
                <HStack key={key.keyId} justify="space-between" py={3}>
                  <VStack align="start" spacing={1} flex={1} minW={0}>
                    <HStack spacing={2}>
                      <Icon as={FiAward} color="myGray.500" />
                      <Text fontSize="sm" fontWeight="medium">{key.title}</Text>
                      {key.gpgKeyId && (
                        <Text fontSize="xs" color="myGray.500" fontFamily="mono">
                          {key.gpgKeyId}
                        </Text>
                      )}
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
                    onClick={() => setDeletingGpgKeyId(key.keyId)}
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
            <Heading size="md">{t('settings.addNewGpgKey')}</Heading>
          </VStack>
          <VStack spacing={4} align="stretch">
            <FormControl>
              <FormLabel>{t('settings.gpgKeyTitle')}</FormLabel>
              <Input
                value={newGpgTitle}
                onChange={(e) => setNewGpgTitle(e.target.value)}
                placeholder={t('settings.gpgKeyTitlePlaceholder')}
              />
            </FormControl>
            <FormControl>
              <FormLabel>{t('settings.gpgPublicKey')}</FormLabel>
              <Textarea
                value={newGpgKey}
                onChange={(e) => setNewGpgKey(e.target.value)}
                placeholder={t('settings.gpgPublicKeyPlaceholder')}
                rows={6}
                fontFamily="mono"
                fontSize="sm"
              />
            </FormControl>
            <HStack justify="flex-end">
              <Button
                variant="primary"
                leftIcon={<Icon as={FiPlus} />}
                onClick={handleAddGPGKey}
                isLoading={saving}
              >
                {t('settings.addGpgKey')}
              </Button>
            </HStack>
          </VStack>
        </Box>
      </VStack>

      <Modal isOpen={deletingGpgKeyId !== null} onClose={() => setDeletingGpgKeyId(null)}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('settings.deleteGpgKeyTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>{t('settings.deleteGpgKeyDesc')}</Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => setDeletingGpgKeyId(null)}>
              {t('common.cancel')}
            </Button>
            <Button variant="solid" colorScheme="red" onClick={handleConfirmDeleteGpgKey}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </>
  )
}
