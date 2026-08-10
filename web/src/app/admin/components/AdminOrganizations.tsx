'use client'

import { Box, Heading, Text, Button, VStack, HStack, Table, Thead, Tbody, Tr, Th, Td, Badge, Icon, Modal, ModalOverlay, ModalContent, ModalHeader, ModalBody, ModalFooter, ModalCloseButton, useDisclosure, FormControl, FormLabel, Input } from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { api, Organization } from '@/lib/api'
import { AiOutlineBank } from 'react-icons/ai'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { getLocalizedErrorMessage } from '@/lib/errorMessages'

export default function AdminOrganizations() {
  const toast = useGithubToast()
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'
  const [organizations, setOrganizations] = useState<Organization[]>([])
  const [loading, setLoading] = useState(true)
  const { isOpen, onOpen, onClose } = useDisclosure()
  const [newOrganization, setNewOrganization] = useState({ organizationName: '', description: '' })
  const [editingOrganization, setEditingOrganization] = useState<Organization | null>(null)
  const [editForm, setEditForm] = useState({ description: '', url: '' })
  const [deletingOrganization, setDeletingOrganization] = useState<Organization | null>(null)

  const loadOrganizations = async () => {
    try {
      const data = await api.adminListOrganizations()
      setOrganizations(data)
    } catch (error) {
      console.error('Failed to load organizations:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadOrganizations()
  }, [])

  const handleCreateOrganization = async () => {
    try {
      await api.adminCreateOrganization(newOrganization)
      toast({ title: t('organizations.createSuccess'), status: 'success', duration: 2000 })
      await loadOrganizations()
      onClose()
      setNewOrganization({ organizationName: '', description: '' })
    } catch (error: any) {
      toast({ title: t('organizations.createFailed'), description: getLocalizedErrorMessage(error, t), status: 'error', duration: 3000 })
    }
  }

  const handleEditClick = (organization: Organization) => {
    setEditingOrganization(organization)
    setEditForm({
      description: organization.description || '',
      url: organization.url || '',
    })
  }

  const handleSaveEdit = async () => {
    if (!editingOrganization) return
    try {
      await api.adminUpdateOrganization(editingOrganization.userName, {
        description: editForm.description,
        url: editForm.url,
      })
      toast({ title: t('admin.userUpdated'), status: 'success', duration: 2000 })
      await loadOrganizations()
      setEditingOrganization(null)
    } catch (error: any) {
      toast({ title: t('admin.updateFailed'), description: getLocalizedErrorMessage(error, t), status: 'error', duration: 3000 })
    }
  }

  const handleConfirmDelete = async () => {
    if (!deletingOrganization) return
    try {
      await api.adminDeleteOrganization(deletingOrganization.userName)
      toast({ title: t('admin.userDeleted'), status: 'success', duration: 2000 })
      await loadOrganizations()
    } catch (error: any) {
      toast({ title: t('admin.deleteFailed'), description: getLocalizedErrorMessage(error, t), status: 'error', duration: 3000 })
    } finally {
      setDeletingOrganization(null)
    }
  }

  if (loading) {
    return <Text>{t('common.loading')}</Text>
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
      <HStack justify="space-between" mb={4}>
        <Heading size="md">{t('admin.organizations')}</Heading>
        <Button leftIcon={<Icon as={AiOutlineBank} />} variant="primary" size="sm" onClick={onOpen}>
          {t('organizations.createOrganization')}
        </Button>
      </HStack>
      <Table variant="simple" borderColor="myGray.200">
        <Thead>
          <Tr bg="myGray.100">
            <Th>{t('organizations.organizationName')}</Th>
            <Th>{t('organizations.description')}</Th>
            <Th>{t('admin.registerTime')}</Th>
            <Th>{t('admin.actions')}</Th>
          </Tr>
        </Thead>
        <Tbody>
          {organizations.map((organization) => (
            <Tr key={organization.userName}>
              <Td>
                <HStack spacing={3}>
                  <Badge colorScheme="purple">{t('organizations.organizationBadge')}</Badge>
                  <Text fontWeight="medium">{organization.userName}</Text>
                </HStack>
              </Td>
              <Td>
                <Text fontSize="sm" color="myGray.600" maxW="300px" isTruncated>
                  {organization.description || '-'}
                </Text>
              </Td>
              <Td>
                <Text fontSize="sm">{new Date(organization.registeredDate).toLocaleDateString(dateLocale)}</Text>
              </Td>
              <Td>
                <HStack spacing={2}>
                  <Button size="xs" variant="whiteBase" onClick={() => handleEditClick(organization)}>
                    {t('common.edit')}
                  </Button>
                  <Button size="xs" variant="danger" onClick={() => setDeletingOrganization(organization)}>
                    {t('common.delete')}
                  </Button>
                </HStack>
              </Td>
            </Tr>
          ))}
          {organizations.length === 0 && (
            <Tr>
              <Td colSpan={4}>
                <Text textAlign="center" color="myGray.500" py={4}>
                  {t('organizations.noOrganizations')}
                </Text>
              </Td>
            </Tr>
          )}
        </Tbody>
      </Table>

      {/* 新建组织模态框 */}
      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('organizations.createTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('organizations.organizationName')}</FormLabel>
                <Input
                  value={newOrganization.organizationName}
                  onChange={(e) => setNewOrganization({ ...newOrganization, organizationName: e.target.value })}
                  placeholder={t('organizations.organizationNamePlaceholder')}
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('organizations.description')}</FormLabel>
                <Input
                  value={newOrganization.description}
                  onChange={(e) => setNewOrganization({ ...newOrganization, description: e.target.value })}
                  placeholder={t('organizations.descriptionPlaceholder')}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button variant="primary" onClick={handleCreateOrganization}>
              {t('common.create')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* 编辑组织 Modal */}
      <Modal isOpen={editingOrganization !== null} onClose={() => setEditingOrganization(null)}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('common.edit')} - {editingOrganization?.userName}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4} align="stretch">
              <FormControl>
                <FormLabel>{t('organizations.organizationName')}</FormLabel>
                <Input value={editingOrganization?.userName || ''} isDisabled />
              </FormControl>
              <FormControl>
                <FormLabel>{t('organizations.description')}</FormLabel>
                <Input
                  value={editForm.description}
                  onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
                />
              </FormControl>
              <FormControl>
                <FormLabel>URL</FormLabel>
                <Input
                  value={editForm.url}
                  onChange={(e) => setEditForm({ ...editForm, url: e.target.value })}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => setEditingOrganization(null)}>
              {t('common.cancel')}
            </Button>
            <Button variant="primary" onClick={handleSaveEdit}>
              {t('common.save')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* 删除组织确认 Modal */}
      <Modal isOpen={deletingOrganization !== null} onClose={() => setDeletingOrganization(null)}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('common.delete')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>
              {t('admin.deleteUserConfirm', { username: deletingOrganization?.userName || '' })}
            </Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => setDeletingOrganization(null)}>
              {t('common.cancel')}
            </Button>
            <Button variant="danger" onClick={handleConfirmDelete}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  )
}
