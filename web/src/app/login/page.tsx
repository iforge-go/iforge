'use client'

import {
  Box,
  Button,
  Container,
  FormControl,
  FormLabel,
  Input,
  VStack,
  HStack,
  Divider,
  Heading,
  Text,
  Link as ChakraLink,
} from '@chakra-ui/react'
import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { useSiteSettings } from '@/contexts/SiteSettingsContext'
import { getLocalizedErrorMessage } from '@/lib/errorMessages'
import { FiExternalLink } from 'react-icons/fi'
import { useCurrentUser } from '@/contexts/UserContext'

export default function LoginPage() {
  const router = useRouter()
  const toast = useGithubToast()
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()
  const { settings: siteSettings } = useSiteSettings()
  const [loading, setLoading] = useState(false)
  const [oidcEnabled, setOidcEnabled] = useState(false)
  const [oidcLoading, setOidcLoading] = useState(false)
  const [formData, setFormData] = useState({
    userName: '',
    password: '',
  })

  // 检查 OIDC 是否启用
  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const result = await api.getOIDCStatus()
        if (!cancelled) setOidcEnabled(result.enabled)
      } catch {
        // OIDC 检测失败时静默处理，不显示按钮
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  // 已登录用户自动跳转到首页
  useEffect(() => {
    if (!authLoading && user) {
      router.replace('/')
    }
  }, [authLoading, user, router])

  // 认证检查中或已登录时，不渲染登录表单
  if (authLoading || user) {
    return null
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      const result = await api.login(formData.userName, formData.password)
      localStorage.setItem('token', result.token)
      window.dispatchEvent(new Event('auth-change'))
      toast({
        title: t('auth.loginSuccess'),
        status: 'success',
        duration: 2000,
      })
      router.push('/')
    } catch (error: any) {
      toast({
        title: t('auth.loginFailed'),
        description: getLocalizedErrorMessage(error, t),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }

  const handleOIDCLogin = async () => {
    setOidcLoading(true)
    try {
      const result = await api.getOIDCAuthURL()
      if (result.authUrl) {
        window.location.href = result.authUrl
      } else {
        throw new Error('No auth URL returned')
      }
    } catch (error: any) {
      toast({
        title: t('auth.oidcLoginFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
      setOidcLoading(false)
    }
  }

  return (
    <Container maxW="md" py={{ base: '12', md: '24' }}>
      <VStack spacing="8" align="stretch">
        <VStack spacing="2" textAlign="center">
          <Heading size="2xl" color="myGray.900">
            {t('auth.loginTitle')}
          </Heading>
          <Text color="myGray.600">
            {t('auth.loginSubtitle')}
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
            <FormControl isRequired>
              <FormLabel color="myGray.700">{t('auth.username')}</FormLabel>
              <Input
                type="text"
                value={formData.userName}
                onChange={(e) => setFormData({ ...formData, userName: e.target.value })}
                placeholder={t('auth.usernamePlaceholder')}
                autoComplete="username"
              />
            </FormControl>

            <FormControl isRequired>
              <HStack justify="space-between" align="end" mb="2">
                <FormLabel color="myGray.700" mb="0">{t('auth.password')}</FormLabel>
                <ChakraLink
                  href="/forgot-password"
                  color="primary.600"
                  fontWeight="medium"
                  fontSize="sm"
                >
                  {t('auth.forgotPassword')}
                </ChakraLink>
              </HStack>
              <Input
                type="password"
                value={formData.password}
                onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                placeholder={t('auth.passwordPlaceholder')}
                autoComplete="current-password"
              />
            </FormControl>

            <Button
              type="submit"
              variant="primary"
              size="lg"
              width="full"
              isLoading={loading}
            >
              {t('auth.login')}
            </Button>

            {oidcEnabled && (
              <>
                <HStack w="full" spacing="3">
                  <Divider />
                  <Text fontSize="sm" color="myGray.500" whiteSpace="nowrap">
                    {t('auth.or')}
                  </Text>
                  <Divider />
                </HStack>
                <Button
                  variant="outline"
                  size="lg"
                  width="full"
                  onClick={handleOIDCLogin}
                  isLoading={oidcLoading}
                  leftIcon={<FiExternalLink />}
                >
                  {t('auth.oidcLogin')}
                </Button>
              </>
            )}
          </VStack>
        </Box>

        {siteSettings?.allowRegistration && (
          <Text textAlign="center" color="myGray.600">
            {t('auth.noAccount')}{' '}
            <ChakraLink href="/register" color="primary.600" fontWeight="medium">
              {t('auth.registerNow')}
            </ChakraLink>
          </Text>
        )}
      </VStack>
    </Container>
  )
}
