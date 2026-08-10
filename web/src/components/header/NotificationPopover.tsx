'use client'

import {
  Avatar,
  Box,
  Button,
  Flex,
  HStack,
  Icon,
  IconButton,
  Popover,
  PopoverArrow,
  PopoverBody,
  PopoverContent,
  PopoverFooter,
  PopoverHeader,
  PopoverTrigger,
  Spinner,
  Text,
  VStack,
} from '@chakra-ui/react'
import { useRouter } from 'next/navigation'
import { useEffect, useState, useCallback, useRef, useMemo } from 'react'
import { FiBell, FiCheck, FiTrash2, FiInbox } from 'react-icons/fi'
import { api } from '@/lib/api'
import type { Notification, Participant } from '@/lib/types'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import { useWebSocket, useWebSocketStatus } from '@/contexts/WebSocketContext'
import { useGithubToast } from '@/app/providers'
import { formatRelativeTime } from '@/lib/time'

interface NotificationPopoverProps {
  unreadCount: number
  onUnreadCountChange: (count: number) => void
}

export function NotificationPopover({ unreadCount, onUnreadCountChange }: NotificationPopoverProps) {
  const router = useRouter()
  const { user } = useCurrentUser()
  const [recentUnread, setRecentUnread] = useState<Notification[]>([])
  const [recentParticipants, setRecentParticipants] = useState<Participant[]>([])
  const [popoverOpen, setPopoverOpen] = useState(false)
  const [loadingRecent, setLoadingRecent] = useState(false)
  const lastFetchAtRef = useRef<number>(0)
  const { t } = useI18n()
  const toast = useGithubToast()

  // WebSocket 实时通知
  useWebSocket('notification', (payload: Notification) => {
    onUnreadCountChange(unreadCount + 1)
    if (!payload) return
    const targetUrl = resolveNotificationTarget(payload)
    toast({
      position: 'top',
      duration: 5000,
      isClosable: true,
      render: ({ onClose }) => (
        <Box
          bg="white"
          borderWidth="1px"
          borderColor="myGray.200"
          borderRadius="md"
          boxShadow="md"
          p={3}
          minW="280px"
          maxW="360px"
          cursor={targetUrl ? 'pointer' : 'default'}
          onClick={() => {
            if (!targetUrl) return
            onClose()
            router.push(targetUrl)
          }}
        >
          <HStack spacing={2} align="start">
            <Icon as={FiBell} color="primary.500" mt={0.5} flexShrink={0} />
            <VStack align="start" spacing={0.5} flex={1} minW={0}>
              <Text fontSize="sm" fontWeight="medium" color="myGray.900">
                {payload.actor}
              </Text>
              <Text fontSize="xs" color="myGray.600" noOfLines={2}>
                {payload.message}
              </Text>
            </VStack>
          </HStack>
        </Box>
      ),
    })
  })

  const loadRecentUnread = useCallback(async (force = false) => {
    if (!force && Date.now() - lastFetchAtRef.current < 5000 && recentUnread.length > 0) return
    setLoadingRecent(true)
    try {
      const data = await api.listNotifications(true)
      setRecentUnread(data?.notifications || [])
      setRecentParticipants(data?.participants || [])
      lastFetchAtRef.current = Date.now()
    } catch {
      // 静默失败
    } finally {
      setLoadingRecent(false)
    }
  }, [recentUnread.length])

  const participantMap = useMemo(() => {
    const m = new Map<string, Participant>()
    for (const p of recentParticipants) m.set(p.userName, p)
    return m
  }, [recentParticipants])

  useEffect(() => {
    if (popoverOpen) {
      loadRecentUnread()
    }
  }, [popoverOpen, loadRecentUnread])

  const handleMarkAsRead = async (slug: string) => {
    try {
      await api.markNotificationAsRead(slug)
      setRecentUnread((prev) => prev.filter((n) => n.slug !== slug))
      onUnreadCountChange(Math.max(0, unreadCount - 1))
    } catch {
      // 静默失败
    }
  }

  const handleDelete = async (slug: string) => {
    try {
      await api.deleteNotification(slug)
      const deleted = recentUnread.find((n) => n.slug !== slug)
      setRecentUnread((prev) => prev.filter((n) => n.slug !== slug))
      if (deleted && !deleted.read) {
        onUnreadCountChange(Math.max(0, unreadCount - 1))
      }
    } catch {
      // 静默失败
    }
  }

  const handleMarkAllAsRead = async () => {
    try {
      await api.markAllNotificationsAsRead()
      setRecentUnread([])
      onUnreadCountChange(0)
    } catch {
      // 静默失败
    }
  }

  const handleClickNotification = (notification: Notification) => {
    if (!notification.read) {
      handleMarkAsRead(notification.slug)
    }
    const targetUrl = resolveNotificationTarget(notification)
    if (targetUrl) {
      setPopoverOpen(false)
      router.push(targetUrl)
    }
  }

  if (!user) return null

  return (
    <Box position="relative" display="inline-flex">
      <Popover
        isOpen={popoverOpen}
        onOpen={() => setPopoverOpen(true)}
        onClose={() => setPopoverOpen(false)}
        placement="bottom-end"
        closeOnBlur
        closeOnEsc
      >
        <PopoverTrigger>
          <IconButton
            aria-label={t('notification.title')}
            icon={<FiBell />}
            size="sm"
            variant="ghost"
          />
        </PopoverTrigger>
        <PopoverContent w="380px" maxW="90vw" boxShadow="lg" borderColor="myGray.200">
          <PopoverArrow />
          <PopoverHeader fontWeight="semibold" fontSize="sm" borderBottomColor="myGray.200">
            <HStack justify="space-between">
              <Text>{t('notification.title')}</Text>
              {unreadCount > 0 && (
                <Button size="xs" variant="ghost" onClick={handleMarkAllAsRead}>
                  {t('notification.markAllAsRead')}
                </Button>
              )}
            </HStack>
          </PopoverHeader>
          <PopoverBody p={0} maxH="400px" overflowY="auto">
            {loadingRecent ? (
              <Flex justify="center" py={6}>
                <Spinner size="sm" />
              </Flex>
            ) : recentUnread.length === 0 ? (
              <VStack py={6} spacing={2} color="myGray.500">
                <Icon as={FiInbox} w={6} h={6} />
                <Text fontSize="sm">{t('notification.noUnreadNotifications')}</Text>
              </VStack>
            ) : (
              <VStack spacing={0} align="stretch">
                {recentUnread.slice(0, 5).map((n) => {
                  const { icon: TypeIcon, color } = getTypeIcon(n.notificationType)
                  const targetUrl = resolveNotificationTarget(n)
                  return (
                    <HStack
                      key={n.slug}
                      spacing={2}
                      align="start"
                      px={3}
                      py={2}
                      borderBottom="1px solid"
                      borderColor="myGray.100"
                      _hover={{ bg: 'blue.50' }}
                      cursor={targetUrl ? 'pointer' : 'default'}
                      onClick={() => handleClickNotification(n)}
                    >
                      <Box w={2} h={2} borderRadius="full" bg="primary.500" mt={2} flexShrink={0} />
                      <Icon as={TypeIcon} color={color} mt={1} flexShrink={0} />
                      <VStack align="start" spacing={0.5} flex={1} minW={0}>
                        <HStack spacing={1} flexWrap="wrap">
                          <Avatar size="2xs" src={participantMap.get(n.actor)?.image || undefined} name={n.actor} />
                          <Text fontSize="xs" fontWeight="medium" color="myGray.900">
                            {participantMap.get(n.actor)?.fullName || n.actor}
                          </Text>
                        </HStack>
                        <Text fontSize="xs" color="myGray.700" noOfLines={2}>
                          {n.message}
                        </Text>
                        <Text fontSize="10px" color="myGray.400">
                          {formatRelativeTime(n.registeredDate, t)}
                        </Text>
                      </VStack>
                      <HStack spacing={0.5} flexShrink={0}>
                        <IconButton
                          aria-label={t('notification.markAsRead')}
                          icon={<Icon as={FiCheck} />}
                          size="xs"
                          variant="ghost"
                          h={6}
                          minW={6}
                          onClick={(e) => {
                            e.stopPropagation()
                            handleMarkAsRead(n.slug)
                          }}
                        />
                        <IconButton
                          aria-label={t('common.delete')}
                          icon={<Icon as={FiTrash2} />}
                          size="xs"
                          variant="ghost"
                          colorScheme="red"
                          h={6}
                          minW={6}
                          onClick={(e) => {
                            e.stopPropagation()
                            handleDelete(n.slug)
                          }}
                        />
                      </HStack>
                    </HStack>
                  )
                })}
                {recentUnread.length > 5 && (
                  <Box px={3} py={2} fontSize="xs" color="myGray.500" bg="myGray.50">
                    {t('notification.moreUnread', { count: recentUnread.length - 5 })}
                  </Box>
                )}
              </VStack>
            )}
          </PopoverBody>
          <PopoverFooter borderTopColor="myGray.200" py={2}>
            <HStack justify="space-between" w="full">
              <Button
                size="xs"
                variant="ghost"
                onClick={() => {
                  setPopoverOpen(false)
                  router.push('/notifications')
                }}
              >
                {t('notification.viewAll')}
              </Button>
              <WebSocketStatusIndicator />
            </HStack>
          </PopoverFooter>
        </PopoverContent>
      </Popover>
      {unreadCount > 0 && (
        <Box
          position="absolute"
          top="-2px"
          right="-2px"
          bg="red.500"
          color="white"
          fontSize="10px"
          fontWeight="bold"
          minW="16px"
          h="16px"
          px="4px"
          borderRadius="full"
          display="flex"
          alignItems="center"
          justifyContent="center"
          pointerEvents="none"
          zIndex={1}
        >
          {unreadCount > 99 ? '99+' : unreadCount}
        </Box>
      )}
    </Box>
  )
}

