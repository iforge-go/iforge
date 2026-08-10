'use client'

import { Box, VStack, HStack, Text, Button, FormControl, FormLabel, Input, Switch, Select } from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { api, GeneralSettings } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

export default function GeneralSettingsPage() {
  const toast = useGithubToast()
  const { t } = useI18n()
  const [settings, setSettings] = useState<GeneralSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    const loadSettings = async () => {
      try {
        const data = await api.getGeneralSettings()
        setSettings(data)
      } catch (error) {
        console.error('Failed to load general settings:', error)
      } finally {
        setLoading(false)
      }
    }
    loadSettings()
  }, [])

  const handleSave = async () => {
    if (!settings) return
    setSaving(true)
    try {
      await api.updateGeneralSettings(settings)
      toast({ title: t('admin.settingsSaved'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('admin.saveFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <Text>{t('common.loading')}</Text>
  }

  if (!settings) {
    return <Text color="red.500">Failed to load settings</Text>
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
      <VStack spacing={6} align="stretch">
        <FormControl display="flex" alignItems="center">
          <FormLabel htmlFor="allow-register" mb="0" flex="1">
            {t('admin.allowRegister')}
          </FormLabel>
          <Switch
            id="allow-register"
            colorScheme="primary"
            isChecked={settings.allowRegistration}
            onChange={(e) => setSettings({ ...settings, allowRegistration: e.target.checked })}
          />
        </FormControl>

        <FormControl display="flex" alignItems="center">
          <FormLabel htmlFor="allow-anonymous" mb="0" flex="1">
            {t('admin.allowAnonymous')}
          </FormLabel>
          <Switch
            id="allow-anonymous"
            colorScheme="primary"
            isChecked={settings.allowAnonymous}
            onChange={(e) => setSettings({ ...settings, allowAnonymous: e.target.checked })}
          />
        </FormControl>

        <FormControl>
          <FormLabel>{t('admin.siteName')}</FormLabel>
          <Input
            value={settings.siteName}
            onChange={(e) => setSettings({ ...settings, siteName: e.target.value })}
          />
        </FormControl>

        <FormControl>
          <FormLabel>{t('admin.siteDesc')}</FormLabel>
          <Input
            value={settings.description}
            onChange={(e) => setSettings({ ...settings, description: e.target.value })}
            placeholder={t('admin.siteDescPlaceholder')}
          />
        </FormControl>

        <FormControl>
          <FormLabel>{t('admin.defaultBranch')}</FormLabel>
          <Input
            value={settings.defaultBranch}
            onChange={(e) => setSettings({ ...settings, defaultBranch: e.target.value })}
            placeholder="main"
          />
        </FormControl>

        <FormControl>
          <FormLabel>{t('admin.timezone')}</FormLabel>
          <Select
            value={settings.timezone}
            onChange={(e) => setSettings({ ...settings, timezone: e.target.value })}
            placeholder={t('admin.selectTimezone')}
          >
            {Intl.supportedValuesOf('timeZone').map((tz) => (
              <option key={tz} value={tz}>
                {tz}
              </option>
            ))}
          </Select>
        </FormControl>

        <HStack justify="flex-end">
          <Button variant="primary" isLoading={saving} onClick={handleSave}>
            {t('admin.saveSettings')}
          </Button>
        </HStack>
      </VStack>
    </Box>
  )
}
