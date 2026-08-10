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
  Link as ChakraLink,
} from '@chakra-ui/react'
import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { getLocalizedErrorMessage } from '@/lib/errorMessages'
import { UsernameInput } from '@/components/UsernameInput'
import type { UsernameStatus } from '@/hooks/useUsernameCheck'
import { useCurrentUser } from '@/contexts/UserContext'
import { useSiteSettings } from '@/contexts/SiteSettingsContext'
import { Alert, AlertIcon, AlertTitle, AlertDescription, Spinner, Center } from '@chakra-ui/react'

export default function RegisterPage() {
  const router = useRouter()
  const toast = useGithubToast()
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()
  const { settings: siteSettings, loading: settingsLoading } = useSiteSettings()
  const [loading, setLoading] = useState(false)
  const [formData, setFormData] = useState({
    userName: '',
    password: '',
    fullName: '',
    mailAddress: '',
  })
  const [usernameStatus, setUsernameStatus] = useState<UsernameStatus>('idle')

  // 已登录用户自动跳转到首页
  useEffect(() => {
    if (!authLoading && user) {
      router.replace('/')
    }
  }, [authLoading, user, router])

  // 认证检查中或已登录时，不渲染注册表单
  if (authLoading || user) {
    return null
  }

  // 正在加载配置时显示加载状态
  if (settingsLoading) {
    return (
      <Center py="20">
        <Spinner size="xl" />
      </Center>
    )
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    // 用户名必须通过实时校验为"可用"才允许提交
    if (usernameStatus !== 'available') {
      toast({ title: t('auth.usernameNotConfirmed'), status: 'warning', duration: 2000 })
      return
    }
    setLoading(true)

    try {
      await api.register(
        formData.userName,
        formData.password,
        formData.fullName,
        formData.mailAddress
      )
      toast({
        title: t('auth.registerSuccess'),
        description: t('auth.registerSuccessDesc'),
        status: 'success',
        duration: 2000,
      })
      router.push('/login')
    } catch (error: any) {
      toast({
        title: t('auth.registerFailed'),
        description: getLocalizedErrorMessage(error, t),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }

  // 注册已关闭时显示提示
  if (siteSettings && !siteSettings.allowRegistration) {
    return (
      <Container maxW="md" py={{ base: '12', md: '24' }}>
        <VStack spacing="8" align="stretch">
          <VStack spacing="2" textAlign="center">
            <Heading size="2xl" color="myGray.900">
              {t('auth.registerTitle')}
            </Heading>
          </VStack>

          <Alert
            status="warning"
            variant="subtle"
            flexDirection="column"
            alignItems="center"
            justifyContent="center"
            textAlign="center"
            py="10"
            bg="orange.50"
            borderRadius="lg"
            border="1px solid"
            borderColor="orange.200"
          >
            <AlertIcon boxSize="40px" mr={0} color="orange.500" />
            <AlertTitle mt={4} mb={1} fontSize="lg" fontWeight="bold" color="orange.700">
              {t('auth.registrationDisabled')}
            </AlertTitle>
            <AlertDescription maxWidth="sm" color="orange.600">
              {t('auth.registrationDisabledDesc')}
            </AlertDescription>
          </Alert>

          <Text textAlign="center" color="myGray.600">
            {t('auth.hasAccount')}{' '}
            <ChakraLink href="/login" color="primary.600" fontWeight="medium">
              {t('auth.loginNow')}
            </ChakraLink>
          </Text>
        </VStack>
      </Container>
    )
  }

  return (
    <Container maxW="md" py={{ base: '12', md: '24' }}>
      <VStack spacing="8" align="stretch">
        <VStack spacing="2" textAlign="center">
          <Heading size="2xl" color="myGray.900">
            {t('auth.registerTitle')}
          </Heading>
          <Text color="myGray.600">
            {t('auth.registerSubtitle')}
          </Text>
        </VStack>

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
            <UsernameInput
              value={formData.userName}
              onChange={(v) => setFormData({ ...formData, userName: v })}
              onStatusChange={setUsernameStatus}
              label={t('auth.username')}
              placeholder={t('auth.usernamePlaceholder')}
              isRequired
              autoComplete="username"
            />

            <FormControl isRequired>
              <FormLabel color="myGray.700">{t('auth.fullName')}</FormLabel>
              <Input
                type="text"
                value={formData.fullName}
                onChange={(e) => setFormData({ ...formData, fullName: e.target.value })}
                placeholder={t('auth.fullNamePlaceholder')}
              />
            </FormControl>

            <FormControl isRequired>
              <FormLabel color="myGray.700">{t('auth.email')}</FormLabel>
              <Input
                type="email"
                value={formData.mailAddress}
                onChange={(e) => setFormData({ ...formData, mailAddress: e.target.value })}
                placeholder={t('auth.emailPlaceholder')}
                autoComplete="email"
              />
            </FormControl>

            <FormControl isRequired>
              <FormLabel color="myGray.700">{t('auth.password')}</FormLabel>
              <Input
                type="password"
                value={formData.password}
                onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                placeholder={t('auth.passwordPlaceholder')}
                autoComplete="new-password"
              />
            </FormControl>

            <Button
              type="submit"
              variant="primary"
              size="lg"
              width="full"
              isLoading={loading}
            >
              {t('auth.register')}
            </Button>
          </VStack>
        </Box>

        <Text textAlign="center" color="myGray.600">
          {t('auth.hasAccount')}{' '}
          <ChakraLink href="/login" color="primary.600" fontWeight="medium">
            {t('auth.loginNow')}
          </ChakraLink>
        </Text>
      </VStack>
    </Container>
  )
}
