'use client'

import { Box, VStack, HStack, Text, Button, FormControl, FormLabel, Input, Icon } from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { api, RepositorySettings } from '@/lib/api'
import { FiFolder } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

export default function RepositorySettingsPage() {
  const toast = useGithubToast()
  const { t } = useI18n()
  const [settings, setSettings] = useState<RepositorySettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    const loadSettings = async () => {
      try {
        const data = await api.getRepositorySettings()
        setSettings({
          maxDiffFiles: data.maxDiffFiles ?? 100,
          maxDiffLines: data.maxDiffLines ?? 1000,
          httpsUrlTemplate: data.httpsUrlTemplate ?? '',
          sshUrlTemplate: data.sshUrlTemplate ?? '',
        })
      } catch (error) {
        console.error('Failed to load repository settings:', error)
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
      await api.updateRepositorySettings(settings)
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
          <Icon as={FiFolder} w={5} h={5} color="primary.500" />
          <Text fontWeight="bold" fontSize="lg" color="myGray.900">
            {t('admin.settingsRepository')}
          </Text>
        </HStack>

        <FormControl>
          <FormLabel>最大差异文件数</FormLabel>
          <Input
            type="number"
            value={settings.maxDiffFiles}
            onChange={(e) => setSettings({ ...settings, maxDiffFiles: parseInt(e.target.value) || 100 })}
            placeholder="100"
          />
        </FormControl>

        <FormControl>
          <FormLabel>最大差异行数</FormLabel>
          <Input
            type="number"
            value={settings.maxDiffLines}
            onChange={(e) => setSettings({ ...settings, maxDiffLines: parseInt(e.target.value) || 1000 })}
            placeholder="1000"
          />
        </FormControl>

        <FormControl>
          <FormLabel>HTTPS Clone URL 前缀</FormLabel>
          <Input
            value={settings.httpsUrlTemplate}
            onChange={(e) => setSettings({ ...settings, httpsUrlTemplate: e.target.value })}
            placeholder="https://{host}:{port}"
          />
          <Text fontSize="xs" color="myGray.500" mt={1}>
            可用变量：{'{host}'}、{'{port}'}。端口 443/80 时自动省略。系统自动拼接 /{'{owner}'}/{'{repo}'}.git。例如：https://git.example.com:8443
          </Text>
        </FormControl>

        <FormControl>
          <FormLabel>SSH Clone URL 前缀</FormLabel>
          <Input
            value={settings.sshUrlTemplate}
            onChange={(e) => setSettings({ ...settings, sshUrlTemplate: e.target.value })}
            placeholder="ssh://git@{host}:{port}"
          />
          <Text fontSize="xs" color="myGray.500" mt={1}>
            可用变量：{'{host}'}、{'{port}'}。端口 22 时自动转换为 SCP-like 格式。系统自动拼接 /{'{owner}'}/{'{repo}'}.git。例如：ssh://git@{'{host}'}:{'{port}'}
          </Text>
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
