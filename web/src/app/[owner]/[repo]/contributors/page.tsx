'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  VStack,
  HStack,
  Avatar,
  SimpleGrid,
  Card,
  CardBody,
  Badge,
  Stat,
  StatLabel,
  StatNumber,
  StatGroup,
  Icon,
} from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { api, Contributor } from '@/lib/api'
import { FiUsers, FiGitCommit, FiPlus, FiMinus } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

export default function ContributorsPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'

  const [contributors, setContributors] = useState<Contributor[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.getContributors(owner, repoName)
      .then((data) => setContributors(data || []))
      .catch((error) => {
        console.error('Failed to load contributors:', error)
      })
      .finally(() => setLoading(false))
  }, [owner, repoName])

  if (loading) {
    return (
      <Container maxW="container.xl" py={8}>
        <Text>{t('common.loading')}</Text>
      </Container>
    )
  }

  const totalCommits = contributors.reduce((sum, c) => sum + (c.commits || 0), 0)
  const totalAdditions = contributors.reduce((sum, c) => sum + (c.additions || 0), 0)
  const totalDeletions = contributors.reduce((sum, c) => sum + (c.deletions || 0), 0)

  return (
    <Container maxW="container.xl" py={8}>
      <VStack spacing={6} align="stretch">
        <Heading size="lg" color="myGray.900">
          {t('repo.contributors')}
        </Heading>

        {/* 统计概览 */}
        <StatGroup borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
          <Stat>
            <StatLabel>
              <HStack spacing={2}>
                <Icon as={FiUsers} />
                <Text>{t('contributor.count')}</Text>
              </HStack>
            </StatLabel>
            <StatNumber>{contributors.length}</StatNumber>
          </Stat>
          <Stat>
            <StatLabel>
              <HStack spacing={2}>
                <Icon as={FiGitCommit} />
                <Text>{t('contributor.totalCommits')}</Text>
              </HStack>
            </StatLabel>
            <StatNumber>{totalCommits}</StatNumber>
          </Stat>
          <Stat>
            <StatLabel>
              <HStack spacing={2}>
                <Icon as={FiPlus} color="green.500" />
                <Text>{t('contributor.additions')}</Text>
              </HStack>
            </StatLabel>
            <StatNumber color="green.500">+{totalAdditions.toLocaleString()}</StatNumber>
          </Stat>
          <Stat>
            <StatLabel>
              <HStack spacing={2}>
                <Icon as={FiMinus} color="red.500" />
                <Text>{t('contributor.deletions')}</Text>
              </HStack>
            </StatLabel>
            <StatNumber color="red.500">-{totalDeletions.toLocaleString()}</StatNumber>
          </Stat>
        </StatGroup>

        {/* 贡献者列表 */}
        {contributors.length === 0 ? (
          <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
            <Box py={12}>
              <VStack spacing={4}>
                <Icon as={FiUsers} w={12} h={12} color="myGray.400" />
                <Text color="myGray.600">{t('contributor.noContributors')}</Text>
              </VStack>
            </Box>
          </Box>
        ) : (
          <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} spacing={4}>
            {contributors.map((contributor) => (
              <Card
                key={contributor.name}
                borderWidth="1px"
                borderColor="myGray.200"
                _hover={{ shadow: 'md' }}
              >
                <CardBody>
                  <VStack align="start" spacing={3}>
                    <HStack spacing={3} w="full">
                      <Avatar
                        size="md"
                        name={contributor.name}
                        src={contributor.avatarUrl || undefined}
                      />
                      <VStack align="start" spacing={0} flex={1} minW={0}>
                        <Text fontWeight="medium" color="myGray.900" noOfLines={1}>
                          {contributor.name}
                        </Text>
                        {contributor.email && (
                          <Text fontSize="xs" color="myGray.500" noOfLines={1}>
                            {contributor.email}
                          </Text>
                        )}
                      </VStack>
                    </HStack>

                    <HStack spacing={2} flexWrap="wrap">
                      <Badge colorScheme="blue" fontSize="xs">
                        {contributor.commits} {t('contributor.commits')}
                      </Badge>
                      {contributor.additions > 0 && (
                        <Badge colorScheme="green" fontSize="xs">
                          +{contributor.additions.toLocaleString()}
                        </Badge>
                      )}
                      {contributor.deletions > 0 && (
                        <Badge colorScheme="red" fontSize="xs">
                          -{contributor.deletions.toLocaleString()}
                        </Badge>
                      )}
                    </HStack>

                    {contributor.lastCommitAt && (
                      <Text fontSize="xs" color="myGray.500">
                        {t('contributor.lastCommit')} {new Date(contributor.lastCommitAt).toLocaleDateString(dateLocale)}
                      </Text>
                    )}
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
