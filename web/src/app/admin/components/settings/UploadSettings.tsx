'use client'

import { Box, VStack, HStack, Text, Button, FormControl, FormLabel, Input, Icon } from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { api, UploadSettings } from '@/lib/api'
import { FiUpload } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

export default function UploadSettingsPage() {
  const toast = useGithubToast()
  const { t } = useI18n()
  const [settings, setSettings] = useState<UploadSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    const loadSettings = async () => {
      try {
        const data = await api.getUploadSettings()
        setSettings(data)
      } catch (error) {
        console.error('Failed to load upload settings:', error)
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
      await api.updateUploadSettings(settings)
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
          <Icon as={FiUpload} w={5} h={5} color="primary.500" />
          <Text fontWeight="bold" fontSize="lg" color="myGray.900">
            {t('admin.settingsUpload')}
          </Text>
        </HStack>

        <FormControl>
          <FormLabel>最大文件大小 (MB)</FormLabel>
          <Input
            type="number"
            value={settings.maxFileSize}
            onChange={(e) => setSettings({ ...settings, maxFileSize: parseInt(e.target.value) || 10 })}
            placeholder="10"
          />
        </FormControl>

        <FormControl>
          <FormLabel>超时时间 (秒)</FormLabel>
          <Input
            type="number"
            value={settings.timeout}
            onChange={(e) => setSettings({ ...settings, timeout: parseInt(e.target.value) || 300 })}
            placeholder="300"
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
