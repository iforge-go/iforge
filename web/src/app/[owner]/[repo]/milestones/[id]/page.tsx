'use client'

import {
  Box,
  Button,
  Container,
  FormControl,
  FormLabel,
  Heading,
  Input,
  Textarea,
  VStack,
  HStack,
  Icon,
  Text,
  Badge,
} from '@chakra-ui/react'
import { FiCalendar, FiEdit2, FiTrash2, FiCheckCircle, FiXCircle } from 'react-icons/fi'
import { useParams, useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'
import { api, Milestone } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { ConfirmDialog } from '@/components/ConfirmDialog'

export default function MilestoneDetailPage() {
  const params = useParams()
  const router = useRouter()
  const toast = useGithubToast()
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'
  const owner = params.owner as string
  const repo = params.repo as string
  const milestoneId = parseInt(params.id as string)
  const { userRole } = useRepo()

  const [milestone, setMilestone] = useState<Milestone | null>(null)
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState(false)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [dueDate, setDueDate] = useState('')
  const [saving, setSaving] = useState(false)

  const [confirmOpen, setConfirmOpen] = useState(false)
  const [confirmAction, setConfirmAction] = useState<(() => void) | null>(null)

  // Developer 及以上权限可以管理里程碑
  const canManageMilestone = userRole === 'owner' || userRole === 'member'

  const loadMilestone = async () => {
    try {
      setLoading(true)
      const data = await api.getMilestone(owner, repo, milestoneId)
      setMilestone(data)
      setTitle(data.title)
      setDescription(data.description || '')
      setDueDate(data.dueDate ? data.dueDate.split('T')[0] : '')
    } catch (error) {
      console.error('Failed to load milestone:', error)
      toast({
        title: t('common.error'),
        description: t('repo.loadMilestoneFailed'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadMilestone()
  }, [owner, repo, milestoneId])

  const handleSave = async () => {
    if (!title.trim()) {
      toast({
        title: t('common.error'),
        description: t('repo.enterMilestoneTitle'),
        status: 'error',
        duration: 3000,
      })
      return
    }

    try {
      setSaving(true)
      await api.updateMilestone(
        owner,
        repo,
        milestoneId,
        title,
        description,
        dueDate ? new Date(dueDate).toISOString() : undefined
      )
      toast({
        title: t('common.success'),
        description: t('repo.milestoneUpdated'),
        status: 'success',
        duration: 3000,
      })
      setEditing(false)
      await loadMilestone()
    } catch (error: any) {
      toast({
        title: t('common.error'),
        description: error.message || t('repo.updateFailed'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = () => {
    setConfirmAction(() => async () => {
      try {
        await api.deleteMilestone(owner, repo, milestoneId)
        toast({
          title: t('common.success'),
          description: t('repo.milestoneDeleted'),
          status: 'success',
          duration: 3000,
        })
        router.push(`/${owner}/${repo}/milestones`)
      } catch (error: any) {
        toast({
          title: t('common.error'),
          description: error.message || t('repo.deleteFailed'),
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

  const handleClose = async () => {
    try {
      await api.closeMilestone(owner, repo, milestoneId)
      toast({
        title: t('common.success'),
        description: t('repo.milestoneClosed'),
        status: 'success',
        duration: 3000,
      })
      await loadMilestone()
    } catch (error: any) {
      toast({
        title: t('common.error'),
        description: error.message || t('repo.operationFailed'),
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleReopen = async () => {
    try {
      await api.reopenMilestone(owner, repo, milestoneId)
      toast({
        title: t('common.success'),
        description: t('repo.milestoneReopened'),
        status: 'success',
        duration: 3000,
      })
      await loadMilestone()
    } catch (error: any) {
      toast({
        title: t('common.error'),
        description: error.message || t('repo.operationFailed'),
        status: 'error',
        duration: 3000,
      })
    }
  }

  if (loading) {
    return (
      <Container maxW="container.md" py={8}>
        <Text>{t('common.loading')}</Text>
      </Container>
    )
  }

  if (!milestone) {
    return (
      <Container maxW="container.md" py={8}>
        <Text>{t('repo.milestoneNotExist')}</Text>
      </Container>
    )
  }

  const isClosed = !!milestone.closedDate

  return (
    <Container maxW="container.md" py={8}>
      <VStack spacing={6} align="stretch">
        <HStack justify="space-between">
          <HStack spacing={3}>
            <Heading size="md">{milestone.title}</Heading>
            {isClosed && (
              <Badge colorScheme="purple" fontSize="sm">
                {t('repo.closed')}
              </Badge>
            )}
          </HStack>
          {!editing && canManageMilestone && (
            <HStack spacing={2}>
              {isClosed ? (
                <Button
                  leftIcon={<Icon as={FiXCircle} />}
                  size="sm"
                  onClick={handleReopen}
                >
                  {t('repo.reopen')}
                </Button>
              ) : (
                <Button
                  leftIcon={<Icon as={FiCheckCircle} />}
                  size="sm"
                  colorScheme="purple"
                  onClick={handleClose}
                >
                  {t('repo.close')}
                </Button>
              )}
              <Button
                leftIcon={<Icon as={FiEdit2} />}
                size="sm"
                onClick={() => setEditing(true)}
              >
                {t('repo.edit')}
              </Button>
              <Button
                leftIcon={<Icon as={FiTrash2} />}
                size="sm"
                colorScheme="red"
                variant="ghost"
                onClick={handleDelete}
              >
                {t('repo.delete')}
              </Button>
            </HStack>
          )}
        </HStack>

        <Box bg="white" p={6} borderRadius="md" borderWidth="1px">
          {editing ? (
            <VStack spacing={4} align="stretch">
              <FormControl isRequired>
                <FormLabel>{t('repo.title')}</FormLabel>
                <Input
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder={t('repo.enterMilestoneTitle')}
                />
              </FormControl>

              <FormControl>
                <FormLabel>{t('repo.description')}</FormLabel>
                <Textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder={t('repo.enterMilestoneDescription')}
                  rows={4}
                />
              </FormControl>

              <FormControl>
                <FormLabel>
                  <HStack spacing={2}>
                    <Icon as={FiCalendar} />
                    <Text>{t('repo.dueDateOptional')}</Text>
                  </HStack>
                </FormLabel>
                <Input
                  type="date"
                  value={dueDate}
                  onChange={(e) => setDueDate(e.target.value)}
                />
              </FormControl>

              <HStack justify="flex-end" pt={4}>
                <Button variant="ghost" onClick={() => setEditing(false)}>
                  {t('repo.cancel')}
                </Button>
                <Button
                  colorScheme="blue"
                  onClick={handleSave}
                  isLoading={saving}
                >
                  {t('repo.save')}
                </Button>
              </HStack>
            </VStack>
          ) : (
            <VStack spacing={4} align="stretch">
              {milestone.description && (
                <Box>
                  <Text fontWeight="medium" mb={2}>
                    {t('repo.description')}
                  </Text>
                  <Text color="gray.700" whiteSpace="pre-wrap">
                    {milestone.description}
                  </Text>
                </Box>
              )}

              {milestone.dueDate && (
                <Box>
                  <HStack spacing={2}>
                    <Icon as={FiCalendar} color="gray.500" />
                    <Text fontWeight="medium">{t('repo.dueDate')}</Text>
                  </HStack>
                  <Text color="gray.700" ml={6}>
                    {new Date(milestone.dueDate).toLocaleDateString(dateLocale)}
                  </Text>
                </Box>
              )}

              {milestone.closedDate && (
                <Box>
                  <HStack spacing={2}>
                    <Icon as={FiCheckCircle} color="purple.500" />
                    <Text fontWeight="medium">{t('repo.closedAt')}</Text>
                  </HStack>
                  <Text color="gray.700" ml={6}>
                    {new Date(milestone.closedDate).toLocaleDateString(dateLocale)}
                  </Text>
                </Box>
              )}
            </VStack>
          )}
        </Box>

        <ConfirmDialog
          isOpen={confirmOpen}
          onClose={() => setConfirmOpen(false)}
          onConfirm={handleConfirm}
          title={t('common.confirm')}
          message={t('repo.confirmDeleteMilestone')}
        />
      </VStack>
    </Container>
  )
}
