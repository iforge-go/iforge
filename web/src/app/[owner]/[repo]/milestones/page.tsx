'use client'

import {
  Box,
  Button,
  Container,
  Heading,
  HStack,
  Icon,
  Text,
  VStack,
  Badge,
} from '@chakra-ui/react'
import { FiPlus, FiCheckCircle, FiCircle, FiFlag } from 'react-icons/fi'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useEffect, useState } from 'react'
import { api, Milestone } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'

export default function MilestonesPage() {
  const params = useParams()
  const owner = params.owner as string
  const repo = params.repo as string
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'
  const { userRole } = useRepo()

  const [milestones, setMilestones] = useState<Milestone[]>([])
  const [loading, setLoading] = useState(true)

  // Developer 及以上权限可以创建里程碑
  const canCreateMilestone = userRole === 'owner' || userRole === 'member'

  const loadMilestones = async () => {
    try {
      setLoading(true)
      const data = await api.listMilestones(owner, repo)
      setMilestones(data || [])
    } catch (error) {
      console.error('Failed to load milestones:', error)
      setMilestones([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadMilestones()
  }, [owner, repo])

  const openMilestones = milestones.filter(m => !m.closedDate)
  const closedMilestones = milestones.filter(m => m.closedDate)

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
        {/* Issues / Milestones sub-tabs */}
        <HStack spacing={1} borderBottom="1px solid" borderColor="myGray.200">
          <Link href={`/${owner}/${repo}/issues`}>
            <Button
              variant="ghost"
              size="sm"
              fontWeight="medium"
              color="myGray.500"
              mb="-1px"
              borderRadius="0"
              _hover={{ color: 'myGray.900' }}
            >
              <Icon as={FiCheckCircle} mr={1} />
              {t('repo.issues')}
            </Button>
          </Link>
          <Link href={`/${owner}/${repo}/milestones`}>
            <Button
              variant="ghost"
              size="sm"
              fontWeight="semibold"
              borderBottom="2px solid"
              borderColor="primary.600"
              mb="-1px"
              borderRadius="0"
            >
              <Icon as={FiFlag} mr={1} />
              {t('repo.milestones')}
              {milestones.length > 0 && (
                <Badge ml={1} colorScheme="gray" fontSize="xs">{milestones.length}</Badge>
              )}
            </Button>
          </Link>
        </HStack>

        <HStack justify="space-between">
          <Heading size="md">{t('repo.milestones')}</Heading>
          {canCreateMilestone && (
            <Link href={`/${owner}/${repo}/milestones/new`}>
              <Button leftIcon={<Icon as={FiPlus} />} colorScheme="blue" size="sm">
                {t('repo.newMilestone')}
              </Button>
            </Link>
          )}
        </HStack>

        {openMilestones.length > 0 && (
          <VStack spacing={4} align="stretch">
            <HStack>
              <Icon as={FiCircle} color="green.500" />
              <Heading size="sm">{openMilestones.length} {t('repo.openMilestonesCount')}</Heading>
            </HStack>
            {openMilestones.map((milestone) => (
              <MilestoneCard key={milestone.milestoneId} milestone={milestone} owner={owner} repo={repo} t={t} dateLocale={dateLocale} />
            ))}
          </VStack>
        )}

        {closedMilestones.length > 0 && (
          <VStack spacing={4} align="stretch" mt={6}>
            <HStack>
              <Icon as={FiCheckCircle} color="purple.500" />
              <Heading size="sm">{closedMilestones.length} {t('repo.closedMilestonesCount')}</Heading>
            </HStack>
            {closedMilestones.map((milestone) => (
              <MilestoneCard key={milestone.milestoneId} milestone={milestone} owner={owner} repo={repo} t={t} dateLocale={dateLocale} />
            ))}
          </VStack>
        )}

        {milestones.length === 0 && (
          <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
            <Box py={12}>
              <VStack spacing={4}>
                <Text color="gray.500">{t('repo.noMilestones')}</Text>
                {canCreateMilestone && (
                  <Link href={`/${owner}/${repo}/milestones/new`}>
                    <Button colorScheme="blue" variant="outline" size="sm">{t('repo.createFirstMilestone')}</Button>
                  </Link>
                )}
              </VStack>
            </Box>
          </Box>
        )}
      </VStack>
    </Container>
  )
}

function MilestoneCard({ milestone, owner, repo, t, dateLocale }: { milestone: Milestone; owner: string; repo: string; t: (key: string, params?: Record<string, string | number>) => string; dateLocale: string }) {
  const isClosed = !!milestone.closedDate

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
      <Box p={4}>
        <VStack align="stretch" spacing={3}>
          <HStack justify="space-between">
            <Link href={`/${owner}/${repo}/milestones/${milestone.milestoneId}`}>
              <Heading size="sm" _hover={{ color: 'blue.500' }} cursor="pointer">
                {milestone.title}
              </Heading>
            </Link>
            {isClosed && <Badge colorScheme="purple">{t('repo.closed')}</Badge>}
          </HStack>

          {milestone.description && (
            <Text fontSize="sm" color="gray.600" noOfLines={2}>{milestone.description}</Text>
          )}

          {milestone.dueDate && (
            <Text fontSize="xs" color="gray.500">{t('repo.dueDate')}: {new Date(milestone.dueDate).toLocaleDateString(dateLocale)}</Text>
          )}
        </VStack>
      </Box>
    </Box>
  )
}
