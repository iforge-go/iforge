'use client'

import {
  Box,
  VStack,
  HStack,
  FormControl,
  FormLabel,
  Input,
  Textarea,
  Avatar,
  Button,
  Heading,
  Text,
} from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { api, User } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'

interface ProfileTabProps {
  user: User
}

export default function ProfileTab({ user }: ProfileTabProps) {
  const toast = useGithubToast()
  const { t } = useI18n()
  const { refreshUser } = useCurrentUser()
  const [saving, setSaving] = useState(false)
  const [profileForm, setProfileForm] = useState({
    fullName: '',
    email: '',
    description: '',
  })

  useEffect(() => {
    setProfileForm({
      fullName: user.fullName || '',
      email: user.mailAddress || '',
      description: user.description || '',
    })
  }, [user])

  const handleProfileSave = async () => {
    setSaving(true)
    try {
      await api.updateUser({
        fullName: profileForm.fullName,
        mailAddress: profileForm.email,
        description: profileForm.description,
      })
      toast({ title: t('settings.profileUpdated'), status: 'success', duration: 2000 })
      // 重新获取用户信息
      refreshUser()
    } catch (error: any) {
      toast({ title: t('settings.updateFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setSaving(false)
    }
  }

  return (
    <Box minH="500px" borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
      <VStack spacing={4} align="stretch" mb={4}>
        <Heading size="md">{t('settings.profileTitle')}</Heading>
        <Text fontSize="sm" color="myGray.600">
          {t('settings.profileDesc')}
        </Text>
      </VStack>
      <VStack spacing={5}>
        <HStack spacing={6} w="full">
          <Avatar size="xl" name={user.fullName || user.userName} src={user.image || undefined} />
          <VStack align="start" spacing={1}>
            <Button variant="whiteBase" size="sm">
              {t('settings.changeAvatar')}
            </Button>
            <Text fontSize="xs" color="myGray.500">
              {t('settings.avatarHint')}
            </Text>
          </VStack>
        </HStack>

        <FormControl>
          <FormLabel>{t('settings.username')}</FormLabel>
          <Input value={user.userName} isReadOnly bg="myGray.50" />
          <Text fontSize="xs" color="myGray.500" mt={1}>
            {t('settings.usernameReadonly')}
          </Text>
        </FormControl>

        <FormControl>
          <FormLabel>{t('settings.fullName')}</FormLabel>
          <Input
            value={profileForm.fullName}
            onChange={(e) => setProfileForm({ ...profileForm, fullName: e.target.value })}
            placeholder={t('settings.fullNamePlaceholder')}
          />
        </FormControl>

        <FormControl>
          <FormLabel>{t('settings.emailAddress')}</FormLabel>
          <Input
            type="email"
            value={profileForm.email}
            onChange={(e) => setProfileForm({ ...profileForm, email: e.target.value })}
            placeholder="your@email.com"
          />
        </FormControl>

        <FormControl>
          <FormLabel>{t('settings.description')}</FormLabel>
          <Textarea
            value={profileForm.description}
            onChange={(e) => setProfileForm({ ...profileForm, description: e.target.value })}
            placeholder={t('settings.descriptionPlaceholder')}
            rows={3}
          />
        </FormControl>

        <HStack justify="flex-end" w="full">
          <Button variant="primary" onClick={handleProfileSave} isLoading={saving}>
            {t('settings.saveChanges')}
          </Button>
        </HStack>
      </VStack>
    </Box>
  )
}
