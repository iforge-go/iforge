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
  Link as ChakraLink,
  SimpleGrid,
  Card,
  CardBody,
  Badge,
} from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import type { WikiPage } from '@/lib/api'
import { FiBook, FiPlus, FiFileText } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

export default function WikiPage() {
  const params = useParams()
  const router = useRouter()
  const owner = params.owner as string
  const repoName = params.repo as string
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'

  const [pages, setPages] = useState<WikiPage[]>([])
  const [loading, setLoading] = useState(true)

  const loadPages = async () => {
    try {
      const data = await api.listWikiPages(owner, repoName)
      setPages(data || [])
    } catch (error) {
      console.error('Failed to load wiki pages:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadPages()
  }, [owner, repoName])

  if (loading) {
    return (
      <Container maxW="container.xl" py={8}>
        <Text>{t('common.loading')}</Text>
      </Container>
    )
  }

  return (
    <Container maxW="container.xl" py={8}>
      <VStack spacing={6} align="stretch">
        <HStack justify="space-between">
          <VStack align="start" spacing={1}>
            <Heading size="lg" color="myGray.900">
              {t('wiki.title')}
            </Heading>
            <Text fontSize="sm" color="myGray.600">
              {t('wiki.description')}
            </Text>
          </VStack>
          <Button leftIcon={<FiPlus />} colorScheme="primary" onClick={() => router.push(`/${owner}/${repoName}/wiki/new`)}>
            {t('wiki.newPage')}
          </Button>
        </HStack>

        {pages.length === 0 ? (
          <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
            <Box py={12}>
              <VStack spacing={4}>
                <Icon as={FiBook} w={12} h={12} color="myGray.400" />
                <Text color="myGray.600">{t('wiki.noPages')}</Text>
                <Button leftIcon={<FiPlus />} colorScheme="primary" onClick={() => router.push(`/${owner}/${repoName}/wiki/new`)}>
                  {t('wiki.createFirst')}
                </Button>
              </VStack>
            </Box>
          </Box>
        ) : (
          <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} spacing={4}>
            {pages.map((page) => (
              <Card
                key={page.pageName}
                cursor="pointer"
                onClick={() => router.push(`/${owner}/${repoName}/wiki/${page.pageName}`)}
                _hover={{ shadow: 'md', borderColor: 'primary.500' }}
                borderWidth="1px"
                borderColor="myGray.200"
              >
                <CardBody>
                  <VStack align="start" spacing={3}>
                    <HStack spacing={2}>
                      <Icon as={FiFileText} color="primary.500" />
                      <Heading size="sm" color="myGray.900">
                        {page.title}
                      </Heading>
                    </HStack>
                    <Text fontSize="sm" color="myGray.600" noOfLines={3}>
                      {page.content.substring(0, 150)}
                      {page.content.length > 150 ? '...' : ''}
                    </Text>
                    <HStack spacing={2} mt="auto">
                      <Badge colorScheme="gray" fontSize="xs">
                        {page.pageName}
                      </Badge>
                      <Text fontSize="xs" color="myGray.500">
                        {t('wiki.lastEdited')} {new Date(page.updatedAt).toLocaleDateString(dateLocale)}
                      </Text>
                    </HStack>
                  </VStack>
                </CardBody>
              </Card>
            ))}
          </SimpleGrid>
        )}
      </VStack>
    </Container>
  )
}
