'use client'

import {
  Box,
  Button,
  VStack,
  HStack,
  Text,
  Heading,
  IconButton,
  Badge,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  FormControl,
  FormLabel,
  Input,
  Textarea,
  Switch,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  useDisclosure,
  Tooltip,
  Code,
  Divider,
  Icon,
} from '@chakra-ui/react'
import { FiPlus, FiTrash2, FiKey } from 'react-icons/fi'
import { useState, useEffect } from 'react'
import { api, DeployKey } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useGithubToast } from '@/app/providers'
import { formatRelativeTime } from '@/lib/time'

interface DeployKeyManagerProps {
  owner: string
  repoName: string
}

export default function DeployKeyManager({ owner, repoName }: DeployKeyManagerProps) {
  const { t } = useI18n()
  const toast = useGithubToast()
  const [deployKeys, setDeployKeys] = useState<DeployKey[]>([])
  const [loading, setLoading] = useState(true)
  const { isOpen, onOpen, onClose } = useDisclosure()

  const [newTitle, setNewTitle] = useState('')
  const [newPublicKey, setNewPublicKey] = useState('')
  const [newAllowWrite, setNewAllowWrite] = useState(false)
  const [creating, setCreating] = useState(false)
  const [deletingId, setDeletingId] = useState<number | null>(null)

  useEffect(() => {
    loadDeployKeys()
  }, [owner, repoName])

  const loadDeployKeys = async () => {
    setLoading(true)
    try {
      const data = await api.listDeployKeys(owner, repoName)
      setDeployKeys(data || [])
    } catch (error: any) {
      toast({
        title: t('deployKey.loadFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async () => {
    if (!newTitle.trim() || !newPublicKey.trim()) {
      toast({
        title: t('deployKey.fillFields'),
        status: 'warning',
        duration: 2000,
      })
      return
    }

    setCreating(true)
    try {
      await api.createDeployKey(owner, repoName, {
        title: newTitle.trim(),
        publicKey: newPublicKey.trim(),
        allowWrite: newAllowWrite,
      })
      toast({
        title: t('deployKey.created'),
        status: 'success',
        duration: 2000,
      })
      onClose()
      setNewTitle('')
      setNewPublicKey('')
      setNewAllowWrite(false)
      loadDeployKeys()
    } catch (error: any) {
      toast({
        title: t('deployKey.createFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (id: number) => {
    setDeletingId(id)
    try {
      await api.deleteDeployKey(owner, repoName, id)
      toast({
        title: t('deployKey.deleted'),
        status: 'success',
        duration: 2000,
      })
      loadDeployKeys()
    } catch (error: any) {
      toast({
        title: t('deployKey.deleteFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <>
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
        <Box px={6} pt={6} pb={2}>
          <Heading size="md">{t('deployKey.title')}</Heading>
          <Text fontSize="sm" color="myGray.600" mt={1}>
            {t('deployKey.description')}
          </Text>
        </Box>
        <Box p={6}>
          <VStack spacing={4} align="stretch">
            <HStack justify="space-between">
              <Text fontSize="sm" color="myGray.600">
                {deployKeys.length} {t('deployKey.count')}
              </Text>
              <Button leftIcon={<FiPlus />} size="sm" variant="primary" onClick={onOpen}>
                {t('deployKey.add')}
              </Button>
            </HStack>

            {loading ? (
              <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
            ) : deployKeys.length === 0 ? (
              <Box py={8} textAlign="center">
                <Text color="myGray.500">{t('deployKey.noKeys')}</Text>
              </Box>
            ) : (
              <VStack spacing={0} align="stretch" divider={<Divider />}>
                {deployKeys.map((key) => (
                  <HStack key={key.id} justify="space-between" py={3} align="start">
                    <VStack align="start" spacing={1} flex={1} minW={0}>
                      <HStack spacing={2}>
                        <Icon as={FiKey} color="myGray.500" />
                        <Text fontSize="sm" fontWeight="medium">{key.title}</Text>
                        {key.allowWrite ? (
                          <Badge colorScheme="orange" fontSize="xs">
                            {t('deployKey.write')}
                          </Badge>
                        ) : (
                          <Badge colorScheme="gray" fontSize="xs">
                            {t('deployKey.readOnly')}
                          </Badge>
                        )}
                      </HStack>
                      <Code fontSize="xs" color="myGray.600" noOfLines={1} w="full">
                        {key.publicKey}
                      </Code>
                      <Text fontSize="xs" color="myGray.500">
                        {t('common.addedAt')} {formatRelativeTime(key.createdAt, t)}
                      </Text>
                    </VStack>
                    <Tooltip label={t('common.delete')}>
                      <IconButton
                        aria-label={t('common.delete')}
                        icon={<FiTrash2 />}
                        size="xs"
                        variant="ghost"
                        colorScheme="red"
                        isLoading={deletingId === key.id}
                        onClick={() => handleDelete(key.id)}
                      />
                    </Tooltip>
                  </HStack>
                ))}
              </VStack>
            )}
          </VStack>
        </Box>
      </Box>

      <Modal isOpen={isOpen} onClose={onClose} size="xl">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('deployKey.add')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('deployKey.titleLabel')}</FormLabel>
                <Input
                  value={newTitle}
                  onChange={(e) => setNewTitle(e.target.value)}
                  placeholder={t('deployKey.titlePlaceholder')}
                />
              </FormControl>
              <FormControl isRequired>
                <FormLabel>{t('deployKey.publicKey')}</FormLabel>
                <Textarea
                  value={newPublicKey}
                  onChange={(e) => setNewPublicKey(e.target.value)}
                  placeholder={t('deployKey.publicKeyPlaceholder')}
                  rows={6}
                  fontFamily="mono"
                  fontSize="sm"
                />
              </FormControl>
              <FormControl display="flex" alignItems="center">
                <FormLabel mb={0}>{t('deployKey.allowWrite')}</FormLabel>
                <Switch
                  isChecked={newAllowWrite}
                  onChange={(e) => setNewAllowWrite(e.target.checked)}
                  colorScheme="primary"
                />
                <Text fontSize="xs" color="myGray.500" ml={2}>
                  {t('deployKey.allowWriteDesc')}
                </Text>
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button variant="primary" onClick={handleCreate} isLoading={creating}>
              {t('deployKey.add')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </>
  )
}

