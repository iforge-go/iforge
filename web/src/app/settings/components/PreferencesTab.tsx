'use client'

import {
  Box,
  VStack,
  HStack,
  FormControl,
  FormLabel,
  Select,
  Switch,
  Button,
  Heading,
  Text,
  Spinner,
} from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { api, AccountPreference } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

interface PreferencesTabProps {
  isActive: boolean
}

export default function PreferencesTab({ isActive }: PreferencesTabProps) {
  const toast = useGithubToast()
  const { t } = useI18n()
  const [saving, setSaving] = useState(false)

  // Account Preferences state
  const [, setPreferences] = useState<AccountPreference | null>(null)
  const [prefLoading, setPrefLoading] = useState(false)
  const [prefForm, setPrefForm] = useState({
    highlighterTheme: 'github',
    notification: true,
    timezone: 'UTC',
  })

  // Account Preferences: load when the preferences tab is active
  useEffect(() => {
    if (!isActive) return
    setPrefLoading(true)
    api.getUserPreferences()
      .then((pref) => {
        setPreferences(pref)
        setPrefForm({
          highlighterTheme: pref.highlighterTheme || 'github',
          notification: pref.notification,
          timezone: pref.timezone || 'UTC',
        })
      })
      .catch((err) => {
        toast({ title: t('settings.preferencesSaveFailed'), description: err.message, status: 'error', duration: 3000 })
      })
      .finally(() => setPrefLoading(false))
  }, [isActive, t, toast])

  const handleSavePreferences = async () => {
    setSaving(true)
    try {
      await api.updateUserPreferences(prefForm)
      toast({ title: t('settings.preferencesSaved'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('settings.preferencesSaveFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setSaving(false)
    }
  }

  return (
    <Box minH="500px" borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
      <VStack spacing={4} align="stretch" mb={4}>
        <Heading size="md">{t('settings.preferencesTitle')}</Heading>
        <Text fontSize="sm" color="myGray.600">
          {t('settings.preferencesDesc')}
        </Text>
      </VStack>
      <VStack spacing={5} align="stretch">
        {prefLoading ? (
          <HStack spacing={3}>
            <Spinner size="sm" />
            <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
          </HStack>
        ) : (
          <>
            <FormControl>
              <FormLabel>{t('settings.highlighterTheme')}</FormLabel>
              <Select
                value={prefForm.highlighterTheme}
                onChange={(e) => setPrefForm({ ...prefForm, highlighterTheme: e.target.value })}
                bg="white"
              >
                <option value="github">GitHub Dark</option>
                <option value="monokai">Monokai</option>
                <option value="dracula">Dracula</option>
                <option value="solarized-light">Solarized Light</option>
                <option value="solarized-dark">Solarized Dark</option>
                <option value="vim">Vim</option>
                <option value="atom-one">Atom One</option>
              </Select>
            </FormControl>

            <FormControl>
              <FormLabel>{t('settings.timezone')}</FormLabel>
              <Select
                value={prefForm.timezone}
                onChange={(e) => setPrefForm({ ...prefForm, timezone: e.target.value })}
                bg="white"
              >
                <option value="UTC">UTC</option>
                <option value="Asia/Shanghai">Asia/Shanghai (UTC+8)</option>
                <option value="Asia/Tokyo">Asia/Tokyo (UTC+9)</option>
                <option value="Asia/Singapore">Asia/Singapore (UTC+8)</option>
                <option value="Europe/London">Europe/London (UTC+0)</option>
                <option value="Europe/Paris">Europe/Paris (UTC+1)</option>
                <option value="America/New_York">America/New_York (UTC-5)</option>
                <option value="America/Los_Angeles">America/Los_Angeles (UTC-8)</option>
                <option value="America/Chicago">America/Chicago (UTC-6)</option>
              </Select>
            </FormControl>

            <FormControl>
              <HStack justify="space-between" align="center">
                <VStack align="start" spacing={0}>
                  <FormLabel mb={0}>{t('settings.notificationEnabled')}</FormLabel>
                  <Text fontSize="xs" color="myGray.500">
                    {t('settings.notifDesc')}
                  </Text>
                </VStack>
                <Switch
                  isChecked={prefForm.notification}
                  onChange={(e) => setPrefForm({ ...prefForm, notification: e.target.checked })}
                  colorScheme="primary"
                  size="lg"
                />
              </HStack>
            </FormControl>

            <HStack justify="flex-end">
              <Button
                variant="primary"
                onClick={handleSavePreferences}
                isLoading={saving}
              >
                {t('settings.saveChanges')}
              </Button>
            </HStack>
          </>
        )}
      </VStack>
    </Box>
  )
}
