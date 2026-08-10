'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  VStack,
  HStack,
  Input,
  InputGroup,
  InputLeftElement,
  InputRightElement,
  IconButton,
  Button,
  Icon,
  Spinner,
  Select,
  Code,
  Badge,
  Collapse,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
} from '@chakra-ui/react'
import { Suspense, useEffect, useMemo, useState } from 'react'
import { useParams, useRouter, useSearchParams } from 'next/navigation'
import Link from 'next/link'
import { api, CodeSearchResult } from '@/lib/api'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useI18n } from '@/contexts/I18nContext'
import { FiSearch, FiFile, FiChevronRight, FiChevronDown, FiAlertTriangle } from 'react-icons/fi'

// 解析 "L<行号>: <行内容>" 格式
function parseMatch(match: string): { line: number; content: string } | null {
  const m = /^L(\d+):\s?(.*)$/.exec(match)
  if (!m) return null
  return { line: parseInt(m[1], 10), content: m[2] }
}

// 高亮匹配子串（大小写不敏感）
function highlight(content: string, query: string) {
  if (!query) return content
  const lower = content.toLowerCase()
  const q = query.toLowerCase()
  const parts: Array<{ text: string; hit: boolean }> = []
  let i = 0
  while (i < content.length) {
    const idx = lower.indexOf(q, i)
    if (idx === -1) {
      parts.push({ text: content.slice(i), hit: false })
      break
    }
    if (idx > i) parts.push({ text: content.slice(i, idx), hit: false })
    parts.push({ text: content.slice(idx, idx + q.length), hit: true })
    i = idx + q.length
  }
  return parts.map((p, idx) =>
    p.hit ? (
      <Box as="mark" key={idx} bg="yellow.200" color="myGray.900" px="2px" borderRadius="sm">
        {p.text}
      </Box>
    ) : (
      <Box as="span" key={idx}>{p.text}</Box>
    ),
  )
}

