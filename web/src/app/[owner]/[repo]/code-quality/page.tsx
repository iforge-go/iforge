'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  VStack,
  HStack,
  Button,
  Icon,
  Badge,
  Spinner,
  SimpleGrid,
  Progress,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  TableContainer,
} from '@chakra-ui/react'
import { useGithubToast } from '@/app/providers'
import { useParams } from 'next/navigation'
import { useState } from 'react'
import { api, CodeQualityReport } from '@/lib/api'
import { FiActivity, FiCheckCircle, FiAlertTriangle } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

const MAINTAINABILITY_COLORS: Record<string, string> = {
  A: 'green',
  B: 'green',
  C: 'yellow',
  D: 'orange',
  F: 'red',
}

export default function CodeQualityPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const toast = useGithubToast()
  const { t } = useI18n()

  const [report, setReport] = useState<CodeQualityReport | null>(null)
  const [loading, setLoading] = useState(false)

  const handleAnalyze = async () => {
    setLoading(true)
    try {
      const data = await api.analyzeCodeQuality(owner, repoName)
      setReport(data)
      toast({ title: t('repo.codeQuality'), status: 'success', duration: 2000 })
    } catch (err: any) {
      toast({
        title: t('repo.codeQualityAnalyzeFailed'),
        description: err.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }

  const formatPercent = (v: number) => `${(v * 100).toFixed(1)}%`
  const formatScore = (v: number) => v.toFixed(2)

  return (
    <Container maxW="container.xl" py={6}>
      <VStack spacing={6} align="stretch">
        <HStack justify="space-between">
          <VStack align="start" spacing={1}>
            <Heading size="lg" color="myGray.900">{t('repo.codeQualityTitle')}</Heading>
            <Text fontSize="sm" color="myGray.600">{t('repo.codeQualityDesc')}</Text>
          </VStack>
          <Button
            variant="primary"
            leftIcon={<Icon as={FiActivity} />}
            onClick={handleAnalyze}
            isLoading={loading}
          >
            {loading ? t('repo.analyzing') : t('repo.analyzeCodeQuality')}
          </Button>
        </HStack>

        {loading && !report && (
          <Box py={12} textAlign="center">
            <Spinner size="lg" />
            <Text mt={4} color="myGray.500">{t('repo.analyzing')}</Text>
          </Box>
        )}

        {report && (
          <VStack spacing={6} align="stretch">
            {/* Overview cards */}
            <SimpleGrid columns={{ base: 1, md: 3 }} spacing={4}>
              <Box borderWidth="1px" borderRadius="lg" p={5} bg="white" borderColor="myGray.200">
                <Text fontSize="sm" color="myGray.500">{t('repo.totalLines')}</Text>
                <Text fontSize="2xl" fontWeight="bold" color="myGray.900">{report.totalLines.toLocaleString()}</Text>
              </Box>
              <Box borderWidth="1px" borderRadius="lg" p={5} bg="white" borderColor="myGray.200">
                <Text fontSize="sm" color="myGray.500">{t('repo.totalFiles')}</Text>
                <Text fontSize="2xl" fontWeight="bold" color="myGray.900">{report.totalFiles}</Text>
              </Box>
              <Box borderWidth="1px" borderRadius="lg" p={5} bg="white" borderColor="myGray.200">
                <Text fontSize="sm" color="myGray.500">{t('repo.maintainability')}</Text>
                <HStack spacing={2} mt={1}>
                  <Badge
                    fontSize="lg"
                    colorScheme={MAINTAINABILITY_COLORS[report.maintainability] || 'gray'}
                    px={3}
                    py={1}
                  >
                    {report.maintainability}
                  </Badge>
                </HStack>
              </Box>
            </SimpleGrid>

            {/* Metrics */}
            <Box borderWidth="1px" borderRadius="lg" bg="white" borderColor="myGray.200" overflow="hidden">
              <Box px={5} py={3} borderBottom="1px" borderColor="myGray.200">
                <Heading size="sm">{t('repo.complexityScore')}</Heading>
              </Box>
              <SimpleGrid columns={{ base: 1, md: 2 }} spacing={5} p={5}>
                <VStack align="stretch" spacing={2}>
                  <HStack justify="space-between">
                    <Text fontSize="sm" color="myGray.600">{t('repo.avgLineLength')}</Text>
                    <Text fontSize="sm" fontWeight="medium">{formatScore(report.avgLineLength)}</Text>
                  </HStack>
                </VStack>
                <VStack align="stretch" spacing={2}>
                  <HStack justify="space-between">
                    <Text fontSize="sm" color="myGray.600">{t('repo.maxLineLength')}</Text>
                    <Text fontSize="sm" fontWeight="medium">{report.maxLineLength}</Text>
                  </HStack>
                </VStack>
                <VStack align="stretch" spacing={2}>
                  <HStack justify="space-between">
                    <Text fontSize="sm" color="myGray.600">{t('repo.commentRatio')}</Text>
                    <Text fontSize="sm" fontWeight="medium">{formatPercent(report.commentRatio)}</Text>
                  </HStack>
                  <Progress value={report.commentRatio * 100} size="sm" colorScheme="green" />
                </VStack>
                <VStack align="stretch" spacing={2}>
                  <HStack justify="space-between">
                    <Text fontSize="sm" color="myGray.600">{t('repo.emptyLineRatio')}</Text>
                    <Text fontSize="sm" fontWeight="medium">{formatPercent(report.emptyLineRatio)}</Text>
                  </HStack>
                  <Progress value={report.emptyLineRatio * 100} size="sm" colorScheme="blue" />
                </VStack>
                <VStack align="stretch" spacing={2}>
                  <HStack justify="space-between">
                    <Text fontSize="sm" color="myGray.600">{t('repo.complexityScore')}</Text>
                    <Text fontSize="sm" fontWeight="medium">{formatScore(report.complexityScore)}</Text>
                  </HStack>
                  <Progress
                    value={Math.min(report.complexityScore * 10, 100)}
                    size="sm"
                    colorScheme={report.complexityScore > 7 ? 'red' : report.complexityScore > 4 ? 'yellow' : 'green'}
                  />
                </VStack>
              </SimpleGrid>
            </Box>

            {/* Language stats */}
            {Object.keys(report.languageStats).length > 0 && (
              <Box borderWidth="1px" borderRadius="lg" bg="white" borderColor="myGray.200" overflow="hidden">
                <Box px={5} py={3} borderBottom="1px" borderColor="myGray.200">
                  <Heading size="sm">{t('repo.languageStats')}</Heading>
                </Box>
                <Box p={5}>
                  <VStack align="stretch" spacing={3}>
                    {Object.entries(report.languageStats)
                      .sort(([, a], [, b]) => b - a)
                      .map(([lang, count]) => {
                        const maxCount = Math.max(...Object.values(report.languageStats))
                        return (
                          <VStack key={lang} align="stretch" spacing={1}>
                            <HStack justify="space-between">
                              <Text fontSize="sm" fontWeight="medium" color="myGray.900">{lang}</Text>
                              <Text fontSize="sm" color="myGray.600">{count} {t('repo.totalFiles')}</Text>
                            </HStack>
                            <Progress value={(count / maxCount) * 100} size="sm" colorScheme="primary" />
                          </VStack>
                        )
                      })}
                  </VStack>
                </Box>
              </Box>
            )}

            {/* Long functions */}
            <Box borderWidth="1px" borderRadius="lg" bg="white" borderColor="myGray.200" overflow="hidden">
              <Box px={5} py={3} borderBottom="1px" borderColor="myGray.200">
                <HStack spacing={2}>
                  <Icon as={FiAlertTriangle} color="orange.500" />
                  <Heading size="sm">{t('repo.longFunctions')}</Heading>
                </HStack>
              </Box>
              {report.longFunctions && report.longFunctions.length > 0 ? (
                <TableContainer>
                  <Table size="sm">
                    <Thead>
                      <Tr>
                        <Th>{t('repo.fileName')}</Th>
                        <Th>Function</Th>
                        <Th isNumeric>Line</Th>
                        <Th isNumeric>Length</Th>
                      </Tr>
                    </Thead>
                    <Tbody>
                      {report.longFunctions.map((fn, i) => (
                        <Tr key={i}>
                          <Td fontFamily="mono" fontSize="sm">{fn.file}</Td>
                          <Td fontFamily="mono" fontSize="sm">{fn.name}</Td>
                          <Td isNumeric fontSize="sm">{fn.line}</Td>
                          <Td isNumeric fontSize="sm">
                            <Badge colorScheme={fn.length > 100 ? 'red' : 'yellow'}>{fn.length}</Badge>
                          </Td>
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                </TableContainer>
              ) : (
                <Box px={5} py={4}>
                  <HStack spacing={2}>
                    <Icon as={FiCheckCircle} color="green.500" />
                    <Text fontSize="sm" color="myGray.500">{t('repo.noLongFunctions')}</Text>
                  </HStack>
                </Box>
              )}
            </Box>

            {/* Large files */}
            <Box borderWidth="1px" borderRadius="lg" bg="white" borderColor="myGray.200" overflow="hidden">
              <Box px={5} py={3} borderBottom="1px" borderColor="myGray.200">
                <HStack spacing={2}>
                  <Icon as={FiAlertTriangle} color="orange.500" />
                  <Heading size="sm">{t('repo.largeFiles')}</Heading>
                </HStack>
              </Box>
              {report.largeFiles && report.largeFiles.length > 0 ? (
                <TableContainer>
                  <Table size="sm">
                    <Thead>
                      <Tr>
                        <Th>{t('repo.fileName')}</Th>
                        <Th isNumeric>Lines</Th>
                        <Th isNumeric>Size</Th>
                      </Tr>
                    </Thead>
                    <Tbody>
                      {report.largeFiles.map((f, i) => (
                        <Tr key={i}>
                          <Td fontFamily="mono" fontSize="sm">{f.path}</Td>
                          <Td isNumeric fontSize="sm">{f.lines}</Td>
                          <Td isNumeric fontSize="sm">{(f.size / 1024).toFixed(1)} KB</Td>
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                </TableContainer>
              ) : (
                <Box px={5} py={4}>
                  <HStack spacing={2}>
                    <Icon as={FiCheckCircle} color="green.500" />
                    <Text fontSize="sm" color="myGray.500">{t('repo.noLargeFiles')}</Text>
                  </HStack>
                </Box>
              )}
            </Box>
          </VStack>
        )}
      </VStack>
    </Container>
  )
}
