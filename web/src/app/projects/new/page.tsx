'use client'

import {
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
  Switch,
  useToast,
  Card,
  CardBody,
} from '@chakra-ui/react'
import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'

export default function NewProjectPage() {
  const router = useRouter()
  const { t } = useI18n()
  const toast = useToast()
  const { user, authLoading } = useCurrentUser()

  const [formData, setFormData] = useState({
    name: '',
    description: '',
    isPrivate: false,
  })
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!formData.name) {
      toast({
        title: t('common.error'),
        description: t('pms.nameAndSlugRequired'),
        status: 'error',
        duration: 3000,
      })
      return
    }

    setIsSubmitting(true)
    try {
      const project = await api.createProject({
        name: formData.name,
        description: formData.description || undefined,
        isPrivate: formData.isPrivate,
      })
      
      toast({
        title: t('pms.projectCreated'),
        description: t('pms.projectCreatedDesc'),
        status: 'success',
        duration: 3000,
      })
      
      router.push(`/projects/${project.slug}`)
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
          <Heading size="lg">{t('pms.newProject')}</Heading>
          <Text color="gray.600">{t('pms.newProjectDesc')}</Text>
        </VStack>

        <Card>
          <CardBody>
            <form onSubmit={handleSubmit}>
              <VStack spacing={6}>
                <FormControl isRequired>
                  <FormLabel>{t('pms.projectName')}</FormLabel>
                  <Input
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder={t('pms.projectNamePlaceholder')}
                  />
                </FormControl>

                <FormControl>
                  <FormLabel>{t('pms.projectDescription')}</FormLabel>
                  <Textarea
                    value={formData.description}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    placeholder={t('pms.projectDescriptionPlaceholder')}
                    rows={4}
                  />
                </FormControl>

                <FormControl display="flex" alignItems="center">
                  <FormLabel htmlFor="is-private" mb="0" mr={3}>
                    {t('pms.privateProject')}
                  </FormLabel>
                  <Switch
                    id="is-private"
                    isChecked={formData.isPrivate}
                    onChange={(e) => setFormData({ ...formData, isPrivate: e.target.checked })}
                  />
                  <Text fontSize="sm" color="gray.500" ml={3}>
                    {t('pms.privateProjectHelp')}
                  </Text>
                </FormControl>

                <HStack spacing={4} w="100%">
                  <Button
                    variant="outline"
                    onClick={() => router.push('/pms')}
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
                    {t('pms.createProject')}
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