function resolveNotificationTarget(n: Notification): string | null {
  if (n.taskId && n.projectSlug) {
    return `/projects/${n.projectSlug}/tasks?task=${n.taskId}`
  }
  if (n.storyId && n.storySlug && n.projectSlug) {
    return `/projects/${n.projectSlug}/story/${n.storySlug}`
  }
  if (n.issueId && n.repositoryUserName && n.repositoryName) {
    return `/${n.repositoryUserName}/${n.repositoryName}/issues/${n.issueId}`
  }
  if (n.repositoryUserName && n.repositoryName) {
    return `/${n.repositoryUserName}/${n.repositoryName}`
  }
  return null
}

const TYPE_ICON_MAP: Record<string, { icon: any; color: string }> = {
  issue_open: { icon: FiBell, color: 'green.500' },
  issue_close: { icon: FiBell, color: 'purple.500' },
  issue_reopen: { icon: FiBell, color: 'green.500' },
  issue_comment: { icon: FiBell, color: 'blue.500' },
  mention: { icon: FiBell, color: 'yellow.500' },
  assign: { icon: FiBell, color: 'purple.500' },
  collaborator_added: { icon: FiBell, color: 'green.500' },
  task_assigned: { icon: FiBell, color: 'blue.500' },
  story_assigned: { icon: FiBell, color: 'green.500' },
}

function getTypeIcon(type: string) {
  return TYPE_ICON_MAP[type] || { icon: FiBell, color: 'myGray.500' }
}

function WebSocketStatusIndicator() {
  const isConnected = useWebSocketStatus()
  const { t } = useI18n()
  return (
    <HStack spacing={1} color={isConnected ? 'green.500' : 'myGray.400'}>
      <Box
        w={2}
        h={2}
        borderRadius="full"
        bg={isConnected ? 'green.500' : 'myGray.400'}
        flexShrink={0}
      />
      <Text fontSize="10px" color={isConnected ? 'green.500' : 'myGray.400'}>
        {isConnected ? 'Connected' : 'Disconnected'}
      </Text>
    </HStack>
  )
}
