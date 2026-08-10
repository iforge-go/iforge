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
} from '@chakra-ui/react'
import { FiCalendar } from 'react-icons/fi'
import { useParams, useRouter } from 'next/navigation'
import { useState } from 'react'
import { api } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

export default function NewMilestonePage() {
  const params = useParams()
  const router = useRouter()
  const toast = useGithubToast()
  const { t } = useI18n()
  const owner = params.owner as string
  const repo = params.repo as string

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [dueDate, setDueDate] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async () => {
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
      setLoading(true)
      await api.createMilestone(
        owner,
        repo,
        title,
        description,
        dueDate ? new Date(dueDate).toISOString() : undefined
      )
      toast({
        title: t('common.success'),
        description: t('repo.milestoneCreated'),
        status: 'success',
        duration: 3000,
      })
      router.push(`/${owner}/${repo}/milestones`)
    } catch (error: any) {
      toast({
        title: t('common.error'),
        description: error.message || t('repo.createFailed'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <Container maxW="container.md" py={8}>
      <VStack spacing={6} align="stretch">
        <Heading size="md">{t('repo.newMilestone')}</Heading>

        <Box bg="white" p={6} borderRadius="md" borderWidth="1px">
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
              <Button
                variant="ghost"
                onClick={() => router.push(`/${owner}/${repo}/milestones`)}
              >
                {t('common.cancel')}
              </Button>
              <Button
                colorScheme="blue"
                onClick={handleSubmit}
                isLoading={loading}
              >
                {t('repo.createMilestone')}
              </Button>
            </HStack>
          </VStack>
        </Box>
      </VStack>
    </Container>
  )
}
