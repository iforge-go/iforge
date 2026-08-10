'use client'

import { Box, VStack, HStack, Text, Button, FormControl, FormLabel, Input, Switch, Icon } from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { api, SMTPSettings } from '@/lib/api'
import { FiMail, FiEye, FiEyeOff } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

export default function SMTPSettingsPage() {
  const toast = useGithubToast()
  const { t } = useI18n()
  const [settings, setSettings] = useState<SMTPSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [showPassword, setShowPassword] = useState(false)

  useEffect(() => {
    const loadSettings = async () => {
      try {
        const data = await api.getSMTPSettings()
        setSettings(data)
      } catch (error) {
        console.error('Failed to load SMTP settings:', error)
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
      await api.updateSMTPSettings(settings)
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
      await api.testSMTPSettings()
      toast({ title: 'SMTP 测试成功', description: '邮件发送测试通过', status: 'success', duration: 3000 })
    } catch (error: any) {
      toast({ title: 'SMTP 测试失败', description: error.message, status: 'error', duration: 3000 })
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
          <Icon as={FiMail} w={5} h={5} color="primary.500" />
          <Text fontWeight="bold" fontSize="lg" color="myGray.900">
            {t('admin.settingsSMTP')}
          </Text>
        </HStack>

        <FormControl>
          <FormLabel>SMTP Host</FormLabel>
          <Input
            value={settings.host}
            onChange={(e) => setSettings({ ...settings, host: e.target.value })}
            placeholder="smtp.example.com"
          />
        </FormControl>

        <FormControl>
          <FormLabel>SMTP Port</FormLabel>
          <Input
            type="number"
            value={settings.port}
            onChange={(e) => setSettings({ ...settings, port: parseInt(e.target.value) || 587 })}
            placeholder="587"
          />
        </FormControl>

        <FormControl>
          <FormLabel>Username</FormLabel>
          <Input
            value={settings.username}
            onChange={(e) => setSettings({ ...settings, username: e.target.value })}
            placeholder="user@example.com"
          />
        </FormControl>

        <FormControl>
          <FormLabel>Password</FormLabel>
          <HStack>
            <Input
              type={showPassword ? 'text' : 'password'}
              value={settings.password}
              onChange={(e) => setSettings({ ...settings, password: e.target.value })}
              placeholder="••••••••"
            />
            <Button
              size="sm"
              variant="ghost"
              onClick={() => setShowPassword(!showPassword)}
              leftIcon={<Icon as={showPassword ? FiEyeOff : FiEye} />}
            />
          </HStack>
        </FormControl>

        <FormControl>
          <FormLabel>From Address</FormLabel>
          <Input
            value={settings.from}
            onChange={(e) => setSettings({ ...settings, from: e.target.value })}
            placeholder="noreply@example.com"
          />
        </FormControl>

        <FormControl display="flex" alignItems="center">
          <FormLabel htmlFor="smtp-ssl" mb="0" flex="1">
            SSL/TLS
          </FormLabel>
          <Switch
            id="smtp-ssl"
            colorScheme="primary"
            isChecked={settings.ssl}
            onChange={(e) => setSettings({ ...settings, ssl: e.target.checked })}
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