function RepoSearchContent() {
  const params = useParams()
  const router = useRouter()
  const searchParams = useSearchParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const { t } = useI18n()
  const { branches } = useRepo()

  const initialQuery = searchParams.get('q') || ''
  const initialRef = searchParams.get('ref') || ''

  const [query, setQuery] = useState(initialQuery)
  const [selectedRef, setSelectedRef] = useState(initialRef)
  const [submittedQuery, setSubmittedQuery] = useState(initialQuery)
  const [results, setResults] = useState<CodeSearchResult[]>([])
  const [truncated, setTruncated] = useState(false)
  const [loading, setLoading] = useState(false)
  const [searched, setSearched] = useState(false)
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set())

  const effectiveRef = selectedRef || 'HEAD'

  const totalMatches = useMemo(
    () => results.reduce((sum, r) => sum + r.matches.length, 0),
    [results],
  )

  const runSearch = async (q: string, ref: string) => {
    if (!q.trim()) return
    setLoading(true)
    setSearched(true)
    try {
      const data = await api.searchCode(owner, repoName, q.trim(), ref || undefined)
      setResults(data?.results || [])
      setTruncated(data?.truncated || false)
      // 默认展开前 5 个文件
      setExpandedPaths(new Set((data?.results || []).slice(0, 5).map((r) => r.path)))
    } catch (err) {
      console.error('Failed to search code:', err)
      setResults([])
      setTruncated(false)
    } finally {
      setLoading(false)
    }
  }

  // 初始加载：如果 URL 有 q 参数，自动执行搜索
  useEffect(() => {
    if (initialQuery) {
      runSearch(initialQuery, initialRef)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setSubmittedQuery(query.trim())
    // 同步 URL，便于分享
    const qs = new URLSearchParams()
    if (query.trim()) qs.set('q', query.trim())
    if (selectedRef) qs.set('ref', selectedRef)
    router.replace(`/${owner}/${repoName}/search${qs.toString() ? '?' + qs.toString() : ''}`)
    runSearch(query, selectedRef)
  }

  const togglePath = (path: string) => {
    setExpandedPaths((prev) => {
      const next = new Set(prev)
      if (next.has(path)) {
        next.delete(path)
      } else {
        next.add(path)
      }
      return next
    })
  }

  return (
    <Container maxW="container.xl" py={6}>
      <VStack spacing={5} align="stretch">
        <VStack align="start" spacing={1}>
          <Heading size="lg" color="myGray.900">{t('repo.codeSearchTitle')}</Heading>
          <Text fontSize="sm" color="myGray.600">{t('repo.codeSearchDesc')}</Text>
        </VStack>

        {/* 搜索表单 */}
        <Box as="form" onSubmit={handleSubmit}>
          <HStack spacing={2} flexWrap="wrap">
            <InputGroup size="md" maxW="600px" flex="1">
              <InputLeftElement pointerEvents="none">
                <Icon as={FiSearch} color="myGray.400" />
              </InputLeftElement>
              <Input
                placeholder={t('repo.codeSearchPlaceholder')}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                bg="white"
                borderWidth="1px"
                borderColor="myGray.200"
                borderRadius="md"
                pl={10}
                pr={3}
              />
              {query && (
                <InputRightElement>
                  <IconButton
                    aria-label="clear"
                    icon={<Text color="myGray.400">×</Text>}
                    size="xs"
                    variant="ghost"
                    onClick={() => setQuery('')}
                  />
                </InputRightElement>
              )}
            </InputGroup>
            <Select
              size="md"
              w="180px"
              flexShrink={0}
              placeholder={t('repo.codeSearchAllBranches')}
              value={selectedRef}
              onChange={(e) => setSelectedRef(e.target.value)}
              bg="white"
              borderWidth="1px"
              borderColor="myGray.200"
              borderRadius="md"
            >
              {branches.map((b) => (
                <option key={b.name} value={b.name}>{b.name}</option>
              ))}
            </Select>
            <Button
              type="submit"
              variant="primary"
              leftIcon={<Icon as={FiSearch} />}
              isLoading={loading}
              flexShrink={0}
            >
              {t('common.search')}
            </Button>
          </HStack>
        </Box>

        {/* 搜索状态 */}
        {loading && (
          <HStack spacing={3} py={6} justify="center">
            <Spinner size="sm" />
            <Text fontSize="sm" color="myGray.500">{t('repo.codeSearchSearching')}</Text>
          </HStack>
        )}

        {!loading && searched && (
          <>
            {/* 结果统计 */}
            <HStack spacing={3} fontSize="sm" color="myGray.600">
              <Text>
                {totalMatches > 0
                  ? t('repo.codeSearchStats', { files: results.length, matches: totalMatches, query: submittedQuery })
                  : t('repo.codeSearchNoResults', { query: submittedQuery })}
              </Text>
            </HStack>

            {/* 截断提示 */}
            {truncated && (
              <Alert status="warning" borderRadius="md" py={2}>
                <AlertIcon as={FiAlertTriangle} />
                <AlertTitle fontSize="sm">{t('repo.codeSearchTruncatedTitle')}</AlertTitle>
                <AlertDescription fontSize="sm">
                  {t('repo.codeSearchTruncatedDesc')}
                </AlertDescription>
              </Alert>
            )}

            {/* 结果列表 */}
            {results.length > 0 && (
              <VStack spacing={2} align="stretch">
                {results.map((result) => {
                  const expanded = expandedPaths.has(result.path)
                  const firstLine = parseMatch(result.matches[0] || '')
                  return (
                    <Box
                      key={result.path}
                      borderWidth="1px"
                      borderRadius="md"
                      borderColor="myGray.200"
                      bg="white"
                      overflow="hidden"
                    >
                      {/* 文件头 */}
                      <HStack
                        spacing={2}
                        px={4}
                        py={2}
                        bg="myGray.50"
                        borderBottom={expanded ? '1px solid' : 'none'}
                        borderColor="myGray.200"
                        cursor="pointer"
                        _hover={{ bg: 'myGray.100' }}
                        onClick={() => togglePath(result.path)}
                      >
                        <Icon
                          as={expanded ? FiChevronDown : FiChevronRight}
                          w={4}
                          h={4}
                          color="myGray.500"
                          flexShrink={0}
                        />
                        <Icon as={FiFile} w={4} h={4} color="myGray.500" flexShrink={0} />
                        <Link
                          href={`/${owner}/${repoName}/blob/${encodeURIComponent(effectiveRef)}/${result.path}`}
                          onClick={(e) => e.stopPropagation()}
                        >
                          <Text
                            fontSize="sm"
                            fontFamily="mono"
                            color="primary.600"
                            _hover={{ textDecoration: 'underline' }}
                          >
                            {result.path}
                          </Text>
                        </Link>
                        <Badge colorScheme="gray" fontSize="xs">
                          {result.matches.length}
                        </Badge>
                        {!expanded && firstLine && (
                          <Text fontSize="xs" color="myGray.500" noOfLines={1} flex={1} ml={2}>
                            <Code fontSize="xs" color="myGray.400">L{firstLine.line}</Code>
                            {' '}
                            {firstLine.content}
                          </Text>
                        )}
                      </HStack>

                      {/* 匹配行展开 */}
                      <Collapse in={expanded}>
                        <VStack spacing={0} align="stretch">
                          {result.matches.map((m, idx) => {
                            const parsed = parseMatch(m)
                            if (!parsed) return null
                            const lineUrl = `/${owner}/${repoName}/blob/${encodeURIComponent(effectiveRef)}/${result.path}#L${parsed.line}`
                            return (
                              <Link key={idx} href={lineUrl}>
                                <HStack
                                  spacing={3}
                                  px={4}
                                  py={1.5}
                                  align="flex-start"
                                  borderBottom={idx < result.matches.length - 1 ? '1px solid' : 'none'}
                                  borderColor="myGray.100"
                                  _hover={{ bg: 'myGray.50' }}
                                  fontSize="sm"
                                  fontFamily="mono"
                                >
                                  <Text color="myGray.400" minW="50px" flexShrink={0}>
                                    L{parsed.line}
                                  </Text>
                                  <Text whiteSpace="pre-wrap" wordBreak="break-word" flex={1}>
                                    {highlight(parsed.content, submittedQuery)}
                                  </Text>
                                </HStack>
                              </Link>
                            )
                          })}
                        </VStack>
                      </Collapse>
                    </Box>
                  )
                })}
              </VStack>
            )}

            {/* 空结果 */}
            {results.length === 0 && (
              <Box borderWidth="1px" borderRadius="lg" bg="white" borderColor="myGray.200" py={12}>
                <VStack spacing={3}>
                  <Icon as={FiSearch} w={10} h={10} color="myGray.400" />
                  <Text color="myGray.600" fontSize="sm">
                    {t('repo.codeSearchNoResults', { query: submittedQuery })}
                  </Text>
                </VStack>
              </Box>
            )}
          </>
        )}

        {/* 初始状态（未搜索） */}
        {!searched && !loading && (
          <Box borderWidth="1px" borderRadius="lg" bg="white" borderColor="myGray.200" py={12}>
            <VStack spacing={3}>
              <Icon as={FiSearch} w={10} h={10} color="myGray.400" />
              <Text color="myGray.600" fontSize="sm" textAlign="center" maxW="500px">
                {t('repo.codeSearchEmptyHint')}
              </Text>
            </VStack>
          </Box>
        )}
      </VStack>
    </Container>
  )
}

export default function RepoSearchPage() {
  return (
    <Suspense fallback={null}>
      <RepoSearchContent />
    </Suspense>
  )
}
