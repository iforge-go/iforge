'use client'

import { Box, Heading, Text, Button, VStack, HStack, Table, Thead, Tbody, Tr, Th, Td, Badge, Avatar, Icon, Modal, ModalOverlay, ModalContent, ModalHeader, ModalBody, ModalFooter, ModalCloseButton, useDisclosure, FormControl, FormLabel, Input, Switch } from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { api, User } from '@/lib/api'
import { FiUsers } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { getLocalizedErrorMessage } from '@/lib/errorMessages'
import { UsernameInput } from '@/components/UsernameInput'
import type { UsernameStatus } from '@/hooks/useUsernameCheck'

export default function AdminUsers() {
  const toast = useGithubToast()
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const { isOpen, onOpen, onClose } = useDisclosure()
  const { isOpen: isEditOpen, onOpen: onEditOpen, onClose: onEditClose } = useDisclosure()
  const [newUser, setNewUser] = useState({
    userName: '',
    password: '',
    fullName: '',
    mailAddress: '',
    isAdmin: false,
  })
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [editForm, setEditForm] = useState({
    fullName: '',
    mailAddress: '',
    isAdmin: false,
    password: '',
  })
  const [deletingUser, setDeletingUser] = useState<User | null>(null)
  const [usernameStatus, setUsernameStatus] = useState<UsernameStatus>('idle')

  const loadUsers = async () => {
    try {
      const data = await api.listUsers()
      setUsers(data)
    } catch (error) {
      console.error('Failed to load users:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadUsers()
  }, [])

  const handleCreateUser = async () => {
    // 用户名必须通过实时校验为"可用"才允许创建
    if (usernameStatus !== 'available') {
      toast({ title: t('auth.usernameNotConfirmed'), status: 'warning', duration: 2000 })
      return
    }
    try {
      await api.createUser(newUser)
      toast({ title: t('admin.userCreated'), status: 'success', duration: 2000 })
      await loadUsers()
      onClose()
      setNewUser({ userName: '', password: '', fullName: '', mailAddress: '', isAdmin: false })
    } catch (error: any) {
      toast({ title: t('admin.createFailed'), description: getLocalizedErrorMessage(error, t), status: 'error', duration: 3000 })
    }
  }

  const handleEditClick = (user: User) => {
    setEditingUser(user)
    setEditForm({
      fullName: user.fullName || '',
      mailAddress: user.mailAddress || '',
      isAdmin: user.isAdmin,
      password: '',
    })
    onEditOpen()
  }

  const handleEditClose = () => {
    // 先触发关闭动画
    onEditClose()
  }

  // Modal 关闭动画完成后清理状态，避免动画期间的重渲染导致抖动
  const handleEditCloseComplete = () => {
    setEditingUser(null)
    setEditForm({ fullName: '', mailAddress: '', isAdmin: false, password: '' })
  }

  // 检查是否是最后一个管理员
  const isLastAdmin = (userName: string) => {
    const adminCount = users.filter(u => u.isAdmin).length
    return adminCount === 1 && users.find(u => u.userName === userName)?.isAdmin
  }

  const handleSaveEdit = async () => {
    if (!editingUser) return
    try {
      const data: { fullName?: string; mailAddress?: string; isAdmin?: boolean; password?: string } = {
        fullName: editForm.fullName,
        mailAddress: editForm.mailAddress,
        isAdmin: editForm.isAdmin,
      }
      if (editForm.password) {
        data.password = editForm.password
      }
      await api.adminUpdateUser(editingUser.userName, data)
      toast({ title: t('admin.userUpdated'), status: 'success', duration: 2000 })
      await loadUsers()
      handleEditClose()
    } catch (error: any) {
      toast({ title: t('admin.updateFailed'), description: getLocalizedErrorMessage(error, t), status: 'error', duration: 3000 })
    }
  }

  const handleConfirmDelete = async () => {
    if (!deletingUser) return
    try {
      await api.adminDeleteUser(deletingUser.userName)
      toast({ title: t('admin.userDeleted'), status: 'success', duration: 2000 })
      await loadUsers()
    } catch (error: any) {
      toast({ title: t('admin.deleteFailed'), description: getLocalizedErrorMessage(error, t), status: 'error', duration: 3000 })
    } finally {
      setDeletingUser(null)
    }
  }

  if (loading) {
    return <Text>{t('common.loading')}</Text>
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
      <HStack justify="space-between" mb={4}>
        <Heading size="md">{t('admin.userList')}</Heading>
        <Button leftIcon={<Icon as={FiUsers} />} variant="primary" size="sm" onClick={onOpen}>
          {t('admin.newUser')}
        </Button>
      </HStack>
      <Table variant="simple" borderColor="myGray.200">
        <Thead>
          <Tr bg="myGray.100">
            <Th>{t('admin.user')}</Th>
            <Th>{t('admin.emailCol')}</Th>
            <Th>{t('admin.role')}</Th>
            <Th>{t('admin.registerTime')}</Th>
            <Th>{t('admin.actions')}</Th>
          </Tr>
        </Thead>
        <Tbody>
          {users.map((user) => (
            <Tr key={user.userName}>
              <Td>
                <HStack spacing={3}>
                  <Avatar size="sm" name={user.fullName || user.userName} src={user.image || undefined} />
                  <VStack align="start" spacing={0}>
                    <Text fontWeight="medium">{user.fullName || user.userName}</Text>
                    <Text fontSize="xs" color="myGray.500">
                      @{user.userName}
                    </Text>
                  </VStack>
                </HStack>
              </Td>
              <Td>{user.mailAddress}</Td>
              <Td>
                <Badge colorScheme={user.isAdmin ? 'red' : 'gray'}>{user.isAdmin ? t('admin.admin') : t('admin.userRole')}</Badge>
              </Td>
              <Td>
                <Text fontSize="sm">{new Date(user.registeredDate).toLocaleDateString(dateLocale)}</Text>
              </Td>
              <Td>
                <HStack spacing={2}>
                  <Button size="xs" variant="whiteBase" onClick={() => handleEditClick(user)}>
                    {t('common.edit')}
                  </Button>
                  {!user.isAdmin && (
                    <Button size="xs" variant="danger" onClick={() => setDeletingUser(user)}>
                      {t('common.delete')}
                    </Button>
                  )}
                </HStack>
              </Td>
            </Tr>
          ))}
        </Tbody>
      </Table>

      {/* 新建用户模态框 */}
      <Modal isOpen={isOpen} onClose={onClose} blockScrollOnMount={false}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('admin.newUser')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <UsernameInput
                value={newUser.userName}
                onChange={(v) => setNewUser({ ...newUser, userName: v })}
                onStatusChange={setUsernameStatus}
                label={t('admin.user')}
                isRequired
              />
              <FormControl isRequired>
                <FormLabel>{t('admin.passwordCol')}</FormLabel>
                <Input
                  type="password"
                  value={newUser.password}
                  onChange={(e) => setNewUser({ ...newUser, password: e.target.value })}
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('admin.fullNameCol')}</FormLabel>
                <Input
                  value={newUser.fullName}
                  onChange={(e) => setNewUser({ ...newUser, fullName: e.target.value })}
                />
              </FormControl>
              <FormControl isRequired>
                <FormLabel>{t('admin.emailCol')}</FormLabel>
                <Input
                  type="email"
                  value={newUser.mailAddress}
                  onChange={(e) => setNewUser({ ...newUser, mailAddress: e.target.value })}
                />
              </FormControl>
              <FormControl display="flex" alignItems="center">
                <FormLabel htmlFor="is-admin" mb="0">
                  {t('admin.adminPrivilege')}
                </FormLabel>
                <Switch
                  id="is-admin"
                  colorScheme="primary"
                  isChecked={newUser.isAdmin}
                  onChange={(e) => setNewUser({ ...newUser, isAdmin: e.target.checked })}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button variant="primary" onClick={handleCreateUser}>
              {t('common.create')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* 编辑用户 Modal */}
      <Modal
        isOpen={isEditOpen}
        onClose={handleEditClose}
        onCloseComplete={handleEditCloseComplete}
        isCentered
        motionPreset="none"
        blockScrollOnMount={false}
      >
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('admin.editUser')} - {editingUser?.userName}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4} align="stretch">
              <FormControl>
                <FormLabel>{t('admin.user')}</FormLabel>
                <Input value={editingUser?.userName || ''} isDisabled />
              </FormControl>
              <FormControl>
                <FormLabel>{t('admin.fullNameCol')}</FormLabel>
                <Input
                  value={editForm.fullName}
                  onChange={(e) => setEditForm({ ...editForm, fullName: e.target.value })}
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('admin.emailCol')}</FormLabel>
                <Input
                  value={editForm.mailAddress}
                  onChange={(e) => setEditForm({ ...editForm, mailAddress: e.target.value })}
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('admin.newPassword')}</FormLabel>
                <Input
                  type="password"
                  value={editForm.password}
                  onChange={(e) => setEditForm({ ...editForm, password: e.target.value })}
                  placeholder={t('admin.newPasswordPlaceholder')}
                />
              </FormControl>
              <FormControl display="flex" alignItems="center">
                <HStack spacing={3}>
                  <Switch
                    isChecked={editForm.isAdmin}
                    isDisabled={isLastAdmin(editingUser?.userName || '')}
                    onChange={(e) => setEditForm({ ...editForm, isAdmin: e.target.checked })}
                  />
                  <Text>{t('admin.adminPrivilege')}</Text>
                  {isLastAdmin(editingUser?.userName || '') && (
                    <Text fontSize="xs" color="orange.500">
                      ({t('admin.lastAdminWarning')})
                    </Text>
                  )}
                </HStack>
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={handleEditClose}>
              {t('common.cancel')}
            </Button>
            <Button variant="primary" onClick={handleSaveEdit}>
              {t('common.save')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* 删除用户确认 Modal */}
      <Modal isOpen={deletingUser !== null} onClose={() => setDeletingUser(null)} blockScrollOnMount={false}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('admin.deleteUser')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>
              {t('admin.deleteUserConfirm', { username: deletingUser?.userName || '' })}
            </Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => setDeletingUser(null)}>
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
