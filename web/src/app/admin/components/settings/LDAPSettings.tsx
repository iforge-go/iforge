'use client'

import { Box, VStack, HStack, Text, Button, FormControl, FormLabel, Input, Switch, Icon } from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { api, LDAPSettings } from '@/lib/api'
import { FiShield, FiEye, FiEyeOff } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

export default function LDAPSettingsPage() {
  const toast = useGithubToast()
  const { t } = useI18n()
  const [settings, setSettings] = useState<LDAPSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [showPassword, setShowPassword] = useState(false)

  useEffect(() => {
    const loadSettings = async () => {
      try {
        const data = await api.getLDAPSettings()
        setSettings(data)
      } catch (error) {
        console.error('Failed to load LDAP settings:', error)
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
      await api.updateLDAPSettings(settings)
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
      await api.testLDAPSettings()
      toast({ title: 'LDAP 测试成功', description: 'LDAP 连接测试通过', status: 'success', duration: 3000 })
    } catch (error: any) {
      toast({ title: 'LDAP 测试失败', description: error.message, status: 'error', duration: 3000 })
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
          <Icon as={FiShield} w={5} h={5} color="primary.500" />
          <Text fontWeight="bold" fontSize="lg" color="myGray.900">
            {t('admin.settingsLDAP')}
          </Text>
        </HStack>

        <FormControl display="flex" alignItems="center">
          <FormLabel htmlFor="ldap-enabled" mb="0" flex="1">
            启用 LDAP
          </FormLabel>
          <Switch
            id="ldap-enabled"
            colorScheme="primary"
            isChecked={settings.enabled}
            onChange={(e) => setSettings({ ...settings, enabled: e.target.checked })}
          />
        </FormControl>

        <FormControl>
          <FormLabel>LDAP Host</FormLabel>
          <Input
            value={settings.host}
            onChange={(e) => setSettings({ ...settings, host: e.target.value })}
            placeholder="ldap.example.com"
          />
        </FormControl>

        <FormControl>
          <FormLabel>LDAP Port</FormLabel>
          <Input
            type="number"
            value={settings.port}
            onChange={(e) => setSettings({ ...settings, port: parseInt(e.target.value) || 389 })}
            placeholder="389"
          />
        </FormControl>

        <FormControl>
          <FormLabel>Base DN</FormLabel>
          <Input
            value={settings.baseDN}
            onChange={(e) => setSettings({ ...settings, baseDN: e.target.value })}
            placeholder="dc=example,dc=com"
          />
        </FormControl>

        <FormControl>
          <FormLabel>Bind DN</FormLabel>
          <Input
            value={settings.bindDN}
            onChange={(e) => setSettings({ ...settings, bindDN: e.target.value })}
            placeholder="cn=admin,dc=example,dc=com"
          />
        </FormControl>

        <FormControl>
          <FormLabel>Bind Password</FormLabel>
          <HStack>
            <Input
              type={showPassword ? 'text' : 'password'}
              value={settings.bindPassword}
              onChange={(e) => setSettings({ ...settings, bindPassword: e.target.value })}
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
          <FormLabel>User Filter</FormLabel>
          <Input
            value={settings.userFilter}
            onChange={(e) => setSettings({ ...settings, userFilter: e.target.value })}
            placeholder="(uid={0})"
          />
        </FormControl>

        <FormControl>
          <FormLabel>Email Attribute</FormLabel>
          <Input
            value={settings.emailAttr}
            onChange={(e) => setSettings({ ...settings, emailAttr: e.target.value })}
            placeholder="mail"
          />
        </FormControl>

        <FormControl>
          <FormLabel>Name Attribute</FormLabel>
          <Input
            value={settings.nameAttr}
            onChange={(e) => setSettings({ ...settings, nameAttr: e.target.value })}
            placeholder="cn"
          />
        </FormControl>

        <FormControl display="flex" alignItems="center">
          <FormLabel htmlFor="ldap-tls" mb="0" flex="1">
            TLS
          </FormLabel>
          <Switch
            id="ldap-tls"
            colorScheme="primary"
            isChecked={settings.tls}
            onChange={(e) => setSettings({ ...settings, tls: e.target.checked })}
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
