'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  Button,
  VStack,
  HStack,
  Input,
  InputGroup,
  InputLeftElement,
  InputRightElement,
  Badge,
  Icon,
  Flex,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
} from '@chakra-ui/react'
import Link from 'next/link'
import { Suspense, useEffect, useState } from 'react'
import { useSearchParams, useRouter } from 'next/navigation'
import { api, Repository } from '@/lib/api'
import { FiSearch, FiBook, FiStar, FiGitBranch, FiChevronDown, FiX } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

function ReposContent() {
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'
  const router = useRouter()
  const searchParams = useSearchParams()
  const [repos, setRepos] = useState<Repository[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [sortBy, setSortBy] = useState<'updated' | 'name' | 'stars'>('updated')

  // 从 URL ?q= 初始化搜索词（Header 跳转过来时自动填充）
  useEffect(() => {
    const q = searchParams.get('q')
    if (q) setSearchQuery(q)
  }, [searchParams])

  useEffect(() => {
    api.listRepos()
      .then(data => setRepos(data || []))
      .finally(() => setLoading(false))
  }, [])

  // 清除搜索：清空输入框并从 URL 移除 ?q=
  const handleClearSearch = () => {
    setSearchQuery('')
    if (searchParams.has('q')) {
      router.replace('/repos')
    }
  }

  const filteredRepos = repos
    .filter(repo => 
      repo.repositoryName.toLowerCase().includes(searchQuery.toLowerCase()) ||
      repo.description?.toLowerCase().includes(searchQuery.toLowerCase())
    )
    .sort((a, b) => {
      switch (sortBy) {
        case 'name':
          return a.repositoryName.localeCompare(b.repositoryName)
        case 'stars':
          return 0 // starsCount 字段已移除
        case 'updated':
        default:
          return new Date(b.updatedDate).getTime() - new Date(a.updatedDate).getTime()
      }
    })

  if (loading) {
    return (
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={8}>
          <Text>{t('common.loading')}</Text>
        </Container>
      </Box>
    )
  }

  return (
    <Box bg="myGray.50">
      <Container maxW="container.xl" py={8}>
        <VStack spacing={6} align="stretch">
          {/* 页面标题 */}
          <HStack justify="space-between">
            <Heading size="lg" color="myGray.900">
              {t('repos.title')}
            </Heading>
            <Link href="/new">
              <Button leftIcon={<Icon as={FiBook} />} variant="primary">
                {t('repos.newRepo')}
              </Button>
            </Link>
          </HStack>

          {/* 搜索和筛选 */}
          <HStack spacing={4}>
            <InputGroup flex={1}>
              <InputLeftElement>
                <Icon as={FiSearch} color="myGray.400" />
              </InputLeftElement>
              <Input
                placeholder={t('repos.searchPlaceholder')}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
              {searchQuery && (
                <InputRightElement cursor="pointer" onClick={handleClearSearch}>
                  <Icon as={FiX} color="myGray.400" _hover={{ color: 'myGray.700' }} />
                </InputRightElement>
              )}
            </InputGroup>
            <Menu>
              <MenuButton as={Button} rightIcon={<Icon as={FiChevronDown} />} variant="whiteBase">
                {t('repos.sort')}: {sortBy === 'updated' ? t('repos.sortUpdated') : sortBy === 'name' ? t('repos.sortName') : t('repos.sortStars')}
              </MenuButton>
              <MenuList>
                <MenuItem onClick={() => setSortBy('updated')}>{t('repos.sortUpdated')}</MenuItem>
                <MenuItem onClick={() => setSortBy('name')}>{t('repos.sortName')}</MenuItem>
                <MenuItem onClick={() => setSortBy('stars')}>{t('repos.sortStars')}</MenuItem>
              </MenuList>
            </Menu>
          </HStack>

          {/* 仓库列表 */}
          <VStack spacing={3} align="stretch">
            {filteredRepos.length === 0 ? (
              <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
                <Box py={12}>
                  <VStack spacing={4}>
                    <Icon as={FiBook} w={12} h={12} color="myGray.400" />
                    <Text color="myGray.600">
                      {searchQuery ? t('repos.noMatch') : t('repos.noRepos')}
                    </Text>
                    {!searchQuery && (
                      <Link href="/new">
                        <Button variant="primary">{t('repos.createFirst')}</Button>
                      </Link>
                    )}
                  </VStack>
                </Box>
              </Box>
            ) : (
              filteredRepos.map((repo) => (
                <Box key={`${repo.userName}/${repo.repositoryName}`} borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
                  <Box py={4} px={4}>
                    <Flex justify="space-between" align="start">
                      <VStack align="start" spacing={2} flex={1}>
                        <Link href={`/${repo.userName}/${repo.repositoryName}`}>
                          <HStack spacing={2}>
                            <Icon as={FiBook} w={4} h={4} color="myGray.500" />
                            <Text fontWeight="medium" color="primary.600" _hover={{ textDecoration: 'underline' }}>
                              {repo.userName}/{repo.repositoryName}
                            </Text>
                          </HStack>
                        </Link>
                        {repo.description && (
                          <Text fontSize="sm" color="myGray.600" noOfLines={2}>
                            {repo.description}
                          </Text>
                        )}
                        <HStack spacing={4} fontSize="xs" color="myGray.500">
                          <HStack spacing={1}>
                            <Box w={2} h={2} bg="yellow.400" borderRadius="full" />
                            <Text>{repo.defaultBranch}</Text>
                          </HStack>
                          <Text>
                            {t('repos.updatedAt')} {new Date(repo.updatedDate).toLocaleDateString(dateLocale)}
                          </Text>
                        </HStack>
                      </VStack>
                      <HStack spacing={2}>
                        {repo.isPrivate && (
                          <Badge colorScheme="yellow" variant="subtle">
                            Private
                          </Badge>
                        )}
                        {(repo.originUserName || repo.parentUserName) && (
                          <Badge colorScheme="blue" variant="subtle">
                            {t('repos.forked')}
                          </Badge>
                        )}
                      </HStack>
                    </Flex>
                  </Box>
                </Box>
              ))
            )}
          </VStack>
        </VStack>
      </Container>
    </Box>
  )
}

export default function ReposPage() {
  return (
    <Suspense fallback={null}>
      <ReposContent />
    </Suspense>
  )
}
