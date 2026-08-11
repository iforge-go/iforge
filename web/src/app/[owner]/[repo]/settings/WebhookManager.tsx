'use client'

import {
  Box,
  Button,
  VStack,
  HStack,
  Text,
  Heading,
  IconButton,
  Badge,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  FormControl,
  FormLabel,
  Input,
  Select,
  Switch,
  Checkbox,
  CheckboxGroup,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  useDisclosure,
  Tooltip,
} from '@chakra-ui/react'
import { FiPlus, FiTrash2, FiSend, FiClock, FiCheck, FiX } from 'react-icons/fi'
import { useState, useEffect, useCallback } from 'react'
import { api, Webhook, WebhookDelivery } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useGithubToast } from '@/app/providers'
import { ConfirmDialog } from '@/components/ConfirmDialog'

interface WebhookManagerProps {
  owner: string
  repoName: string
}

export default function WebhookManager({ owner, repoName }: WebhookManagerProps) {
  const { t } = useI18n()
  const toast = useGithubToast()
  const [webhooks, setWebhooks] = useState<Webhook[]>([])
  const [loading, setLoading] = useState(true)
  const { isOpen: isCreateOpen, onOpen: onCreateOpen, onClose: onCreateClose } = useDisclosure()
  const { isOpen: isDeliveriesOpen, onOpen: onDeliveriesOpen, onClose: onDeliveriesClose } = useDisclosure()
  const [selectedWebhook, setSelectedWebhook] = useState<Webhook | null>(null)
  const [deliveries, setDeliveries] = useState<WebhookDelivery[]>([])
  const [loadingDeliveries, setLoadingDeliveries] = useState(false)

  // Create form state
  const [newUrl, setNewUrl] = useState('')
  const [newContentType, setNewContentType] = useState('application/json')
  const [newToken, setNewToken] = useState('')
  const [newEvents, setNewEvents] = useState<string[]>(['push', 'issues', 'pull_request'])
  const [newActive, setNewActive] = useState(true)
  const [newInsecureSSL, setNewInsecureSSL] = useState(false)
  const [creating, setCreating] = useState(false)

  const [confirmOpen, setConfirmOpen] = useState(false)
  const [confirmAction, setConfirmAction] = useState<(() => void) | null>(null)

  const loadWebhooks = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.listWebhooks(owner, repoName)
      setWebhooks(data || [])
    } catch (error: any) {
      toast({
        title: t('webhook.loadFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }, [owner, repoName, toast, t])

  useEffect(() => {
    loadWebhooks()
  }, [loadWebhooks])

  const handleCreate = async () => {
    if (!newUrl) {
      toast({
        title: t('webhook.urlRequired'),
        status: 'warning',
        duration: 2000,
      })
      return
    }

    setCreating(true)
    try {
      await api.createWebhook(owner, repoName, {
        url: newUrl,
        contentType: newContentType,
        token: newToken || undefined,
        events: newEvents,
        active: newActive,
        insecureSSL: newInsecureSSL,
      })
      toast({
        title: t('webhook.created'),
        status: 'success',
        duration: 2000,
      })
      onCreateClose()
      resetForm()
      loadWebhooks()
    } catch (error: any) {
      toast({
        title: t('webhook.createFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = (id: number) => {
    setConfirmAction(() => async () => {
      try {
        await api.deleteWebhook(owner, repoName, id)
        toast({
          title: t('webhook.deleted'),
          status: 'success',
          duration: 2000,
        })
        loadWebhooks()
      } catch (error: any) {
        toast({
          title: t('webhook.deleteFailed'),
          description: error.message,
          status: 'error',
          duration: 3000,
        })
      }
    })
    setConfirmOpen(true)
  }

  const handleConfirm = () => {
    confirmAction?.()
    setConfirmOpen(false)
    setConfirmAction(null)
  }

  const handleTest = async (id: number) => {
    try {
      await api.testWebhook(owner, repoName, id)
      toast({
        title: t('webhook.testSent'),
        status: 'success',
        duration: 2000,
      })
    } catch (error: any) {
      toast({
        title: t('webhook.testFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleViewDeliveries = async (webhook: Webhook) => {
    setSelectedWebhook(webhook)
    onDeliveriesOpen()
    setLoadingDeliveries(true)
    try {
      const data = await api.listWebhookDeliveries(owner, repoName, webhook.id)
      setDeliveries(data || [])
    } catch (error: any) {
      toast({
        title: t('webhook.loadDeliveriesFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoadingDeliveries(false)
    }
  }

  const resetForm = () => {
    setNewUrl('')
    setNewContentType('application/json')
    setNewToken('')
    setNewEvents(['push', 'issues', 'pull_request'])
    setNewActive(true)
    setNewInsecureSSL(false)
  }

  const eventOptions = [
    { value: 'push', label: t('webhook.eventPush') },
    { value: 'issues', label: t('webhook.eventIssues') },
    { value: 'issue_comment', label: t('webhook.eventIssueComment') },
    { value: 'pull_request', label: t('webhook.eventMergeRequest') },
    { value: 'release', label: t('webhook.eventRelease') },
  ]

  return (
    <>
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
        <Box px={6} pt={6} pb={2}>
          <Heading size="md">{t('webhook.title')}</Heading>
          <Text fontSize="sm" color="myGray.600" mt={1}>
            {t('webhook.description')}
          </Text>
        </Box>
        <Box p={6}>
          <VStack spacing={4} align="stretch">
            <HStack justify="space-between">
              <Text fontSize="sm" color="myGray.600">
                {webhooks.length} {t('webhook.activeWebhooks')}
              </Text>
              <Button leftIcon={<FiPlus />} size="sm" variant="primary" onClick={onCreateOpen}>
                {t('webhook.addWebhook')}
              </Button>
            </HStack>

            {loading ? (
              <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
            ) : webhooks.length === 0 ? (
              <Box py={8} textAlign="center">
                <Text color="myGray.500">{t('webhook.noWebhooks')}</Text>
              </Box>
            ) : (
              <Table variant="simple" size="sm">
                <Thead>
                  <Tr>
                    <Th>{t('webhook.url')}</Th>
                    <Th>{t('webhook.events')}</Th>
                    <Th>{t('webhook.status')}</Th>
                    <Th>{t('webhook.actions')}</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {webhooks.map((webhook) => (
                    <Tr key={webhook.id}>
                      <Td>
                        <Text fontSize="sm" noOfLines={1}>
                          {webhook.url}
                        </Text>
                      </Td>
                      <Td>
                        <HStack spacing={1} flexWrap="wrap">
                          {webhook.events && webhook.events.length > 0 ? (
                            webhook.events.map((event) => (
                              <Badge key={event} colorScheme="blue" fontSize="xs">
                                {event}
                              </Badge>
                            ))
                          ) : (
                            <Badge colorScheme="gray" fontSize="xs">
                              {t('webhook.allEvents')}
                            </Badge>
                          )}
                        </HStack>
                      </Td>
                      <Td>
                        <Badge colorScheme={webhook.active ? 'green' : 'gray'}>
                          {webhook.active ? t('webhook.active') : t('webhook.inactive')}
                        </Badge>
                      </Td>
                      <Td>
                        <HStack spacing={1}>
                          <Tooltip label={t('webhook.viewDeliveries')}>
                            <IconButton
                              aria-label={t('webhook.viewDeliveries')}
                              icon={<FiClock />}
                              size="xs"
                              variant="ghost"
                              onClick={() => handleViewDeliveries(webhook)}
                            />
                          </Tooltip>
                          <Tooltip label={t('webhook.test')}>
                            <IconButton
                              aria-label={t('webhook.test')}
                              icon={<FiSend />}
                              size="xs"
                              variant="ghost"
                              onClick={() => handleTest(webhook.id)}
                            />
                          </Tooltip>
                          <Tooltip label={t('webhook.delete')}>
                            <IconButton
                              aria-label={t('webhook.delete')}
                              icon={<FiTrash2 />}
                              size="xs"
                              variant="ghost"
                              colorScheme="red"
                              onClick={() => handleDelete(webhook.id)}
                            />
                          </Tooltip>
                        </HStack>
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            )}
          </VStack>
        </Box>
      </Box>

      {/* Create Webhook Modal */}
      <Modal isOpen={isCreateOpen} onClose={onCreateClose} size="xl">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('webhook.createWebhook')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('webhook.payloadUrl')}</FormLabel>
                <Input
                  value={newUrl}
                  onChange={(e) => setNewUrl(e.target.value)}
                  placeholder="https://example.com/webhook"
                />
              </FormControl>

              <FormControl>
                <FormLabel>{t('webhook.contentType')}</FormLabel>
                <Select value={newContentType} onChange={(e) => setNewContentType(e.target.value)}>
                  <option value="application/json">application/json</option>
                  <option value="application/x-www-form-urlencoded">application/x-www-form-urlencoded</option>
                </Select>
              </FormControl>

              <FormControl>
                <FormLabel>{t('webhook.secret')}</FormLabel>
                <Input
                  type="password"
                  value={newToken}
                  onChange={(e) => setNewToken(e.target.value)}
                  placeholder={t('webhook.secretPlaceholder')}
                />
              </FormControl>

              <FormControl>
                <FormLabel>{t('webhook.whichEvents')}</FormLabel>
                <CheckboxGroup value={newEvents} onChange={(val) => setNewEvents(val as string[])}>
                  <VStack align="start" spacing={2}>
                    {eventOptions.map((option) => (
                      <Checkbox key={option.value} value={option.value}>
                        {option.label}
                      </Checkbox>
                    ))}
                  </VStack>
                </CheckboxGroup>
              </FormControl>

              <FormControl display="flex" alignItems="center">
                <FormLabel mb={0}>{t('webhook.active')}</FormLabel>
                <Switch
                  isChecked={newActive}
                  onChange={(e) => setNewActive(e.target.checked)}
                  colorScheme="primary"
                />
              </FormControl>

              <FormControl display="flex" alignItems="center">
                <FormLabel mb={0}>{t('webhook.insecureSSL')}</FormLabel>
                <Switch
                  isChecked={newInsecureSSL}
                  onChange={(e) => setNewInsecureSSL(e.target.checked)}
                  colorScheme="primary"
                />
                <Text fontSize="xs" color="myGray.500" ml={2}>
                  {t('webhook.insecureSSLDesc')}
                </Text>
              </FormControl>
            </VStack>
          </ModalBody>

          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onCreateClose}>
              {t('common.cancel')}
            </Button>
            <Button variant="primary" onClick={handleCreate} isLoading={creating}>
              {t('webhook.createWebhook')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      <ConfirmDialog
        isOpen={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={handleConfirm}
        title={t('common.confirm')}
        message={t('webhook.confirmDelete')}
      />

      {/* Deliveries Modal */}
      <Modal isOpen={isDeliveriesOpen} onClose={onDeliveriesClose} size="6xl">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>
            {t('webhook.recentDeliveries')} - {selectedWebhook?.url}
          </ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            {loadingDeliveries ? (
              <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
            ) : deliveries.length === 0 ? (
              <Box py={8} textAlign="center">
                <Text color="myGray.500">{t('webhook.noDeliveries')}</Text>
              </Box>
            ) : (
              <Table variant="simple" size="sm">
                <Thead>
                  <Tr>
                    <Th>{t('webhook.event')}</Th>
                    <Th>{t('webhook.statusCode')}</Th>
                    <Th>{t('webhook.duration')}</Th>
                    <Th>{t('webhook.status')}</Th>
                    <Th>{t('webhook.deliveredAt')}</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {deliveries.map((delivery) => (
                    <Tr key={delivery.id}>
                      <Td>
                        <Text fontSize="sm">{delivery.event}</Text>
                        {delivery.action && (
                          <Text fontSize="xs" color="myGray.500">
                            {delivery.action}
                          </Text>
                        )}
                      </Td>
                      <Td>
                        <Badge colorScheme={delivery.statusCode >= 200 && delivery.statusCode < 300 ? 'green' : 'red'}>
                          {delivery.statusCode}
                        </Badge>
                      </Td>
                      <Td>
                        <Text fontSize="sm">{delivery.duration}ms</Text>
                      </Td>
                      <Td>
                        {delivery.success ? (
                          <HStack spacing={1}>
                            <FiCheck color="green" />
                            <Text fontSize="sm" color="green.500">{t('webhook.success')}</Text>
                          </HStack>
                        ) : (
                          <HStack spacing={1}>
                            <FiX color="red" />
                            <Text fontSize="sm" color="red.500">{t('webhook.failed')}</Text>
                          </HStack>
                        )}
                      </Td>
                      <Td>
                        <Text fontSize="xs">
                          {new Date(delivery.createdAt).toLocaleString()}
                        </Text>
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            )}
          </ModalBody>

          <ModalFooter>
            <Button variant="ghost" onClick={onDeliveriesClose}>
              {t('common.close')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </>
  )
}
