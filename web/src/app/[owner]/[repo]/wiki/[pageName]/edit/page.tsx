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
} from '@chakra-ui/react'
import { useGithubToast } from '@/app/providers'
import { useEffect, useState, useCallback } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { api, WikiPage } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { MarkdownEditor } from '@/components/MarkdownEditor'

export default function WikiEditPage() {
  const params = useParams()
  const router = useRouter()
  const toast = useGithubToast()
  const owner = params.owner as string
  const repoName = params.repo as string
  const pageName = params.pageName as string
  const { t } = useI18n()
  const [page, setPage] = useState<WikiPage | null>(null)
  const [loading, setLoading] = useState(true)
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const loadPage = useCallback(async () => {
    try {
      const data = await api.getWikiPage(owner, repoName, pageName)
      setPage(data)
      setTitle(data.title)
      setContent(data.content)
    } catch (error) {
      console.error('Failed to load wiki page:', error)
      toast({
        title: t('wiki.loadFailed'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }, [owner, repoName, pageName, toast, t])

  useEffect(() => {
    loadPage()
  }, [loadPage])

  const handleSubmit = async () => {
    if (!title.trim()) {
      toast({
        title: t('wiki.titleRequired'),
        status: 'warning',
        duration: 2000,
      })
      return
    }

    setSubmitting(true)
    try {
      await api.updateWikiPage(owner, repoName, pageName, { title, content })
      toast({
        title: t('wiki.updated'),
        status: 'success',
        duration: 2000,
      })
      router.push(`/${owner}/${repoName}/wiki/${pageName}`)
    } catch {
      toast({
        title: t('wiki.updateFailed'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return (
      <Container maxW="container.xl" py={8}>
        <Text>{t('common.loading')}</Text>
      </Container>
    )
  }

  if (!page) {
    return (
      <Container maxW="container.xl" py={8}>
        <Text>{t('wiki.notFound')}</Text>
      </Container>
    )
  }

  return (
    <Container maxW="container.xl" py={8}>
      <VStack spacing={6} align="stretch">
        <Heading size="lg" color="myGray.900">
          {t('wiki.editPage')}
        </Heading>

        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
          <Box p={6}>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('wiki.title')}</FormLabel>
                <Input
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder={t('wiki.titlePlaceholder')}
                />
              </FormControl>

              <FormControl isRequired>
                <FormLabel>{t('wiki.content')}</FormLabel>
                <MarkdownEditor
                  value={content}
                  onChange={setContent}
                  placeholder={t('wiki.contentPlaceholder')}
                />
              </FormControl>

              <HStack spacing={2} justify="flex-end" w="full">
                <Button
                  variant="ghost"
                  onClick={() => router.push(`/${owner}/${repoName}/wiki/${pageName}`)}
                >
                  {t('common.cancel')}
                </Button>
                <Button
                  colorScheme="primary"
                  onClick={handleSubmit}
                  isLoading={submitting}
                >
                  {t('common.save')}
                </Button>
              </HStack>
            </VStack>
          </Box>
        </Box>
      </VStack>
    </Container>
  )
}
