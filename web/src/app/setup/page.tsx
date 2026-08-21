'use client'

import {
  Box,
  Button,
  Container,
  FormControl,
  FormLabel,
  Input,
  VStack,
  Heading,
  Text,
  Alert,
  AlertIcon,
  FormErrorMessage,
  Spinner,
  SimpleGrid,
} from '@chakra-ui/react'
import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { Logo } from '@/components/Logo'
import { UsernameInput } from '@/components/UsernameInput'
import { getLocalizedErrorMessage } from '@/lib/errorMessages'
import type { UsernameStatus } from '@/hooks/useUsernameCheck'

export default function SetupPage() {
  const router = useRouter()
  const toast = useGithubToast()
  const { t } = useI18n()
  const [loading, setLoading] = useState(false)
  const [checking, setChecking] = useState(true)
  const [formData, setFormData] = useState({
    adminUsername: '',
    adminPassword: '',
    confirmPassword: '',
    adminEmail: '',
  })
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [usernameStatus, setUsernameStatus] = useState<UsernameStatus>('idle')

  useEffect(() => {
    api.getSetupStatus()
      .then((result) => {
        if (result.initialized) {
          router.replace('/')
        }
        setChecking(false)
      })
      .catch(() => {
        setChecking(false)
      })
  }, [router])

  const validate = (): boolean => {
    const e: Record<string, string> = {}
    // 用户名可用性由 UsernameInput 实时校验，提交时据 usernameStatus 判断，不在此处重复校验
    if (formData.adminPassword.length < 8) {
      e.adminPassword = t('setup.passwordError')
    }
    if (formData.confirmPassword !== formData.adminPassword) {
      e.confirmPassword = t('auth.passwordMismatch')
    }
    if (!formData.adminEmail || !formData.adminEmail.includes('@')) {
      e.adminEmail = t('setup.emailError')
    }
    setErrors(e)
    return Object.keys(e).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    // 用户名不可用时阻止提交；校验中则提示稍候
    if (usernameStatus === 'idle' && !formData.adminUsername.trim()) {
      toast({ title: t('auth.usernameRequired'), status: 'warning', duration: 2000 })
      return
    }
    if (usernameStatus === 'invalid' || usernameStatus === 'reserved' || usernameStatus === 'taken') {
      toast({ title: t('auth.usernameNotConfirmed'), status: 'warning', duration: 2000 })
      return
    }
    if (usernameStatus === 'checking') {
      toast({ title: t('auth.usernameChecking'), status: 'info', duration: 2000 })
      return
    }
    if (!validate()) return
    setLoading(true)
    try {
      await api.setup({
        adminUsername: formData.adminUsername,
        adminPassword: formData.adminPassword,
        adminEmail: formData.adminEmail,
      })
      toast({
        title: t('setup.success'),
        description: t('setup.successDesc'),
        status: 'success',
        duration: 3000,
      })
      router.push('/login')
    } catch (error: any) {
      toast({
        title: t('setup.failed'),
        description: getLocalizedErrorMessage(error, t),
        status: 'error',
        duration: 4000,
      })
    } finally {
      setLoading(false)
    }
  }

  if (checking) {
    return (
      <Container maxW="md" py={{ base: '12', md: '24' }}>
        <VStack spacing="4">
          <Spinner size="xl" />
        </VStack>
      </Container>
    )
  }

  return (
    <Container maxW="2xl" py={{ base: '12', md: '24' }}>
      <VStack spacing="8" align="stretch">
        <VStack spacing="3" textAlign="center">
          <Logo size={48} />
          <Heading size="2xl" color="myGray.900">
            {t('setup.title')}
          </Heading>
          <Text color="myGray.600">
            {t('setup.subtitle')}
          </Text>
        </VStack>

        <Alert status="info" borderRadius="lg">
          <AlertIcon />
          {t('setup.notice')}
        </Alert>

        <Box
          as="form"
          onSubmit={handleSubmit}
          bg="white"
          p="8"
          borderRadius="lg"
          border="1px solid"
          borderColor="myGray.200"
          boxShadow="0px 0px 1px 0px rgba(19, 51, 107, 0.08), 0px 1px 2px 0px rgba(19, 51, 107, 0.05)"
        >
          <VStack spacing="5">
            <SimpleGrid columns={2} spacing={5} width="full">
              <UsernameInput
                value={formData.adminUsername}
                onChange={(v) => setFormData({ ...formData, adminUsername: v })}
                onStatusChange={setUsernameStatus}
                label={t('auth.username')}
                placeholder={t('auth.usernamePlaceholder')}
                isRequired
                autoComplete="username"
              />

              <FormControl isRequired isInvalid={!!errors.adminEmail}>
                <FormLabel color="myGray.700">{t('auth.email')}</FormLabel>
                <Input
                  type="email"
                  value={formData.adminEmail}
                  onChange={(e) => setFormData({ ...formData, adminEmail: e.target.value })}
                  placeholder={t('auth.emailPlaceholder')}
                  autoComplete="email"
                />
                {errors.adminEmail && (
                  <FormErrorMessage>{errors.adminEmail}</FormErrorMessage>
                )}
              </FormControl>
            </SimpleGrid>

            <SimpleGrid columns={2} spacing={5} width="full">
              <FormControl isRequired isInvalid={!!errors.adminPassword}>
                <FormLabel color="myGray.700">{t('auth.password')}</FormLabel>
                <Input
                  type="password"
                  value={formData.adminPassword}
                  onChange={(e) => setFormData({ ...formData, adminPassword: e.target.value })}
                  placeholder={t('auth.passwordPlaceholder')}
                  autoComplete="new-password"
                />
                {errors.adminPassword && (
                  <FormErrorMessage>{errors.adminPassword}</FormErrorMessage>
                )}
              </FormControl>

              <FormControl isRequired isInvalid={!!errors.confirmPassword}>
                <FormLabel color="myGray.700">{t('auth.confirmPassword')}</FormLabel>
                <Input
                  type="password"
                  value={formData.confirmPassword}
                  onChange={(e) => setFormData({ ...formData, confirmPassword: e.target.value })}
                  placeholder={t('auth.confirmPasswordPlaceholder')}
                  autoComplete="new-password"
                />
                {errors.confirmPassword && (
                  <FormErrorMessage>{errors.confirmPassword}</FormErrorMessage>
                )}
              </FormControl>
            </SimpleGrid>

            <Button
              type="submit"
              variant="primary"
              size="lg"
              width="full"
              isLoading={loading}
            >
              {t('setup.complete')}
            </Button>
          </VStack>
        </Box>
      </VStack>
    </Container>
  )
}
