'use client'

import {
  Box,
  Container,
  VStack,
  Heading,
  Text,
  Spinner,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  Button,
  Link as ChakraLink,
} from '@chakra-ui/react'
import { useState, useEffect, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'

type CallbackStatus = 'processing' | 'success' | 'error'

function OIDCCallbackContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { t } = useI18n()

  const code = searchParams.get('code') || ''
  const state = searchParams.get('state') || ''
  const [status, setStatus] = useState<CallbackStatus>('processing')
  const [errorMessage, setErrorMessage] = useState('')

  useEffect(() => {
    if (!code || !state) {
      setStatus('error')
      setErrorMessage(t('auth.oidcCallbackMissing'))
      return
    }

    let cancelled = false
    ;(async () => {
      try {
        const result = await api.oidcCallback(code, state)
        if (cancelled) return
        if (result.token) {
          localStorage.setItem('token', result.token)
          window.dispatchEvent(new Event('auth-change'))
          setStatus('success')
          setTimeout(() => router.push('/'), 500)
        } else {
          setStatus('error')
          setErrorMessage(t('auth.oidcCallbackFailed'))
        }
      } catch (error: any) {
        if (cancelled) return
        setStatus('error')
        setErrorMessage(error.message || t('auth.oidcCallbackFailed'))
      }
    })()

    return () => {
      cancelled = true
    }
  }, [code, state, router, t])

  return (
    <Container maxW="md" py={{ base: '12', md: '24' }}>
      <VStack spacing="8" align="stretch">
        <VStack spacing="2" textAlign="center">
          <Heading size="2xl" color="myGray.900">
            {t('auth.oidcLogin')}
          </Heading>
        </VStack>

        <Box
          bg="white"
          p="8"
          borderRadius="lg"
          border="1px solid"
          borderColor="myGray.200"
          boxShadow="0px 0px 1px 0px rgba(19, 51, 107, 0.08), 0px 1px 2px 0px rgba(19, 51, 107, 0.05)"
        >
          {status === 'processing' && (
            <VStack spacing="4" py="6">
              <Spinner size="lg" />
              <Text color="myGray.500">{t('auth.oidcCallbackProcessing')}</Text>
            </VStack>
          )}

          {status === 'success' && (
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
                {t('auth.loginSuccess')}
              </AlertTitle>
              <AlertDescription maxWidth="sm" color="myGray.600">
                {t('auth.oidcCallbackProcessing')}
              </AlertDescription>
            </Alert>
          )}

          {status === 'error' && (
            <>
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
                  {t('auth.oidcLoginFailed')}
                </AlertTitle>
                <AlertDescription maxWidth="sm" color="myGray.600">
                  {errorMessage}
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
            </>
          )}
        </Box>

        <Text textAlign="center" color="myGray.600">
          <ChakraLink href="/login" color="primary.600" fontWeight="medium">
            {t('auth.backToLogin')}
          </ChakraLink>
        </Text>
      </VStack>
    </Container>
  )
}

export default function OIDCCallbackPage() {
  return (
    <Suspense>
      <OIDCCallbackContent />
    </Suspense>
  )
}
