'use client'

import {
  Box,
  Button,
  VStack,
  HStack,
  Text,
  Heading,
  Icon,
  IconButton,
  Input,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  FormControl,
  FormLabel,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  useDisclosure,
  Spinner,
  Code,
  Alert,
  AlertIcon,
} from '@chakra-ui/react'
import { FiPlus, FiTrash2, FiKey, FiEdit2 } from 'react-icons/fi'
import { useState, useEffect, useCallback } from 'react'
import { api, Secret } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useGithubToast } from '@/app/providers'
import { formatRelativeTime } from '@/lib/time'

interface SecretManagerProps {
  owner: string
  repoName: string
}

export default function SecretManager({ owner, repoName }: SecretManagerProps) {
  const { t } = useI18n()
  const toast = useGithubToast()
  const [secrets, setSecrets] = useState<Secret[]>([])
  const [loading, setLoading] = useState(true)
  const { isOpen: isEditOpen, onOpen: onEditOpen, onClose: onEditClose } = useDisclosure()
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()

  const [editKey, setEditKey] = useState('')
  const [editValue, setEditValue] = useState('')
  const [isEditing, setIsEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Secret | null>(null)
  const [deleting, setDeleting] = useState(false)

  const loadSecrets = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await api.listSecrets(owner, repoName)
      setSecrets(resp.items || [])
    } catch (error: any) {
      toast({ title: error.message, status: 'error', duration: 3000 })
    } finally {
      setLoading(false)
    }
  }, [owner, repoName, toast])

  useEffect(() => {
    loadSecrets()
  }, [loadSecrets])

  const openCreate = () => {
    setEditKey('')
    setEditValue('')
    setIsEditing(false)
    onEditOpen()
  }

  const openEdit = (secret: Secret) => {
    setEditKey(secret.key)
    setEditValue('')
    setIsEditing(true)
    onEditOpen()
  }

  const handleSave = async () => {
    if (!editKey.trim() || !editValue.trim()) {
      toast({ title: t('cicd.secretName') + ' / ' + t('cicd.secretValue') + ' required', status: 'error', duration: 2000 })
      return
    }
    setSaving(true)
    try {
      await api.setSecret(owner, repoName, editKey.trim(), editValue)
      toast({ title: t('cicd.secretUpdated', { key: editKey }), status: 'success', duration: 2000 })
      onEditClose()
      loadSecrets()
    } catch (error: any) {
      toast({ title: error.message, status: 'error', duration: 3000 })
    } finally {
      setSaving(false)
    }
  }

  const openDelete = (secret: Secret) => {
    setDeleteTarget(secret)
    onDeleteOpen()
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await api.deleteSecret(owner, repoName, deleteTarget.key)
      toast({ title: t('cicd.secretDeleted', { key: deleteTarget.key }), status: 'success', duration: 2000 })
      onDeleteClose()
      setDeleteTarget(null)
      loadSecrets()
    } catch (error: any) {
      toast({ title: error.message, status: 'error', duration: 3000 })
    } finally {
      setDeleting(false)
    }
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
      <Box px={6} pt={6} pb={2}>
        <HStack spacing={2}>
          <Heading size="md">{t('cicd.secrets')}</Heading>
          <Text fontSize="sm" color="myGray.500">CI/CD</Text>
        </HStack>
        <Text fontSize="sm" color="myGray.600" mt={1}>
          {t('cicd.secrets')} · .iforge-ci.yml
        </Text>
      </Box>
      <Box p={6}>
        {loading ? (
          <HStack>
            <Spinner size="sm" />
            <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
          </HStack>
        ) : secrets.length === 0 ? (
          <VStack spacing={3} py={4}>
            <Text color="myGray.500" fontSize="sm">{t('cicd.noSecrets')}</Text>
            <Button leftIcon={<FiPlus />} variant="primary" size="sm" onClick={openCreate}>
              {t('cicd.addSecret')}
            </Button>
          </VStack>
        ) : (
          <VStack spacing={4} align="stretch">
            <Alert status="info" variant="left-accent" borderRadius="md">
              <AlertIcon />
              <Text fontSize="sm">
                Secrets 在 .iforge-ci.yml 中以 <Code fontSize="xs">${'$'}SECRET_NAME</Code> 形式引用,加密存储(AES-256-GCM)。
              </Text>
            </Alert>
            <Box borderWidth="1px" borderRadius="md" overflow="hidden" borderColor="myGray.200">
              <Table variant="simple" size="sm">
                <Thead bg="myGray.50">
                  <Tr>
                    <Th>{t('cicd.secretName')}</Th>
                    <Th width="160px">{t('common.updatedAt')}</Th>
                    <Th width="120px">{t('common.createdAt')}</Th>
                    <Th width="100px" isNumeric>操作</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {secrets.map((secret) => (
                    <Tr key={secret.id} _hover={{ bg: 'myGray.50' }}>
                      <Td>
                        <HStack spacing={2}>
                          <Icon as={FiKey} color="myGray.500" w={3} h={3} />
                          <Code colorScheme="gray" fontSize="sm">{secret.key}</Code>
                        </HStack>
                      </Td>
                      <Td>
                        <Text fontSize="xs" color="myGray.600">{formatRelativeTime(secret.updatedAt, t)}</Text>
                      </Td>
                      <Td>
                        <Text fontSize="xs" color="myGray.500">{secret.createdBy}</Text>
                      </Td>
                      <Td isNumeric>
                        <HStack spacing={1} justify="flex-end">
                          <IconButton
                            aria-label={t('common.edit')}
                            icon={<Icon as={FiEdit2} />}
                            size="xs"
                            variant="ghost"
                            onClick={() => openEdit(secret)}
                          />
                          <IconButton
                            aria-label={t('common.delete')}
                            icon={<Icon as={FiTrash2} />}
                            size="xs"
                            variant="ghost"
                            colorScheme="red"
                            onClick={() => openDelete(secret)}
                          />
                        </HStack>
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </Box>
            <HStack justify="flex-end">
              <Button leftIcon={<FiPlus />} variant="primary" size="sm" onClick={openCreate}>
                {t('cicd.addSecret')}
              </Button>
            </HStack>
          </VStack>
        )}
      </Box>

      {/* 编辑/新建 Modal */}
      <Modal isOpen={isEditOpen} onClose={onEditClose} size="md">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>
            {isEditing ? t('cicd.editSecret') : t('cicd.addSecret')}
          </ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4} align="stretch">
              <FormControl>
                <FormLabel>{t('cicd.secretName')}</FormLabel>
                <Input
                  value={editKey}
                  onChange={(e) => setEditKey(e.target.value)}
                  placeholder="MY_SECRET"
                  isDisabled={isEditing}
                  fontFamily="Consolas, 'SF Mono', Menlo, monospace"
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('cicd.secretValue')}</FormLabel>
                <Input
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                  placeholder={isEditing ? '••••••••（输入新值以覆盖）' : 'secret-value'}
                  type="password"
                  fontFamily="Consolas, 'SF Mono', Menlo, monospace"
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <HStack spacing={2}>
              <Button variant="ghost" onClick={onEditClose}>{t('common.cancel')}</Button>
              <Button variant="primary" onClick={handleSave} isLoading={saving}>{t('common.save')}</Button>
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
              {deleteTarget && t('cicd.deleteSecretConfirm', { key: deleteTarget.key })}
            </Text>
          </ModalBody>
          <ModalFooter>
            <HStack spacing={2}>
              <Button variant="ghost" onClick={onDeleteClose}>{t('common.cancel')}</Button>
              <Button colorScheme="red" onClick={handleDelete} isLoading={deleting}>{t('common.delete')}</Button>
            </HStack>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  )
}
