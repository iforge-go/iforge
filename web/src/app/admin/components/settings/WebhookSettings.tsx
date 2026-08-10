'use client'

import { Box, VStack, HStack, Text, Button, FormControl, FormLabel, Textarea, Switch, Icon } from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { api, WebhookSettings } from '@/lib/api'
import { FiGlobe } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

export default function WebhookSettingsPage() {
  const toast = useGithubToast()
  const { t } = useI18n()
  const [settings, setSettings] = useState<WebhookSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    const loadSettings = async () => {
      try {
        const data = await api.getWebhookSettings()
        setSettings(data)
      } catch (error) {
        console.error('Failed to load webhook settings:', error)
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
      await api.updateWebhookSettings(settings)
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
        <HStack spacing={3} mb={2}>
          <Icon as={FiGlobe} w={5} h={5} color="primary.500" />
          <Text fontWeight="bold" fontSize="lg" color="myGray.900">
            {t('admin.settingsWebhook')}
          </Text>
        </HStack>

        <FormControl display="flex" alignItems="center">
          <FormLabel htmlFor="block-private" mb="0" flex="1">
            阻止私有地址
          </FormLabel>
          <Switch
            id="block-private"
            colorScheme="primary"
            isChecked={settings.blockPrivateAddress}
            onChange={(e) => setSettings({ ...settings, blockPrivateAddress: e.target.checked })}
          />
        </FormControl>

        <FormControl>
          <FormLabel>白名单</FormLabel>
          <Textarea
            value={settings.whitelist}
            onChange={(e) => setSettings({ ...settings, whitelist: e.target.value })}
            placeholder="每行一个 IP 或 CIDR，例如：&#10;192.168.1.0/24&#10;10.0.0.0/8"
            rows={6}
            fontFamily="mono"
            fontSize="sm"
          />
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
