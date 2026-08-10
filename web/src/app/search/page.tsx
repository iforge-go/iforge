'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  Input,
  InputGroup,
  InputLeftElement,
  VStack,
  HStack,
  Tabs,
  TabList,
  Tab,
  TabPanels,
  TabPanel,
  Card,
  CardBody,
  Badge,
  Code,
  Link as ChakraLink,
  Icon,
  SimpleGrid,
  Divider,
} from '@chakra-ui/react'
import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { api, Issue } from '@/lib/api'
import { FiSearch, FiGitCommit, FiFileText, FiAlertCircle } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

interface CodeSearchResult {
  path: string
  fileName: string
  ref: string
  matches: Array<{
    lineNumber: number
    line: string
    before?: string
    after?: string
  }>
}

export default function SearchPage() {
  const router = useRouter()
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'

  const [query, setQuery] = useState('')
  const [issues, setIssues] = useState<Issue[]>([])
  const [codeResults, setCodeResults] = useState<CodeSearchResult[]>([])
  const [loading, setLoading] = useState(false)
  const [searched, setSearched] = useState(false)

  const handleSearch = async () => {
    if (!query.trim()) return

    setLoading(true)
    setSearched(true)
    try {
      const results = await Promise.allSettled([
        api.searchAllIssues(query)
      ])
      const issueResult = results[0]
      if (issueResult.status === 'fulfilled') {
        setIssues(issueResult.value.issues || [])
      } else {
        setIssues([])
      }
    } catch (error) {
      console.error('Search failed:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch()
    }
  }

  return (
    <Container maxW="container.xl" py={8}>
      <VStack spacing={6} align="stretch">
        <Heading size="lg" color="myGray.900">
          {t('search.title')}
        </Heading>

        <InputGroup size="lg">
          <InputLeftElement pointerEvents="none">
            <Icon as={FiSearch} color="myGray.400" />
          </InputLeftElement>
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t('search.placeholder')}
            bg="white"
            borderWidth="1px"
            borderColor="myGray.200"
          />
        </InputGroup>

        <Tabs variant="line" colorScheme="primary">
          <TabList>
            <Tab>
              <HStack spacing={2}>
                <Icon as={FiAlertCircle} />
                <Text>{t('search.issues')}</Text>
                {issues.length > 0 && (
                  <Badge colorScheme="blue" fontSize="xs">
                    {issues.length}
                  </Badge>
                )}
              </HStack>
            </Tab>
          </TabList>

          <TabPanels>
            {/* Issue 搜索结果 */}
            <TabPanel px={0} pt={4}>
              {loading ? (
                <Text color="myGray.500">{t('common.loading')}</Text>
              ) : searched && issues.length === 0 ? (
                <Box py={8} textAlign="center">
                  <Text color="myGray.500">{t('search.noResults')}</Text>
                </Box>
              ) : issues.length === 0 ? (
                <Box py={8} textAlign="center">
                  <Text color="myGray.500">{t('search.startSearching')}</Text>
                </Box>
              ) : (
                <VStack spacing={3} align="stretch">
                  {issues.map((issue) => (
                    <Card
                      key={`${issue.userName}/${issue.repositoryName}/${issue.issueId}`}
                      borderWidth="1px"
                      borderColor="myGray.200"
                      _hover={{ shadow: 'md', borderColor: 'primary.500' }}
                      cursor="pointer"
                      onClick={() => router.push(`/${issue.userName}/${issue.repositoryName}/issues/${issue.issueId}`)}
                    >
                      <CardBody py={4}>
                        <VStack align="start" spacing={2}>
                          <HStack spacing={2} flexWrap="wrap">
                            <Badge colorScheme={issue.closed ? 'red' : 'green'} fontSize="xs">
                              {issue.closed ? t('repo.closed') : t('repo.open')}
                            </Badge>
                            <Text fontWeight="medium" color="myGray.900">
                              {issue.title}
                            </Text>
                            <Text fontSize="sm" color="primary.500">
                              {issue.userName}/{issue.repositoryName}#{issue.issueId}
                            </Text>
                          </HStack>
                          <HStack spacing={3} fontSize="xs" color="myGray.500">
                            <Text>
                              {t('repo.openedBy', { id: issue.issueId, user: issue.openedUserName, date: new Date(issue.registeredDate).toLocaleDateString(dateLocale) })}
                            </Text>
                            <Text>
                              {new Date(issue.registeredDate).toLocaleDateString(dateLocale)}
                            </Text>
                            {issue.labels && issue.labels.length > 0 && (
                              <HStack spacing={1}>
                                {issue.labels.slice(0, 3).map((label) => (
                                  <Badge key={label.labelId} fontSize="xs" style={{ backgroundColor: label.color, color: '#fff' }}>
                                    {label.labelName}
                                  </Badge>
                                ))}
                              </HStack>
                            )}
                          </HStack>
                        </VStack>
                      </CardBody>
                    </Card>
                  ))}
                </VStack>
              )}
            </TabPanel>
          </TabPanels>
        </Tabs>
      </VStack>
    </Container>
  )
}
