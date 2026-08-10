'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  Button,
  VStack,
  HStack,
  Icon,
  Avatar,
  Tabs,
  TabList,
  Tab,
  TabPanels,
  TabPanel,
  IconButton,
  Spinner,
} from '@chakra-ui/react'
import { useGithubToast } from '@/app/providers'
import { useRouter } from 'next/navigation'
import { useEffect, useState, useCallback, useMemo } from 'react'
import { api, Notification, Participant } from '@/lib/api'
import {
  FiBell,
  FiCheckCircle,
  FiAlertCircle,
  FiRotateCcw,
  FiMessageSquare,
  FiAtSign,
  FiUserCheck,
  FiUserPlus,
  FiCheck,
  FiTrash2,
  FiInbox,
} from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { formatRelativeTime } from '@/lib/time'

const TYPE_ICON_MAP: Record<string, { icon: any; color: string }> = {
  issue_open: { icon: FiAlertCircle, color: 'green.500' },
  issue_close: { icon: FiCheckCircle, color: 'purple.500' },
  issue_reopen: { icon: FiRotateCcw, color: 'green.500' },
  issue_comment: { icon: FiMessageSquare, color: 'blue.500' },
  mention: { icon: FiAtSign, color: 'yellow.500' },
  assign: { icon: FiUserCheck, color: 'purple.500' },
  collaborator_added: { icon: FiUserPlus, color: 'green.500' },
  task_assigned: { icon: FiUserCheck, color: 'blue.500' },
  story_assigned: { icon: FiUserCheck, color: 'green.500' },
}

const TYPE_LABEL_KEY: Record<string, string> = {
  issue_open: 'notification.typeIssueOpen',
  issue_close: 'notification.typeIssueClose',
  issue_reopen: 'notification.typeIssueReopen',
  issue_comment: 'notification.typeIssueComment',
  mention: 'notification.typeMention',
  assign: 'notification.typeAssign',
  collaborator_added: 'notification.typeCollaboratorAdded',
  task_assigned: 'notification.typeTaskAssigned',
  story_assigned: 'notification.typeStoryAssigned',
}

