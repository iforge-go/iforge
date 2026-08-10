'use client'

import { Box, VStack, HStack, Text, Button, FormControl, FormLabel, Input, Switch, Icon } from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { api, OIDCSettings } from '@/lib/api'
import { FiKey } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

export default function OIDCSettingsPage() {
  const toast = useGithubToast()
  const { t } = useI18n()
  const [settings, setSettings] = useState<OIDCSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)

  useEffect(() => {
    const loadSettings = async () => {
      try {
        const data = await api.getOIDCSettings()
        setSettings(data)
      } catch (error) {
        console.error('Failed to load OIDC settings:', error)
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
      await api.updateOIDCSettings(settings)
      toast({ title: t('admin.settingsSaved'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('admin.saveFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setSaving(false)
    }
  }

  const handleTest = async () => {
    setTesting(true)
    try {
      await api.testOIDCSettings()
      toast({ title: 'OIDC 测试成功', description: 'OIDC 连接测试通过', status: 'success', duration: 3000 })
    } catch (error: any) {
      toast({ title: 'OIDC 测试失败', description: error.message, status: 'error', duration: 3000 })
    } finally {
      setTesting(false)
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
          <Icon as={FiKey} w={5} h={5} color="primary.500" />
          <Text fontWeight="bold" fontSize="lg" color="myGray.900">
            {t('admin.settingsOIDC')}
          </Text>
        </HStack>

        <FormControl display="flex" alignItems="center">
          <FormLabel htmlFor="oidc-enabled" mb="0" flex="1">
            启用 OIDC
          </FormLabel>
          <Switch
            id="oidc-enabled"
            colorScheme="primary"
            isChecked={settings.enabled}
            onChange={(e) => setSettings({ ...settings, enabled: e.target.checked })}
          />
        </FormControl>

        <FormControl>
          <FormLabel>Client ID</FormLabel>
          <Input
            value={settings.clientId}
            onChange={(e) => setSettings({ ...settings, clientId: e.target.value })}
            placeholder="your-client-id"
          />
        </FormControl>

        <FormControl>
          <FormLabel>Client Secret</FormLabel>
          <Input
            type="password"
            value={settings.clientSecret}
            onChange={(e) => setSettings({ ...settings, clientSecret: e.target.value })}
            placeholder="••••••••"
          />
        </FormControl>

        <FormControl>
          <FormLabel>Auth URL</FormLabel>
          <Input
            value={settings.authURL}
            onChange={(e) => setSettings({ ...settings, authURL: e.target.value })}
            placeholder="https://auth.example.com/authorize"
          />
        </FormControl>

        <FormControl>
          <FormLabel>Token URL</FormLabel>
          <Input
            value={settings.tokenURL}
            onChange={(e) => setSettings({ ...settings, tokenURL: e.target.value })}
            placeholder="https://auth.example.com/token"
          />
        </FormControl>

        <FormControl>
          <FormLabel>User Info URL</FormLabel>
          <Input
            value={settings.userInfoURL}
            onChange={(e) => setSettings({ ...settings, userInfoURL: e.target.value })}
            placeholder="https://auth.example.com/userinfo"
          />
        </FormControl>

        <FormControl>
          <FormLabel>Redirect URL</FormLabel>
          <Input
            value={settings.redirectURL}
            onChange={(e) => setSettings({ ...settings, redirectURL: e.target.value })}
            placeholder="http://localhost:3000/auth/callback"
          />
        </FormControl>

        <HStack justify="flex-end" spacing={3}>
          <Button variant="whiteBase" isLoading={testing} onClick={handleTest}>
            测试连接
          </Button>
          <Button variant="primary" isLoading={saving} onClick={handleSave}>
            {t('admin.saveSettings')}
          </Button>
        </HStack>
      </VStack>
    </Box>
  )
}
