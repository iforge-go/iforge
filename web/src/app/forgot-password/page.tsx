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
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
} from '@chakra-ui/react'
import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'

export default function ForgotPasswordPage() {
  const router = useRouter()
  const toast = useGithubToast()
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()
  const [loading, setLoading] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [email, setEmail] = useState('')

  // 已登录用户自动跳转到首页
  useEffect(() => {
    if (!authLoading && user) {
      router.replace('/')
    }
  }, [authLoading, user, router])

  // 认证检查中或已登录时，不渲染表单
  if (authLoading || user) {
    return null
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      await api.requestPasswordReset(email)
      setSubmitted(true)
    } catch (error: any) {
      // 后端为防止枚举攻击，无论邮箱是否存在都返回相同成功信息。
      // 只有网络/服务错误才提示失败。
      if (error?.status === 429) {
        toast({
          title: t('auth.loginFailed'),
          description: error.message,
          status: 'error',
          duration: 3000,
        })
      } else {
        // 其他错误也按成功处理，避免泄露邮箱存在性
        setSubmitted(true)
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Container maxW="md" py={{ base: '12', md: '24' }}>
      <VStack spacing="8" align="stretch">
        <VStack spacing="2" textAlign="center">
          <Heading size="2xl" color="myGray.900">
            {t('auth.forgotPasswordTitle')}
          </Heading>
          <Text color="myGray.600">
            {t('auth.forgotPasswordSubtitle')}
          </Text>
        </VStack>

        {submitted ? (
          <Box
            bg="white"
            p="8"
            borderRadius="lg"
            border="1px solid"
            borderColor="myGray.200"
            boxShadow="0px 0px 1px 0px rgba(19, 51, 107, 0.08), 0px 1px 2px 0px rgba(19, 51, 107, 0.05)"
          >
            <Alert
              status="success"
              variant="subtle"
              flexDirection="column"
              alignItems="center"
              justifyContent="center"
              textAlign="center"
              borderRadius="md"
              py="6"
            >
              <AlertIcon boxSize="6" mr={0} />
              <AlertTitle mt="3" mb="2" fontSize="lg">
                {t('auth.resetLinkSent')}
              </AlertTitle>
              <AlertDescription maxWidth="sm" color="myGray.600">
                {t('auth.resetLinkSentDesc')}
              </AlertDescription>
            </Alert>
            <Button
              mt="6"
              variant="primary"
              size="lg"
              width="full"
              onClick={() => router.push('/login')}
            >
              {t('auth.backToLogin')}
            </Button>
          </Box>
        ) : (
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
                <FormLabel color="myGray.700">{t('auth.email')}</FormLabel>
                <Input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder={t('auth.emailPlaceholder')}
                  autoComplete="email"
                  autoFocus
                />
              </FormControl>

              <Button
                type="submit"
                variant="primary"
                size="lg"
                width="full"
                isLoading={loading}
              >
                {t('auth.sendResetLink')}
              </Button>
            </VStack>
          </Box>
        )}

        <Text textAlign="center" color="myGray.600">
          <ChakraLink href="/login" color="primary.600" fontWeight="medium">
            {t('auth.backToLogin')}
          </ChakraLink>
        </Text>
      </VStack>
    </Container>
  )
}
