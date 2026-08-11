'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  Button,
  VStack,
  HStack,
  Badge,
  Divider,
} from '@chakra-ui/react'
import { useGithubToast } from '@/app/providers'
import { useEffect, useState, useCallback } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { api, WikiPage } from '@/lib/api'
import { FiEdit, FiTrash2, FiArrowLeft } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { MarkdownRenderer } from '@/components/MarkdownRenderer'
import { ConfirmDialog } from '@/components/ConfirmDialog'

export default function WikiDetailPage() {
  const params = useParams()
  const router = useRouter()
  const toast = useGithubToast()
  const owner = params.owner as string
  const repoName = params.repo as string
  const pageName = params.pageName as string
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'

  const [page, setPage] = useState<WikiPage | null>(null)
  const [loading, setLoading] = useState(true)

  const [confirmOpen, setConfirmOpen] = useState(false)
  const [confirmAction, setConfirmAction] = useState<(() => void) | null>(null)

  const loadPage = useCallback(async () => {
    try {
      const data = await api.getWikiPage(owner, repoName, pageName)
      setPage(data)
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

  const handleDelete = () => {
    setConfirmAction(() => async () => {
      try {
        await api.deleteWikiPage(owner, repoName, pageName)
        toast({
          title: t('wiki.deleted'),
          status: 'success',
          duration: 2000,
        })
        router.push(`/${owner}/${repoName}/wiki`)
      } catch {
        toast({
          title: t('wiki.deleteFailed'),
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
        <HStack justify="space-between">
          <Button
            leftIcon={<FiArrowLeft />}
            variant="ghost"
            onClick={() => router.push(`/${owner}/${repoName}/wiki`)}
          >
            {t('wiki.backToList')}
          </Button>
          <HStack spacing={2}>
            <Button
              leftIcon={<FiEdit />}
              variant="outline"
              onClick={() => router.push(`/${owner}/${repoName}/wiki/${pageName}/edit`)}
            >
              {t('common.edit')}
            </Button>
            <Button
              leftIcon={<FiTrash2 />}
              colorScheme="red"
              variant="outline"
              onClick={handleDelete}
            >
              {t('common.delete')}
            </Button>
          </HStack>
        </HStack>

        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
          <Box p={6}>
            <VStack align="start" spacing={4}>
              <Heading size="lg" color="myGray.900">
                {page.title}
              </Heading>
              <HStack spacing={2}>
                <Badge colorScheme="gray">{page.pageName}</Badge>
                <Text fontSize="sm" color="myGray.600">
                  {t('wiki.lastEdited')} {new Date(page.updatedAt).toLocaleDateString(dateLocale)}
                </Text>
              </HStack>
              <Divider />
              <Box w="full">
                <MarkdownRenderer content={page.content} />
              </Box>
            </VStack>
          </Box>
        </Box>

        <ConfirmDialog
          isOpen={confirmOpen}
          onClose={() => setConfirmOpen(false)}
          onConfirm={handleConfirm}
          title={t('common.confirm')}
          message={t('wiki.confirmDelete')}
        />
      </VStack>
    </Container>
  )
}
