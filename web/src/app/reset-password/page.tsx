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
  Spinner,
} from '@chakra-ui/react'
import { useState, useEffect, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { api } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

type Status = 'validating' | 'valid' | 'invalid' | 'resetting' | 'success'

function ResetPasswordContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const toast = useGithubToast()
  const { t } = useI18n()
  const token = searchParams.get('token') || ''

  const [status, setStatus] = useState<Status>('validating')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  useEffect(() => {
    if (!token) {
      setStatus('invalid')
      return
    }

    let cancelled = false
    ;(async () => {
      try {
        const result = await api.validateResetToken(token)
        if (!cancelled) {
          setStatus(result.valid ? 'valid' : 'invalid')
        }
      } catch {
        if (!cancelled) {
          setStatus('invalid')
        }
      }
    })()

    return () => {
      cancelled = true
    }
  }, [token])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (password.length < 8) {
      toast({
        title: t('auth.loginFailed'),
        description: t('auth.newPasswordPlaceholder'),
        status: 'error',
        duration: 3000,
      })
      return
    }

    if (password !== confirmPassword) {
      toast({
        title: t('auth.loginFailed'),
        description: t('auth.passwordMismatch'),
        status: 'error',
        duration: 3000,
      })
      return
    }

    setStatus('resetting')
    try {
      await api.resetPassword(token, password)
      setStatus('success')
    } catch (error: any) {
      toast({
        title: t('auth.loginFailed'),
        description: error.message || t('auth.invalidOrExpiredToken'),
        status: 'error',
        duration: 3000,
      })
      setStatus('valid')
    }
  }

  return (
    <Container maxW="md" py={{ base: '12', md: '24' }}>
      <VStack spacing="8" align="stretch">
        <VStack spacing="2" textAlign="center">
          <Heading size="2xl" color="myGray.900">
            {t('auth.resetPasswordTitle')}
          </Heading>
          <Text color="myGray.600">
            {t('auth.resetPasswordSubtitle')}
          </Text>
        </VStack>

        {status === 'validating' && (
          <Box bg="white" p="8" borderRadius="lg" border="1px solid" borderColor="myGray.200" boxShadow="0px 0px 1px 0px rgba(19, 51, 107, 0.08), 0px 1px 2px 0px rgba(19, 51, 107, 0.05)">
            <VStack spacing="4" py="6">
              <Spinner size="lg" />
              <Text color="myGray.500">{t('auth.oidcCallbackProcessing')}</Text>
            </VStack>
          </Box>
        )}

        {status === 'invalid' && (
          <Box
            bg="white"
            p="8"
            borderRadius="lg"
            border="1px solid"
            borderColor="myGray.200"
            boxShadow="0px 0px 1px 0px rgba(19, 51, 107, 0.08), 0px 1px 2px 0px rgba(19, 51, 107, 0.05)"
          >
            <Alert
              status="error"
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
                {t('auth.invalidOrExpiredToken')}
              </AlertTitle>
              <AlertDescription maxWidth="sm" color="myGray.600">
                {t('auth.invalidOrExpiredTokenDesc')}
              </AlertDescription>
            </Alert>
            <Button
              mt="6"
              variant="primary"
              size="lg"
              width="full"
              onClick={() => router.push('/forgot-password')}
            >
              {t('auth.forgotPasswordTitle')}
            </Button>
          </Box>
        )}

        {status === 'success' && (
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
                {t('auth.resetSuccess')}
              </AlertTitle>
              <AlertDescription maxWidth="sm" color="myGray.600">
                {t('auth.resetSuccessDesc')}
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
        )}

        {(status === 'valid' || status === 'resetting') && (
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
                <FormLabel color="myGray.700">{t('auth.password')}</FormLabel>
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={t('auth.newPasswordPlaceholder')}
                  autoComplete="new-password"
                  autoFocus
                />
              </FormControl>

              <FormControl isRequired>
                <FormLabel color="myGray.700">{t('auth.confirmPassword')}</FormLabel>
                <Input
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder={t('auth.confirmPasswordPlaceholder')}
                  autoComplete="new-password"
                />
              </FormControl>

              <Button
                type="submit"
                variant="primary"
                size="lg"
                width="full"
                isLoading={status === 'resetting'}
              >
                {t('auth.resetPassword')}
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

export default function ResetPasswordPage() {
  return (
    <Suspense>
      <ResetPasswordContent />
    </Suspense>
  )
}
