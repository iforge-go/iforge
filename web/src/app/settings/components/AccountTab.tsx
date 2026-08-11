'use client'

import {
  Box,
  VStack,
  HStack,
  FormControl,
  FormLabel,
  Input,
  Button,
  Heading,
  Text,
  Icon,
  IconButton,
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
import { api, User, ExtraMailAddress } from '@/lib/api'
import { FiMail, FiTrash2, FiPlus } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { getLocalizedErrorMessage } from '@/lib/errorMessages'

interface AccountTabProps {
  user: User
  isActive: boolean
}

export default function AccountTab({ user, isActive }: AccountTabProps) {
  const toast = useGithubToast()
  const { t } = useI18n()
  const [saving, setSaving] = useState(false)
  const [passwordForm, setPasswordForm] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  })

  // Extra Emails state
  const [extraEmails, setExtraEmails] = useState<ExtraMailAddress[]>([])
  const [newEmail, setNewEmail] = useState('')
  const [deletingEmail, setDeletingEmail] = useState<string | null>(null)
  const [, setEmailLoading] = useState(false)

  // Extra Emails: load when the account tab is active
  useEffect(() => {
    if (!isActive) return
    setEmailLoading(true)
    api.listExtraEmails()
      .then((res) => setExtraEmails(res.addresses || []))
      .catch((err) => {
        toast({ title: t('settings.addEmailFailed'), description: getLocalizedErrorMessage(err, t), status: 'error', duration: 3000 })
      })
      .finally(() => setEmailLoading(false))
  }, [isActive, t, toast])

  const handlePasswordChange = async () => {
    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      toast({ title: t('settings.passwordMismatch'), status: 'warning', duration: 2000 })
      return
    }
    if (passwordForm.newPassword.length < 6) {
      toast({ title: t('settings.passwordTooShort'), status: 'warning', duration: 2000 })
      return
    }
    setSaving(true)
    try {
      await api.changePassword(passwordForm.currentPassword, passwordForm.newPassword)
      toast({ title: t('settings.passwordChanged'), status: 'success', duration: 2000 })
      setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' })
    } catch (error: any) {
      toast({ title: t('settings.passwordChangeFailed'), description: getLocalizedErrorMessage(error, t), status: 'error', duration: 3000 })
    } finally {
      setSaving(false)
    }
  }

  const handleAddEmail = async () => {
    const email = newEmail.trim()
    if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      toast({ title: t('settings.invalidEmail'), status: 'warning', duration: 2000 })
      return
    }
    if (email === user?.mailAddress || extraEmails.some((e) => e.mailAddress === email)) {
      toast({ title: t('settings.emailExists'), status: 'warning', duration: 2000 })
      return
    }
    setSaving(true)
    try {
      await api.addExtraEmail(email)
      setExtraEmails([...extraEmails, { userName: user!.userName, mailAddress: email }])
      setNewEmail('')
      toast({ title: t('settings.addEmailSuccess'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('settings.addEmailFailed'), description: getLocalizedErrorMessage(error, t), status: 'error', duration: 3000 })
    } finally {
      setSaving(false)
    }
  }

  const handleConfirmDeleteEmail = async () => {
    if (!deletingEmail) return
    try {
      await api.deleteExtraEmail(deletingEmail)
      setExtraEmails(extraEmails.filter((e) => e.mailAddress !== deletingEmail))
      toast({ title: t('settings.emailDeleted'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('settings.emailDeleteFailed'), description: getLocalizedErrorMessage(error, t), status: 'error', duration: 3000 })
    } finally {
      setDeletingEmail(null)
    }
  }

  return (
    <>
      <VStack spacing={4} align="stretch">
        {/* 密码 Card */}
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
          <VStack spacing={4} align="stretch" mb={4}>
            <Heading size="md">{t('settings.changePasswordTitle')}</Heading>
            <Text fontSize="sm" color="myGray.600">
              {t('settings.changePasswordDesc')}
            </Text>
          </VStack>
          <VStack spacing={5}>
            <FormControl>
              <FormLabel>{t('settings.currentPassword')}</FormLabel>
              <Input
                type="password"
                value={passwordForm.currentPassword}
                onChange={(e) => setPasswordForm({ ...passwordForm, currentPassword: e.target.value })}
              />
            </FormControl>

            <FormControl>
              <FormLabel>{t('settings.newPassword')}</FormLabel>
              <Input
                type="password"
                value={passwordForm.newPassword}
                onChange={(e) => setPasswordForm({ ...passwordForm, newPassword: e.target.value })}
              />
              <Text fontSize="xs" color="myGray.500" mt={1}>
                {t('settings.newPasswordHint')}
              </Text>
            </FormControl>

            <FormControl>
              <FormLabel>{t('settings.confirmPassword')}</FormLabel>
              <Input
                type="password"
                value={passwordForm.confirmPassword}
                onChange={(e) => setPasswordForm({ ...passwordForm, confirmPassword: e.target.value })}
              />
            </FormControl>

            <HStack justify="flex-end" w="full">
              <Button variant="primary" onClick={handlePasswordChange} isLoading={saving}>
                {t('settings.changePassword')}
              </Button>
            </HStack>
          </VStack>
        </Box>

        {/* 邮箱 Card */}
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
          <VStack spacing={4} align="stretch" mb={4}>
            <Heading size="md">{t('settings.emailTitle')}</Heading>
            <Text fontSize="sm" color="myGray.600">
              {t('settings.emailDesc')}
            </Text>
          </VStack>
          <VStack spacing={4} align="stretch">
            <HStack justify="space-between" p={3} border="1px solid" borderColor="myGray.200" borderRadius="md">
              <HStack spacing={3}>
                <Icon as={FiMail} color="myGray.500" />
                <Text>{user.mailAddress}</Text>
                <Text fontSize="xs" color="white" bg="primary.500" px={2} py={0.5} borderRadius="md">
                  {t('settings.primary')}
                </Text>
              </HStack>
              <Text fontSize="xs" color="green.500">
                {t('settings.verified')}
              </Text>
            </HStack>

            {extraEmails.length > 0 && (
              <VStack align="stretch" spacing={2}>
                <Text fontSize="sm" fontWeight="medium" color="myGray.700">
                  {t('settings.extraEmails')}
                </Text>
                {extraEmails.map((email) => (
                  <HStack
                    key={email.mailAddress}
                    justify="space-between"
                    p={3}
                    border="1px solid"
                    borderColor="myGray.200"
                    borderRadius="md"
                  >
                    <HStack spacing={3}>
                      <Icon as={FiMail} color="myGray.400" />
                      <Text fontSize="sm">{email.mailAddress}</Text>
                    </HStack>
                    <IconButton
                      aria-label={t('common.delete')}
                      icon={<Icon as={FiTrash2} />}
                      variant="ghost"
                      size="sm"
                      colorScheme="red"
                      onClick={() => setDeletingEmail(email.mailAddress)}
                    />
                  </HStack>
                ))}
              </VStack>
            )}

            <Divider />

            <VStack align="stretch" spacing={3}>
              <Text fontSize="sm" fontWeight="medium">
                {t('settings.addEmail')}
              </Text>
              <HStack>
                <Input
                  type="email"
                  placeholder="new@email.com"
                  value={newEmail}
                  onChange={(e) => setNewEmail(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleAddEmail()
                  }}
                />
                <Button
                  variant="primary"
                  leftIcon={<Icon as={FiPlus} />}
                  onClick={handleAddEmail}
                  isLoading={saving}
                >
                  {t('settings.add')}
                </Button>
              </HStack>
            </VStack>
          </VStack>
        </Box>
      </VStack>

      <Modal isOpen={deletingEmail !== null} onClose={() => setDeletingEmail(null)}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('settings.deleteEmailTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>
              {t('settings.deleteEmailConfirm', { email: deletingEmail ?? '' })}
            </Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => setDeletingEmail(null)}>
              {t('common.cancel')}
            </Button>
            <Button variant="solid" colorScheme="red" onClick={handleConfirmDeleteEmail}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </>
  )
}
