'use client'

import {
  Container,
  Grid,
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
  NumberInput,
  NumberInputField,
  NumberInputStepper,
  NumberIncrementStepper,
  NumberDecrementStepper,
  useToast,
  Card,
  CardBody,
} from '@chakra-ui/react'
import { useState } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { FiCpu } from 'react-icons/fi'
import { api, AIOptimizeResult } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import AIOptimizeModal from '@/components/AIOptimizeModal'
import UserSelect from '@/components/UserSelect'

export default function NewUserStoryPage() {
  const router = useRouter()
  const params = useParams()
  const projectSlug = params.slug as string
  const { t } = useI18n()
  const toast = useToast()
  const { user, authLoading } = useCurrentUser()

  const [formData, setFormData] = useState({
    title: '',
    description: '',
    status: 'open',
    priority: 'medium',
    storyPoints: 0,
    acceptanceCriteria: '',
    assigneeName: '',
  })
  const [isSubmitting, setIsSubmitting] = useState(false)
  // AI 优化 Modal
  const [isAIOptimizeOpen, setIsAIOptimizeOpen] = useState(false)

  // 应用 AI 优化结果：把优化后的字段回填到表单
  const handleApplyOptimize = (result: AIOptimizeResult) => {
    setFormData((prev) => ({
      ...prev,
      title: result.title || prev.title,
      description: result.description || prev.description,
      acceptanceCriteria: result.acceptanceCriteria || prev.acceptanceCriteria,
      priority: result.priority || prev.priority,
      storyPoints: result.storyPoints ?? prev.storyPoints,
    }))
  }

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
      const story = await api.createUserStory(projectSlug, {
        title: formData.title,
        description: formData.description || undefined,
        status: formData.status,
        priority: formData.priority,
        storyPoints: formData.storyPoints || undefined,
        acceptanceCriteria: formData.acceptanceCriteria || undefined,
        assigneeName: formData.assigneeName || undefined,
      })
      
      toast({
        title: t('pms.userStoryCreated'),
        description: t('pms.userStoryCreatedDesc'),
        status: 'success',
        duration: 3000,
      })
      
      router.push(`/projects/${projectSlug}/story/${story.slug}`)
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
          <HStack justify="space-between" w="100%" align="flex-start">
            <VStack align="start" spacing={2}>
              <Heading size="lg">{t('pms.newUserStory')}</Heading>
              <Text color="gray.600">{t('pms.newUserStoryDesc')}</Text>
            </VStack>
            <Button
              leftIcon={<FiCpu />}
              colorScheme="purple"
              variant="outline"
              onClick={() => setIsAIOptimizeOpen(true)}
              isDisabled={!formData.title}
            >
              {t('pms.aiOptimize')}
            </Button>
          </HStack>
        </VStack>

        <Card>
          <CardBody>
            <form onSubmit={handleSubmit}>
              <VStack spacing={6}>
                <FormControl isRequired>
                  <FormLabel>{t('pms.userStoryTitle')}</FormLabel>
                  <Input
                    value={formData.title}
                    onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                    placeholder={t('pms.userStoryTitlePlaceholder')}
                  />
                </FormControl>

                <FormControl>
                  <FormLabel>{t('pms.userStoryDescription')}</FormLabel>
                  <Textarea
                    value={formData.description}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    placeholder={t('pms.userStoryDescriptionPlaceholder')}
                    rows={6}
                  />
                </FormControl>

                <FormControl>
                  <FormLabel>{t('pms.acceptanceCriteria')}</FormLabel>
                  <Textarea
                    value={formData.acceptanceCriteria}
                    onChange={(e) => setFormData({ ...formData, acceptanceCriteria: e.target.value })}
                    placeholder={t('pms.acceptanceCriteriaPlaceholder')}
                    rows={5}
                  />
                  <Text fontSize="sm" color="gray.500" mt={1}>
                    {t('pms.acceptanceCriteriaHelp')}
                  </Text>
                </FormControl>

                <Grid templateColumns="1fr 1fr 1fr" gap={4} w="100%">
                  <FormControl>
                    <FormLabel>{t('pms.userStoryStatus')}</FormLabel>
                    <Select
                      value={formData.status}
                      onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                    >
                      <option value="open">{t('pms.statusOpen')}</option>
                      <option value="in_progress">{t('pms.statusInProgress')}</option>
                      <option value="completed">{t('pms.statusCompleted')}</option>
                    </Select>
                  </FormControl>

                  <FormControl>
                    <FormLabel>{t('pms.userStoryPriority')}</FormLabel>
                    <Select
                      value={formData.priority}
                      onChange={(e) => setFormData({ ...formData, priority: e.target.value })}
                    >
                      <option value="urgent">{t('pms.priorityUrgent')}</option>
                      <option value="high">{t('pms.priorityHigh')}</option>
                      <option value="medium">{t('pms.priorityMedium')}</option>
                      <option value="low">{t('pms.priorityLow')}</option>
                    </Select>
                  </FormControl>

                  <FormControl>
                    <FormLabel>{t('pms.storyPoints')}</FormLabel>
                    <NumberInput
                      value={formData.storyPoints}
                      onChange={(valueString) => setFormData({ ...formData, storyPoints: parseInt(valueString) || 0 })}
                      min={0}
                    >
                      <NumberInputField />
                      <NumberInputStepper>
                        <NumberIncrementStepper />
                        <NumberDecrementStepper />
                      </NumberInputStepper>
                    </NumberInput>
                  </FormControl>
                </Grid>

                <FormControl>
                  <FormLabel>{t('pms.assignee')}</FormLabel>
                  <UserSelect
                    value={formData.assigneeName || null}
                    onChange={(u) => setFormData({ ...formData, assigneeName: u?.userName || '' })}
                    placeholder={t('pms.assigneePlaceholder')}
                  />
                </FormControl>

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
                    {t('pms.createUserStory')}
                  </Button>
                </HStack>
              </VStack>
            </form>
          </CardBody>
        </Card>
      </VStack>

      {/* AI 优化 Modal：基于当前草稿优化标题/描述/验收标准 + 建议优先级/故事点 */}
      <AIOptimizeModal
        isOpen={isAIOptimizeOpen}
        onClose={() => setIsAIOptimizeOpen(false)}
        projectSlug={projectSlug}
        draft={{
          title: formData.title,
          description: formData.description,
          acceptanceCriteria: formData.acceptanceCriteria,
        }}
        onApply={handleApplyOptimize}
      />
    </Container>
  )
}
