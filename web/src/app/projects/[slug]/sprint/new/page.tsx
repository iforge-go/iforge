'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  Button,
  VStack,
  HStack,
  FormControl,
  FormLabel,
  Input,
  Textarea,
  Select,
  useToast,
  Card,
  CardBody,
} from '@chakra-ui/react'
import { useState } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'

export default function NewSprintPage() {
  const router = useRouter()
  const params = useParams()
  const projectSlug = params.slug as string
  const { t } = useI18n()
  const toast = useToast()
  const { user, authLoading } = useCurrentUser()

  const [formData, setFormData] = useState({
    title: '',
    description: '',
    goal: '',
    status: 'open',
    startDate: '',
    endDate: '',
  })
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!formData.title) {
      toast({
        title: t('common.error'),
        description: t('pms.titleAndSlugRequired'),
        status: 'error',
        duration: 3000,
      })
      return
    }

    setIsSubmitting(true)
    try {
      const sprint = await api.createSprint(projectSlug, {
        title: formData.title,
        description: formData.description || undefined,
        goal: formData.goal || undefined,
        status: formData.status,
        startDate: formData.startDate || undefined,
        endDate: formData.endDate || undefined,
      })
      
      toast({
        title: t('pms.sprintCreated'),
        description: t('pms.sprintCreatedDesc'),
        status: 'success',
        duration: 3000,
      })
      
      router.push(`/projects/${projectSlug}/sprint/${sprint.slug}`)
    } catch (error: any) {
      toast({
        title: t('common.error'),
        description: error.message || t('pms.createFailed'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  if (authLoading) return null

  if (!user) {
    return (
      <Container maxW="container.md" py={20}>
        <VStack spacing={6} textAlign="center">
          <Heading size="lg">{t('common.pleaseLogin')}</Heading>
          <Button colorScheme="blue" onClick={() => router.push('/login')}>
            {t('auth.login')}
          </Button>
        </VStack>
      </Container>
    )
  }

  return (
    <Container maxW="container.md" py={8}>
      <VStack spacing={8} align="stretch">
        <VStack align="start" spacing={2}>
          <Heading size="lg">{t('pms.newSprint')}</Heading>
          <Text color="gray.600">{t('pms.newSprintDesc')}</Text>
        </VStack>

        <Card>
          <CardBody>
            <form onSubmit={handleSubmit}>
              <VStack spacing={6}>
                <FormControl isRequired>
                  <FormLabel>{t('pms.sprintTitle')}</FormLabel>
                  <Input
                    value={formData.title}
                    onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                    placeholder={t('pms.sprintTitlePlaceholder')}
                  />
                </FormControl>

                <FormControl>
                  <FormLabel>{t('pms.sprintGoal')}</FormLabel>
                  <Textarea
                    value={formData.goal}
                    onChange={(e) => setFormData({ ...formData, goal: e.target.value })}
                    placeholder={t('pms.sprintGoalPlaceholder')}
                    rows={3}
                  />
                </FormControl>

                <FormControl>
                  <FormLabel>{t('pms.sprintDescription')}</FormLabel>
                  <Textarea
                    value={formData.description}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    placeholder={t('pms.sprintDescriptionPlaceholder')}
                    rows={4}
                  />
                </FormControl>

                <FormControl>
                  <FormLabel>{t('pms.sprintStatus')}</FormLabel>
                  <Select
                    value={formData.status}
                    onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                  >
                    <option value="open">{t('pms.statusOpen')}</option>
                    <option value="active">{t('pms.statusActive')}</option>
                    <option value="closed">{t('pms.statusClosed')}</option>
                  </Select>
                </FormControl>

                <HStack spacing={4} w="100%">
                  <FormControl>
                    <FormLabel>{t('pms.startDate')}</FormLabel>
                    <Input
                      type="date"
                      value={formData.startDate}
                      onChange={(e) => setFormData({ ...formData, startDate: e.target.value })}
                    />
                  </FormControl>

                  <FormControl>
                    <FormLabel>{t('pms.endDate')}</FormLabel>
                    <Input
                      type="date"
                      value={formData.endDate}
                      onChange={(e) => setFormData({ ...formData, endDate: e.target.value })}
                    />
                  </FormControl>
                </HStack>

                <HStack spacing={4} w="100%">
                  <Button
                    variant="outline"
                    onClick={() => router.push(`/projects/${projectSlug}`)}
                    flex={1}
                  >
                    {t('common.cancel')}
                  </Button>
                  <Button
                    type="submit"
                    colorScheme="blue"
                    isLoading={isSubmitting}
                    flex={1}
                  >
                    {t('pms.createSprint')}
                  </Button>
                </HStack>
              </VStack>
            </form>
          </CardBody>
        </Card>
      </VStack>
    </Container>
  )
}
