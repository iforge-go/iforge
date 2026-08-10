'use client'

import {
  Box,
  VStack,
  HStack,
  FormControl,
  FormLabel,
  Input,
  Code,
  Button,
  Heading,
  Text,
  Icon,
  Divider,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
} from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { api, AccessToken } from '@/lib/api'
import { FiLock, FiTrash2, FiPlus } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { formatRelativeTime } from '@/lib/time'
import { useI18n } from '@/contexts/I18nContext'

interface AccessTokensTabProps {
  isActive: boolean
}

export default function AccessTokensTab({ isActive }: AccessTokensTabProps) {
  const toast = useGithubToast()
  const { t } = useI18n()
  const [saving, setSaving] = useState(false)

  // Access Tokens state
  const [accessTokens, setAccessTokens] = useState<AccessToken[]>([])
  const [tokenLoading, setTokenLoading] = useState(false)
  const [newTokenNote, setNewTokenNote] = useState('')
  const [deletingTokenId, setDeletingTokenId] = useState<number | null>(null)
  const [newTokenValue, setNewTokenValue] = useState<string | null>(null)

  // Access Tokens: load when the tokens tab is active
  useEffect(() => {
    if (!isActive) return
    setTokenLoading(true)
    api.listAccessTokens()
      .then((data) => setAccessTokens(data || []))
      .catch((err) => {
        toast({ title: t('settings.loadTokensFailed'), description: err.message, status: 'error', duration: 3000 })
      })
      .finally(() => setTokenLoading(false))
  }, [isActive])

  const handleCreateToken = async () => {
    if (!newTokenNote.trim()) {
      toast({ title: t('settings.fillTokenNote'), status: 'warning', duration: 2000 })
      return
    }
    setSaving(true)
    try {
      const { token, value } = await api.createAccessToken(newTokenNote.trim())
      setAccessTokens([token, ...accessTokens])
      setNewTokenValue(value)
      setNewTokenNote('')
      toast({ title: t('settings.tokenCreated'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('settings.createTokenFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setSaving(false)
    }
  }

  const handleConfirmDeleteToken = async () => {
    if (deletingTokenId === null) return
    try {
      await api.deleteAccessToken(deletingTokenId)
      setAccessTokens(accessTokens.filter((t) => t.id !== deletingTokenId))
      toast({ title: t('settings.tokenDeleted'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('settings.deleteTokenFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setDeletingTokenId(null)
    }
  }

  return (
    <>
      <VStack spacing={4} align="stretch">
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
          <VStack spacing={4} align="stretch" mb={4}>
            <Heading size="md">{t('settings.tokensTitle')}</Heading>
            <Text fontSize="sm" color="myGray.600">
              {t('settings.tokensDesc')}
            </Text>
          </VStack>

          {tokenLoading ? (
            <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
          ) : accessTokens.length === 0 ? (
            <Text fontSize="sm" color="myGray.500">{t('settings.noTokens')}</Text>
          ) : (
            <VStack spacing={0} align="stretch" divider={<Divider />}>
              {accessTokens.map((token) => (
                <HStack key={token.id} justify="space-between" py={3}>
                  <VStack align="start" spacing={1} flex={1} minW={0}>
                    <HStack spacing={2}>
                      <Icon as={FiLock} color="myGray.500" />
                      <Text fontSize="sm" fontWeight="medium">{token.note}</Text>
                    </HStack>
                    <Text fontSize="xs" color="myGray.500">
                      {t('common.addedAt')} {formatRelativeTime(token.createdAt, t)}
                    </Text>
                    {token.lastUsedAt && (
                      <Text fontSize="xs" color="myGray.500">
                        {t('settings.lastUsed')} {formatRelativeTime(token.lastUsedAt, t)}
                      </Text>
                    )}
                  </VStack>
                  <Button
                    size="sm"
                    variant="ghost"
                    colorScheme="red"
                    leftIcon={<Icon as={FiTrash2} />}
                    onClick={() => setDeletingTokenId(token.id)}
                  >
                    {t('settings.deleteKey')}
                  </Button>
                </HStack>
              ))}
            </VStack>
          )}
        </Box>

        {newTokenValue && (
          <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} borderColor="green.500">
            <VStack spacing={4} align="stretch">
              <Heading size="md" color="green.500">{t('settings.tokenCreatedTitle')}</Heading>
              <Text fontSize="sm" color="myGray.600">
                {t('settings.tokenCreatedDesc')}
              </Text>
              <Code p={3} borderRadius="md" bg="myGray.50" fontSize="sm" wordBreak="break-all">
                {newTokenValue}
              </Code>
              <HStack justify="flex-end">
                <Button variant="whiteBase" onClick={() => setNewTokenValue(null)}>
                  {t('common.close')}
                </Button>
                <Button
                  variant="primary"
                  onClick={() => {
                    navigator.clipboard.writeText(newTokenValue)
                    toast({ title: t('settings.tokenCopied'), status: 'success', duration: 2000 })
                  }}
                >
                  {t('settings.copyToken')}
                </Button>
              </HStack>
            </VStack>
          </Box>
        )}

        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
          <VStack spacing={4} align="stretch" mb={4}>
            <Heading size="md">{t('settings.createNewToken')}</Heading>
          </VStack>
          <VStack spacing={4} align="stretch">
            <FormControl>
              <FormLabel>{t('settings.tokenNote')}</FormLabel>
              <Input
                value={newTokenNote}
                onChange={(e) => setNewTokenNote(e.target.value)}
                placeholder={t('settings.tokenNotePlaceholder')}
              />
            </FormControl>
            <HStack justify="flex-end">
              <Button
                variant="primary"
                leftIcon={<Icon as={FiPlus} />}
                onClick={handleCreateToken}
                isLoading={saving}
              >
                {t('settings.createToken')}
              </Button>
            </HStack>
          </VStack>
        </Box>
      </VStack>

      <Modal isOpen={deletingTokenId !== null} onClose={() => setDeletingTokenId(null)}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('settings.deleteTokenTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>{t('settings.deleteTokenDesc')}</Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => setDeletingTokenId(null)}>
              {t('common.cancel')}
            </Button>
            <Button variant="solid" colorScheme="red" onClick={handleConfirmDeleteToken}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </>
  )
}
