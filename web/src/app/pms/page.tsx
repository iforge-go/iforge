'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  Button,
  VStack,
  HStack,
  SimpleGrid,
  Card,
  CardBody,
  Badge,
  Icon,
  Spinner,
  Input,
  InputGroup,
  InputLeftElement,
  InputRightElement,
  useToast,
} from '@chakra-ui/react'
import { FiFolder, FiRepeat, FiBook, FiUsers, FiPlus, FiClock, FiSearch, FiX } from 'react-icons/fi'
import { Suspense, useEffect, useMemo, useState } from 'react'
import { api } from '@/lib/api'
import { useRouter, useSearchParams } from 'next/navigation'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'

// 角色颜色映射（与 MembersTab 保持一致）
const roleColorMap: Record<string, string> = {
  owner: 'purple',
  admin: 'blue',
  member: 'gray',
  viewer: 'green',
}
const getRoleLabel = (t: (k: string) => string, role: string) => {
  const key = `pms.role${role.charAt(0).toUpperCase() + role.slice(1)}`
  const translated = t(key)
  return translated === key ? role : translated
}

function ScrumContent() {
  const router = useRouter()
  const { t } = useI18n()
  const toast = useToast()
  const { user, authLoading } = useCurrentUser()
  const searchParams = useSearchParams()
  const [projects, setProjects] = useState<any[]>([])
  const [projectsLoading, setProjectsLoading] = useState(true)
  // 搜索词：从 URL ?q= 初始化（Header 跳转过来时自动填充）
  const [searchQuery, setSearchQuery] = useState('')

  useEffect(() => {
    const q = searchParams.get('q')
    if (q) setSearchQuery(q)
  }, [searchParams])

  useEffect(() => {
    if (authLoading) return
    if (!user) {
      router.replace('/login')
      return
    }
    api.getProjects()
      .then((data) => setProjects(data || []))
      .catch(() => {})
      .finally(() => setProjectsLoading(false))
  }, [authLoading, user, router])

  // 清除搜索：清空输入框并从 URL 移除 ?q=
  const handleClearSearch = () => {
    setSearchQuery('')
    if (searchParams.has('q')) {
      router.replace('/pms')
    }
  }

  // 按名称/描述/slug 前端过滤项目
  const filteredProjects = useMemo(() => {
    if (!searchQuery.trim()) return projects
    const q = searchQuery.toLowerCase()
    return projects.filter(p =>
      p.name?.toLowerCase().includes(q) ||
      p.description?.toLowerCase().includes(q) ||
      p.slug?.toLowerCase().includes(q)
    )
  }, [projects, searchQuery])

  // 从项目列表汇总统计数据（后端已批量填充 memberCount/sprintCount/userStoryCount）
  const totals = useMemo(() => {
    let members = 0, sprints = 0, stories = 0
    for (const p of projects) {
      members += p.memberCount || 0
      sprints += p.sprintCount || 0
      stories += p.userStoryCount || 0
    }
    return { members, sprints, stories }
  }, [projects])

  if (authLoading) return null

  if (projectsLoading) {
    return (
      <Container maxW="container.xl" py={20}>
        <VStack spacing={4}>
          <Spinner size="xl" color="blue.500" />
          <Text color="gray.600">{t('common.loading')}</Text>
        </VStack>
      </Container>
    )
  }

  return (
    <Container maxW="container.xl" py={8}>
      <VStack spacing={8} align="stretch">
        {/* 页面标题 */}
        <HStack justify="space-between">
          <VStack align="start" spacing={2}>
            <Heading size="lg">{t('nav.pms')}</Heading>
            <Text color="gray.600">{t('pms.subtitle')}</Text>
          </VStack>
          <Button leftIcon={<FiPlus />} colorScheme="blue" onClick={() => router.push('/projects/new')}>
            {t('pms.newProject')}
          </Button>
        </HStack>

        {/* 统计卡片 */}
        <SimpleGrid columns={{ base: 1, md: 2, lg: 4 }} spacing={6}>
          <Card bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" overflow="hidden">
            <CardBody>
              <HStack spacing={4}>
                <Box p={3} bg="blue.50" borderRadius="lg">
                  <Icon as={FiFolder} boxSize={6} color="blue.500" />
                </Box>
                <VStack align="start" spacing={0}>
                  <Text fontSize="sm" color="gray.600">{t('pms.projects')}</Text>
                  <Heading size="lg">{projects?.length || 0}</Heading>
                </VStack>
              </HStack>
            </CardBody>
          </Card>

          <Card bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" overflow="hidden">
            <CardBody>
              <HStack spacing={4}>
                <Box p={3} bg="green.50" borderRadius="lg">
                  <Icon as={FiRepeat} boxSize={6} color="green.500" />
                </Box>
                <VStack align="start" spacing={0}>
                  <Text fontSize="sm" color="gray.600">{t('pms.sprints')}</Text>
                  <Heading size="lg">{totals.sprints}</Heading>
                </VStack>
              </HStack>
            </CardBody>
          </Card>

          <Card bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" overflow="hidden">
            <CardBody>
              <HStack spacing={4}>
                <Box p={3} bg="purple.50" borderRadius="lg">
                  <Icon as={FiBook} boxSize={6} color="purple.500" />
                </Box>
                <VStack align="start" spacing={0}>
                  <Text fontSize="sm" color="gray.600">{t('pms.userStories')}</Text>
                  <Heading size="lg">{totals.stories}</Heading>
                </VStack>
              </HStack>
            </CardBody>
          </Card>

          <Card bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" overflow="hidden">
            <CardBody>
              <HStack spacing={4}>
                <Box p={3} bg="orange.50" borderRadius="lg">
                  <Icon as={FiUsers} boxSize={6} color="orange.500" />
                </Box>
                <VStack align="start" spacing={0}>
                  <Text fontSize="sm" color="gray.600">{t('pms.teams')}</Text>
                  <Heading size="lg">{totals.members}</Heading>
                </VStack>
              </HStack>
            </CardBody>
          </Card>
        </SimpleGrid>

        {/* 项目列表 */}
        <Box>
          <HStack justify="space-between" mb={4} flexWrap="wrap" spacing={4}>
            <Heading size="md">{t('pms.projects')}</Heading>
            {/* 项目搜索框：与 /repos 页面风格一致 */}
            <HStack spacing={3} flex={1} maxW="400px" justify="flex-end">
              <InputGroup>
                <InputLeftElement>
                  <Icon as={FiSearch} color="myGray.400" />
                </InputLeftElement>
                <Input
                  placeholder={t('pms.searchPlaceholder')}
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  bg="white"
                  borderColor="myGray.200"
                />
                {searchQuery && (
                  <InputRightElement cursor="pointer" onClick={handleClearSearch}>
                    <Icon as={FiX} color="myGray.400" _hover={{ color: 'myGray.700' }} />
                  </InputRightElement>
                )}
              </InputGroup>
            </HStack>
          </HStack>

          {/* 搜索结果提示条 */}
          {searchQuery && (
            <Text fontSize="sm" color="myGray.600" mb={3}>
              {t('pms.searchResultsCount', { count: filteredProjects.length, query: searchQuery })}
            </Text>
          )}

          {!projects || projects.length === 0 ? (
            <Card bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" overflow="hidden">
              <CardBody py={12}>
                <VStack spacing={4}>
                  <Icon as={FiFolder} boxSize={12} color="gray.300" />
                  <Text color="gray.500">{t('common.noData')}</Text>
                  <Button leftIcon={<FiPlus />} colorScheme="blue" onClick={() => router.push('/projects/new')}>
                    {t('pms.createFirstProject')}
                  </Button>
                </VStack>
              </CardBody>
            </Card>
          ) : filteredProjects.length === 0 ? (
            /* 搜索无匹配 */
            <Card bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" overflow="hidden">
              <CardBody py={12}>
                <VStack spacing={4}>
                  <Icon as={FiSearch} boxSize={12} color="gray.300" />
                  <Text color="gray.500">{t('pms.searchNoMatch', { query: searchQuery })}</Text>
                  <Button variant="ghost" onClick={handleClearSearch}>
                    {t('pms.searchClear')}
                  </Button>
                </VStack>
              </CardBody>
            </Card>
          ) : (
            <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} spacing={6}>
              {filteredProjects.map((project) => (
                <Card
                  key={project.slug}
                  bg="white"
                  borderRadius="lg"
                  border="1px solid #DFE2EA"
                  boxShadow="1"
                  overflow="hidden"
                  cursor="pointer"
                  _hover={{ shadow: 'lg', transform: 'translateY(-2px)' }}
                  transition="all 0.2s"
                  onClick={() => router.push(`/projects/${project.slug}`)}
                >
                  <CardBody>
                    <VStack align="start" spacing={3}>
                      <HStack justify="space-between" w="100%">
                        <Heading size="sm" noOfLines={1}>{project.name}</Heading>
                        <HStack spacing={1}>
                          {project.myRole && (
                            <Badge colorScheme={roleColorMap[project.myRole] || 'gray'} variant="subtle" fontSize="xs">
                              {getRoleLabel(t, project.myRole)}
                            </Badge>
                          )}
                          {project.isPrivate && (
                            <Badge colorScheme="red" variant="outline" fontSize="xs">{t('common.private')}</Badge>
                          )}
                        </HStack>
                      </HStack>
                      <Text fontSize="sm" color="gray.600" noOfLines={2} minH="40px">
                        {project.description || t('common.noDescription')}
                      </Text>
                      <HStack spacing={4} fontSize="xs" color="gray.500">
                        <HStack spacing={1}>
                          <Icon as={FiUsers} />
                          <Text>{project.memberCount || 0} {t('pms.members')}</Text>
                        </HStack>
                        <HStack spacing={1}>
                          <Icon as={FiClock} />
                          <Text>{project.sprintCount || 0} {t('pms.sprints')}</Text>
                        </HStack>
                      </HStack>
                    </VStack>
                  </CardBody>
                </Card>
              ))}
            </SimpleGrid>
          )}
        </Box>
      </VStack>
    </Container>
  )
}

export default function ScrumPage() {
  return (
    <Suspense fallback={null}>
      <ScrumContent />
    </Suspense>
  )
}
