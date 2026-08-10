'use client'

import { useState, useEffect, useMemo, useRef } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import {
  Box,
  Button,
  Card,
  CardBody,
  Flex,
  Heading,
  HStack,
  IconButton,
  Input,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverBody,
  Spinner,
  Stack,
  Text,
  Badge,
  VStack,
  useToast,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
} from '@chakra-ui/react'
import { FiGitBranch, FiPlus, FiSearch, FiBook, FiX, FiLock } from 'react-icons/fi'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import { useProject } from '../ProjectContext'
import { Repository } from '@/lib/types'

// 代码页（对齐 Jira Code tab，iForge 特色：项目关联的 Git 仓库）
// 复用 ProjectHeader.tsx 的仓库关联/取消关联逻辑，但用卡片布局展示
export default function CodePage() {
  const params = useParams()
  const projectSlug = params.slug as string
  const { t } = useI18n()
  const toast = useToast()
  const { user, authLoading } = useCurrentUser()
  const { project, loading: projectLoading, canManageProject } = useProject()

  const [repos, setRepos] = useState<{ userName: string; repositoryName: string }[]>([])
  const [reposLoading, setReposLoading] = useState(true)
  const [isAddOpen, setIsAddOpen] = useState(false)

  // 仓库搜索 state（复用 ProjectHeader 模式）
  const [allRepos, setAllRepos] = useState<Repository[]>([])
  const [reposLoaded, setReposLoaded] = useState(false)
  const [searchKeyword, setSearchKeyword] = useState('')
  const [adding, setAdding] = useState(false)

  // 移除仓库确认
  const [repoToRemove, setRepoToRemove] = useState<{ userName: string; repositoryName: string } | null>(null)
  const [removing, setRemoving] = useState(false)
  const cancelRef = useRef(null as any)

  useEffect(() => {
    if (!projectSlug) return
    setReposLoading(true)
    api.getProjectRepositories(projectSlug)
      .then((data) => setRepos(data || []))
      .catch(() => {})
      .finally(() => setReposLoading(false))
  }, [projectSlug])

  // 仅 admin+ 加载可选仓库列表
  useEffect(() => {
    if (!canManageProject || reposLoaded) return
    api.listRepos()
      .then((data) => setAllRepos(data || []))
      .catch(() => {})
      .finally(() => setReposLoaded(true))
  }, [canManageProject, reposLoaded])

  const filteredRepos = useMemo(() => {
    const linkedSet = new Set(repos.map((r) => `${r.userName}/${r.repositoryName}`))
    const kw = searchKeyword.trim().toLowerCase()
    return allRepos
      .filter((r) => !linkedSet.has(`${r.userName}/${r.repositoryName}`))
      .filter((r) => {
        if (!kw) return true
        return (
          r.userName.toLowerCase().includes(kw) ||
          r.repositoryName.toLowerCase().includes(kw)
        )
      })
      .slice(0, 50)
  }, [allRepos, repos, searchKeyword])

  async function handleSelectRepo(r: Repository) {
    setAdding(true)
    try {
      await api.addProjectRepository(projectSlug, { userName: r.userName, repositoryName: r.repositoryName })
      setRepos((prev) => [...prev, { userName: r.userName, repositoryName: r.repositoryName }])
      setSearchKeyword('')
      setIsAddOpen(false)
      toast({ title: t('pms.repoLinkSuccess'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setAdding(false)
    }
  }

  async function handleConfirmRemoveRepo() {
    if (!repoToRemove) return
    setRemoving(true)
    try {
      await api.removeProjectRepository(projectSlug, repoToRemove.userName, repoToRemove.repositoryName)
      setRepos((prev) => prev.filter((r) => !(r.userName === repoToRemove.userName && r.repositoryName === repoToRemove.repositoryName)))
      toast({ title: t('pms.repoRemoveSuccess'), status: 'success', duration: 2000 })
      setRepoToRemove(null)
    } catch (error: any) {
      toast({ title: t('common.error'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setRemoving(false)
    }
  }

  if (authLoading) return null

  if (!user) {
    return (
      <Box py={20}>
        <Text textAlign="center">{t('common.pleaseLogin')}</Text>
      </Box>
    )
  }

  if (projectLoading || reposLoading) {
    return (
      <Flex justify="center" align="center" minH="60vh">
        <Spinner size="xl" color="blue.500" />
      </Flex>
    )
  }

  if (!project) return null

  return (
    <Stack spacing={4}>
      {/* Header */}
      <Flex justify="space-between" align="center">
        <HStack spacing={2}>
          <FiGitBranch color="#3182ce" />
          <Heading size="md">{t('pms.linkedRepositories')}</Heading>
          <Badge colorScheme="gray" variant="subtle">{repos.length}</Badge>
        </HStack>
        {canManageProject && (
          <Popover
            isOpen={isAddOpen}
            onOpen={() => setIsAddOpen(true)}
            onClose={() => { setIsAddOpen(false); setSearchKeyword('') }}
            placement="bottom-end"
          >
            <PopoverTrigger>
              <Button size="sm" colorScheme="blue" leftIcon={<FiPlus />}>
                {t('pms.addRepository')}
              </Button>
            </PopoverTrigger>
            <PopoverContent w="340px">
              <PopoverBody p={2}>
                <Box position="relative" mb={2}>
                  <Input
                    size="sm"
                    placeholder={t('pms.searchRepoPlaceholder')}
                    value={searchKeyword}
                    onChange={(e) => setSearchKeyword(e.target.value)}
                    pl={8}
                    autoFocus
                  />
                  <Box
                    position="absolute"
                    left={2}
                    top="50%"
                    transform="translateY(-50%)"
                    color="gray.400"
                    pointerEvents="none"
                  >
                    <FiSearch size={14} />
                  </Box>
                </Box>
                <Box maxH="280px" overflowY="auto">
                  {!reposLoaded ? (
                    <Text fontSize="sm" color="gray.400" p={2}>{t('common.loading')}</Text>
                  ) : filteredRepos.length === 0 ? (
                    <Text fontSize="sm" color="gray.400" p={2}>{t('pms.noMatchingRepo')}</Text>
                  ) : (
                    filteredRepos.map((r) => (
                      <Flex
                        key={`${r.userName}/${r.repositoryName}`}
                        align="center"
                        px={2}
                        py={2}
                        borderRadius="md"
                        cursor={adding ? 'wait' : 'pointer'}
                        opacity={adding ? 0.5 : 1}
                        _hover={{ bg: 'gray.100' }}
                        onClick={() => !adding && handleSelectRepo(r)}
                      >
                        <VStack align="start" spacing={0} flex={1} overflow="hidden">
                          <Text fontSize="sm" color="gray.800" noOfLines={1}>
                            <Text as="span" color="gray.500">{r.userName}</Text>
                            <Text as="span" color="gray.400"> / </Text>
                            <Text as="span" fontWeight="medium">{r.repositoryName}</Text>
                          </Text>
                          {r.description && (
                            <Text fontSize="xs" color="gray.500" noOfLines={1}>{r.description}</Text>
                          )}
                        </VStack>
                        {r.isPrivate && (
                          <Badge size="xs" colorScheme="gray" variant="subtle" fontSize="2xs">
                            <FiLock style={{ marginRight: 4 }} />
                            {t('common.private')}
                          </Badge>
                        )}
                      </Flex>
                    ))
                  )}
                </Box>
              </PopoverBody>
            </PopoverContent>
          </Popover>
        )}
      </Flex>

      {/* 仓库卡片列表 */}
      {repos.length === 0 ? (
        <Flex direction="column" align="center" justify="center" py={20} gap={4}>
          <FiBook size={48} color="#94a3b8" />
          <Text color="gray.500" fontSize="lg">{t('pms.noLinkedRepos')}</Text>
        </Flex>
      ) : (
        <VStack spacing={3} align="stretch">
          {repos.map((r) => (
            <Card
              key={`${r.userName}/${r.repositoryName}`}
              bg="white"
              borderRadius="lg"
              border="1px solid #DFE2EA"
              boxShadow="1"
              overflow="hidden"
              _hover={{ borderColor: 'blue.200', boxShadow: 'md' }}
              transition="all 0.2s"
            >
              <CardBody>
                <Flex align="center" justify="space-between">
                  <HStack spacing={3} flex={1} as={Link} href={`/${r.userName}/${r.repositoryName}`} style={{ textDecoration: 'none' }}>
                    <FiBook size={20} color="#3182ce" />
                    <VStack align="start" spacing={0} flex={1}>
                      <Text fontSize="sm" fontWeight="medium" color="gray.800" _hover={{ color: 'blue.600' }}>
                        <Text as="span" color="gray.500">{r.userName}</Text>
                        <Text as="span" color="gray.400"> / </Text>
                        <Text as="span" fontWeight="medium">{r.repositoryName}</Text>
                      </Text>
                    </VStack>
                  </HStack>
                  {canManageProject && (
                    <IconButton
                      size="xs"
                      variant="ghost"
                      icon={<FiX />}
                      onClick={() => setRepoToRemove(r)}
                      aria-label="Remove repository link"
                      color="gray.400"
                      _hover={{ color: 'red.500' }}
                    />
                  )}
                </Flex>
              </CardBody>
            </Card>
          ))}
        </VStack>
      )}

      {/* 移除仓库确认 */}
      <AlertDialog
        isOpen={repoToRemove !== null}
        leastDestructiveRef={cancelRef}
        onClose={() => setRepoToRemove(null)}
        blockScrollOnMount={false}
      >
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              {t('pms.removeRepoTitle')}
            </AlertDialogHeader>
            <AlertDialogBody>
              {repoToRemove && t('pms.removeRepoConfirm', { repo: `${repoToRemove.userName}/${repoToRemove.repositoryName}` })}
            </AlertDialogBody>
            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={() => setRepoToRemove(null)}>
                {t('common.cancel')}
              </Button>
              <Button colorScheme="red" onClick={handleConfirmRemoveRepo} ml={3} isLoading={removing}>
                {t('common.confirm')}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>
    </Stack>
  )
}