export default function NotificationsPage() {
  const router = useRouter()
  const toast = useGithubToast()
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'

  const [notifications, setNotifications] = useState<Notification[]>([])
  const [participants, setParticipants] = useState<Participant[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<'all' | 'unread'>('all')
  const [actionLoading, setActionLoading] = useState(false)

  const loadNotifications = useCallback(async () => {
    try {
      const data = await api.listNotifications(activeTab === 'unread')
      setNotifications(data?.notifications || [])
      setParticipants(data?.participants || [])
    } catch (err: any) {
      toast({
        title: t('notification.loadFailed'),
        description: err.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }, [activeTab])

  useEffect(() => {
    setLoading(true)
    loadNotifications()
  }, [loadNotifications])

  // Sideload participants 查找表：actor userName → 头像信息，避免每条通知都 .find
  const participantMap = useMemo(() => {
    const m = new Map<string, Participant>()
    for (const p of participants) m.set(p.userName, p)
    return m
  }, [participants])

  const handleMarkAsRead = async (slug: string) => {
    try {
      await api.markNotificationAsRead(slug)
      setNotifications((prev) =>
        prev.map((n) => (n.slug === slug ? { ...n, read: true } : n))
      )
      toast({ title: t('notification.markedAsRead'), status: 'success', duration: 2000 })
    } catch (err: any) {
      toast({
        title: t('notification.markAsReadFailed'),
        description: err.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleMarkAllAsRead = async () => {
    setActionLoading(true)
    try {
      await api.markAllNotificationsAsRead()
      setNotifications((prev) => prev.map((n) => ({ ...n, read: true })))
      toast({ title: t('notification.allMarkedAsRead'), status: 'success', duration: 2000 })
    } catch (err: any) {
      toast({
        title: t('notification.markAsReadFailed'),
        description: err.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setActionLoading(false)
    }
  }

  const handleDelete = async (slug: string) => {
    try {
      await api.deleteNotification(slug)
      setNotifications((prev) => prev.filter((n) => n.slug !== slug))
      toast({ title: t('notification.deleted'), status: 'success', duration: 2000 })
    } catch (err: any) {
      toast({
        title: t('notification.deleteFailed'),
        description: err.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleClickNotification = (notification: Notification) => {
    // Mark as read on click
    if (!notification.read) {
      handleMarkAsRead(notification.slug)
    }
    // Navigate to task if available (Scrum task assignment)
    // 深链到 tasks 页面并自动打开任务抽屉
    if (notification.taskId && notification.projectSlug) {
      router.push(`/projects/${notification.projectSlug}/tasks?task=${notification.taskId}`)
    } else if (notification.storyId && notification.storySlug && notification.projectSlug) {
      // 跳转到故事详情页
      router.push(`/projects/${notification.projectSlug}/story/${notification.storySlug}`)
    } else if (notification.issueId) {
      // Navigate to issue/MR if available
      router.push(
        `/${notification.repositoryUserName}/${notification.repositoryName}/issues/${notification.issueId}`
      )
    } else if (notification.repositoryUserName && notification.repositoryName) {
      // Otherwise navigate to the repository
      router.push(`/${notification.repositoryUserName}/${notification.repositoryName}`)
    }
  }

  const getTypeIcon = (type: string) => {
    return TYPE_ICON_MAP[type] || { icon: FiBell, color: 'myGray.500' }
  }

  const getTypeLabel = (type: string) => {
    const key = TYPE_LABEL_KEY[type] || 'notification.typeDefault'
    return t(key)
  }

  const unreadCount = notifications.filter((n) => !n.read).length

  return (
    <Container maxW="container.xl" py={8}>
      <VStack spacing={6} align="stretch">
        <HStack justify="space-between" align="center">
          <Heading size="lg" color="myGray.900">
            {t('notification.title')}
          </Heading>
          <Button
            size="sm"
            variant="outline"
            leftIcon={<Icon as={FiCheck} />}
            onClick={handleMarkAllAsRead}
            isLoading={actionLoading}
            isDisabled={unreadCount === 0}
          >
            {t('notification.markAllAsRead')}
          </Button>
        </HStack>

        <Tabs
          variant="line"
          colorScheme="primary"
          index={activeTab === 'all' ? 0 : 1}
          onChange={(index) => setActiveTab(index === 0 ? 'all' : 'unread')}
        >
          <TabList borderBottomColor="myGray.200">
            <Tab>
              <HStack spacing={2}>
                <Icon as={FiInbox} />
                <Text>{t('notification.all')}</Text>
              </HStack>
            </Tab>
            <Tab>
              <HStack spacing={2}>
                <Icon as={FiBell} />
                <Text>{t('notification.unread')}</Text>
                {unreadCount > 0 && (
                  <Box bg="primary.500" color="white" fontSize="xs" px={1.5} borderRadius="full">
                    {unreadCount}
                  </Box>
                )}
              </HStack>
            </Tab>
          </TabList>

          <TabPanels>
            <TabPanel px={0} pt={4}>
              {renderList()}
            </TabPanel>
            <TabPanel px={0} pt={4}>
              {renderList()}
            </TabPanel>
          </TabPanels>
        </Tabs>
      </VStack>
    </Container>
  )

  function renderList() {
    if (loading) {
      return (
        <Box py={12} textAlign="center">
          <Spinner size="lg" />
        </Box>
      )
    }

    if (notifications.length === 0) {
      return (
        <Box py={12} textAlign="center" borderWidth="1px" borderRadius="lg" bg="white" borderColor="myGray.200">
          <Icon as={FiInbox} w={10} h={10} color="myGray.300" mb={3} />
          <Text color="myGray.500">
            {activeTab === 'unread' ? t('notification.noUnreadNotifications') : t('notification.noNotifications')}
          </Text>
        </Box>
      )
    }

    return (
      <VStack spacing={0} align="stretch" borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
        {notifications.map((notification) => {
          const { icon: TypeIcon, color } = getTypeIcon(notification.notificationType)
          const repoFull = notification.repositoryUserName && notification.repositoryName
            ? `${notification.repositoryUserName}/${notification.repositoryName}`
            : ''
          const isTaskNotification = !!notification.taskId
          return (
            <Box
              key={notification.slug}
              px={4}
              py={3}
              borderBottom="1px"
              borderColor="myGray.100"
              _last={{ borderBottom: 'none' }}
              _hover={{ bg: notification.read ? 'myGray.50' : 'blue.50' }}
              bg={notification.read ? 'white' : 'blue.50'}
              cursor={(notification.taskId || notification.storyId || notification.issueId || notification.repositoryName) ? 'pointer' : 'default'}
              onClick={() => handleClickNotification(notification)}
            >
              <HStack spacing={3} align="start">
                {!notification.read && <Box w={2} h={2} borderRadius="full" bg="primary.500" mt={2} flexShrink={0} />}
                {notification.read && <Box w={2} flexShrink={0} />}
                <Icon as={TypeIcon} color={color} w={5} h={5} mt={0.5} flexShrink={0} />
                <VStack align="start" spacing={1} flex={1} minW={0}>
                  <HStack spacing={2} flexWrap="wrap">
                    <Avatar size="2xs" src={participantMap.get(notification.actor)?.image || undefined} name={notification.actor} />
                    <Text fontSize="sm" fontWeight="medium" color="myGray.900">
                      {participantMap.get(notification.actor)?.fullName || notification.actor}
                    </Text>
                    <Text fontSize="xs" color="myGray.500">
                      {getTypeLabel(notification.notificationType)}
                    </Text>
                  </HStack>
                  <Text fontSize="sm" color="myGray.700" noOfLines={2}>
                    {notification.message}
                  </Text>
                  <HStack spacing={2} fontSize="xs" color="myGray.500">
                    {repoFull && <Text>{repoFull}</Text>}
                    {isTaskNotification && (
                      <>
                        <Text>·</Text>
                        <Text>#{notification.taskId}</Text>
                      </>
                    )}
                    {notification.issueId && (
                      <>
                        <Text>·</Text>
                        <Text>#{notification.issueId}</Text>
                      </>
                    )}
                    <Text>·</Text>
                    <Text>{formatRelativeTime(notification.registeredDate, t)}</Text>
                  </HStack>
                </VStack>
                <HStack spacing={1} flexShrink={0}>
                  {!notification.read && (
                    <IconButton
                      aria-label={t('notification.markAsRead')}
                      icon={<Icon as={FiCheck} />}
                      size="sm"
                      variant="ghost"
                      onClick={(e) => {
                        e.stopPropagation()
                        handleMarkAsRead(notification.slug)
                      }}
                    />
                  )}
                  <IconButton
                    aria-label={t('common.delete')}
                    icon={<Icon as={FiTrash2} />}
                    size="sm"
                    variant="ghost"
                    colorScheme="red"
                    onClick={(e) => {
                      e.stopPropagation()
                      handleDelete(notification.slug)
                    }}
                  />
                </HStack>
              </HStack>
            </Box>
          )
        })}
      </VStack>
    )
  }
}
