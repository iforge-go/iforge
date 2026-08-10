'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  VStack,
  HStack,
  Icon,
  Spinner,
  Button,
} from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { api } from '@/lib/api'
import { Activity } from '@/lib/types'
import { ActivityList } from '@/components/ActivityList'
import { FiTrendingUp } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

const PAGE_SIZE = 50

export default function RepoActivityPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const { t } = useI18n()

  const [activities, setActivities] = useState<Activity[]>([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(false)

  const loadActivities = async (offset: number, append: boolean) => {
    if (append) {
      setLoadingMore(true)
    }
    try {
      const data = await api.listRepoActivities(owner, repoName, PAGE_SIZE, offset)
      const newActivities = data.activities || []
      setActivities(prev => append ? [...prev, ...newActivities] : newActivities)
      setHasMore(newActivities.length === PAGE_SIZE)
    } catch (err) {
      console.error('Failed to load activities:', err)
    } finally {
      setLoading(false)
      setLoadingMore(false)
    }
  }

  useEffect(() => {
    loadActivities(0, false)
  }, [owner, repoName])

  const handleLoadMore = () => {
    loadActivities(activities.length, true)
  }

  return (
    <Container maxW="container.xl" py={6}>
      <VStack spacing={5} align="stretch">
        <VStack align="start" spacing={1}>
          <Heading size="lg" color="myGray.900">{t('repo.activityTitle')}</Heading>
          <Text fontSize="sm" color="myGray.600">{t('repo.activityDesc')}</Text>
        </VStack>

        <Box borderWidth="1px" borderColor="myGray.200" borderRadius="lg" overflow="hidden" bg="white">
          {/* 头部 */}
          <Box px={5} py={3} borderBottom="1px" borderColor="myGray.200">
            <HStack spacing={3}>
              <Icon as={FiTrendingUp} w={5} h={5} color="myGray.600" />
              <Heading size="sm" color="myGray.900">{t('home.activity')}</Heading>
            </HStack>
          </Box>

          {loading ? (
            <HStack justify="center" py={12}>
              <Spinner size="sm" />
              <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
            </HStack>
          ) : (
            <ActivityList activities={activities} showRepoLink={false} />
          )}
        </Box>

        {hasMore && !loading && (
          <HStack justify="center">
            <Button
              variant="whiteBase"
              size="sm"
              borderWidth="1px"
              borderColor="myGray.200"
              onClick={handleLoadMore}
              isLoading={loadingMore}
            >
              {t('repo.activityLoadMore')}
            </Button>
          </HStack>
        )}
      </VStack>
    </Container>
  )
}
