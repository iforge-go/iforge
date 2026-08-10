'use client'

import Link from 'next/link'
import { useState, useEffect, useMemo, useRef } from 'react'
import {
  Box,
  Button,
  Card,
  CardBody,
  CardHeader,
  Flex,
  Heading,
  HStack,
  Stack,
  Text,
  Badge,
  Divider,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverBody,
  Input,
  IconButton,
  Avatar,
  useToast,
  VStack,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
} from '@chakra-ui/react'
import { FiStar, FiEdit, FiTrash2, FiLock, FiUnlock, FiPlus, FiX, FiBook, FiSearch, FiUsers } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { api } from '@/lib/api'
import { Repository } from '@/lib/types'
import { roleLevel } from '@/lib/projectRole'
import { useProject } from '../ProjectContext'

interface ProjectHeaderProps {
  project: any
  projectSlug: string
  onEditClick: () => void
  onDelete: () => void
}

export default function ProjectHeader({ project, projectSlug, onEditClick, onDelete }: ProjectHeaderProps) {
  const { t } = useI18n()
  const toast = useToast()

  // 角色 / 成员 / 派生权限来自 ProjectContext（与后端 RequireRole 对齐）
  const { members, canManageProject, isProjectOwner } = useProject()

  const [repos, setRepos] = useState<{ userName: string; repositoryName: string }[]>([])
  const [reposLoading, setReposLoading] = useState(true)
  const [isAddOpen, setIsAddOpen] = useState(false)

  // 仓库搜索 state
  const [allRepos, setAllRepos] = useState<Repository[]>([])
  const [reposLoaded, setReposLoaded] = useState(false)
  const [searchKeyword, setSearchKeyword] = useState('')
  const [adding, setAdding] = useState(false)

  // 移除仓库确认 dialog state
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

  // 仅 admin+ 需要加载可选仓库列表（用于关联仓库下拉）
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

  return (
    <>
      {/* 项目信息卡片 */}
      <Card bg="white" borderRadius="lg" border="1px solid #DFE2EA" boxShadow="1" overflow="hidden">
        <CardHeader>
          <Flex justify="space-between" align="center">
            <Flex align="center" gap={3}>
              <FiStar size={40} color="#3182ce" />
              <Heading as="h1" size="2xl">{project.name}</Heading>
              {project.isPrivate ? (
                <Badge colorScheme="gray" variant="outline" px={3} py={1} fontSize="sm">
                  <FiLock style={{ marginRight: 4 }} />
                  {t('common.private')}
                </Badge>
              ) : (
                <Badge colorScheme="green" variant="outline" px={3} py={1} fontSize="sm">
                  <FiUnlock style={{ marginRight: 4 }} />
                  {t('common.public')}
                </Badge>
              )}
            </Flex>
            <Flex align="center" gap={3}>
              {isProjectOwner && (
                <>
                  <Button size="sm" colorScheme="blue" variant="outline" onClick={onEditClick}>
                    <FiEdit style={{ marginRight: 8 }} />
                    {t('common.edit')}
                  </Button>
                  <Button size="sm" colorScheme="red" variant="outline" onClick={onDelete}>
                    <FiTrash2 style={{ marginRight: 8 }} />
                    {t('common.delete')}
                  </Button>
                </>
              )}
            </Flex>
          </Flex>
        </CardHeader>
        <CardBody>
          <Stack spacing={6}>
            <Box>
              <Text fontSize="sm" fontWeight="medium" color="gray.500">{t('common.description')}</Text>
              <Text mt={2} color="gray.700">{project.description || t('common.noDescription')}</Text>
            </Box>

            <Divider />

            <HStack spacing={6} fontSize="sm" color="gray.600" flexWrap="wrap">
              <HStack spacing={1}>
                <Text fontWeight="medium" color="gray.500">{t('common.createdAt')}:</Text>
                <Text>{new Date(project.registeredDate).toLocaleString()}</Text>
              </HStack>
              <HStack spacing={1}>
                <Text fontWeight="medium" color="gray.500">{t('common.updatedAt')}:</Text>
                <Text>{new Date(project.updatedDate).toLocaleString()}</Text>
              </HStack>
            </HStack>

            <Divider />

            {/* 关联仓库 + 团队成员 两列 */}
            <Flex gap={6} direction={{ base: 'column', md: 'row' }}>
              {/* 右列：关联仓库 */}
              <Box flex={1} order={{ md: 2 }}>
                <Flex justify="space-between" align="center" mb={2}>
                  <Text fontSize="sm" fontWeight="medium" color="gray.500">{t('pms.linkedRepositories')}</Text>
                  {canManageProject && (
                    <Popover
                      isOpen={isAddOpen}
                      onOpen={() => setIsAddOpen(true)}
                      onClose={() => { setIsAddOpen(false); setSearchKeyword('') }}
                      placement="bottom-end"
                    >
                      <PopoverTrigger>
                        <Button size="xs" variant="outline" leftIcon={<FiPlus />}>
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
                {reposLoading ? (
                  <Text fontSize="sm" color="gray.400">{t('common.loading')}</Text>
                ) : repos.length === 0 ? (
                  <Text fontSize="sm" color="gray.400">{t('pms.noLinkedRepos')}</Text>
                ) : (
                  <HStack spacing={2} flexWrap="wrap">
                    {repos.map((r) => (
                      <HStack
                        key={`${r.userName}/${r.repositoryName}`}
                        spacing={1}
                        bg="gray.100"
                        borderRadius="full"
                        px={2}
                        py={1}
                      >
                        <FiBook size={12} color="#666" />
                        <Link href={`/${r.userName}/${r.repositoryName}`} style={{ textDecoration: 'none' }}>
                          <Text fontSize="xs" color="gray.700" _hover={{ color: 'blue.500' }}>
                            {r.userName}/{r.repositoryName}
                          </Text>
                        </Link>
                        {canManageProject && (
                          <IconButton
                            size="xs"
                            variant="ghost"
                            icon={<FiX />}
                            onClick={() => setRepoToRemove(r)}
                            aria-label="Remove repository link"
                            minW="16px"
                            w="16px"
                            h="16px"
                          />
                        )}
                      </HStack>
                    ))}
                  </HStack>
                )}
              </Box>

              <Box w="1px" bg="gray.200" display={{ base: 'none', md: 'block' }} order={{ md: 1 }} />

              {/* 左列：团队成员 */}
              <Box flex={1} order={{ md: 0 }}>
                <Flex justify="space-between" align="center" mb={2}>
                  <Text fontSize="sm" fontWeight="medium" color="gray.500">{t('pms.teams')}</Text>
                  <Link href={`/projects/${projectSlug}/members`} style={{ textDecoration: 'none' }}>
                    <Button size="xs" variant="ghost" rightIcon={<FiUsers />}>
                      {t('common.viewAll')}
                    </Button>
                  </Link>
                </Flex>
                {members.length === 0 ? (
                  <Text fontSize="sm" color="gray.400">{t('pms.noMembers')}</Text>
                ) : (
                  <HStack spacing={2} flexWrap="wrap">
                    {[...members].sort((a, b) => roleLevel(b.role) - roleLevel(a.role)).slice(0, 6).map((m) => (
                      <HStack
                        key={m.userName}
                        spacing={1}
                        bg="gray.100"
                        borderRadius="full"
                        px={2}
                        py={1}
                      >
                        <Avatar size="2xs" name={m.fullName || m.userName} src={m.image || undefined} />
                        <Text fontSize="xs" color="gray.700">{m.userName}</Text>
                        {m.role && m.role !== 'member' && (
                          <Badge colorScheme="purple" variant="subtle" fontSize="2xs">
                            {t(`pms.role${m.role.charAt(0).toUpperCase() + m.role.slice(1)}`)}
                          </Badge>
                        )}
                      </HStack>
                    ))}
                    {members.length > 6 && (
                      <Text fontSize="xs" color="gray.500">+{members.length - 6}</Text>
                    )}
                  </HStack>
                )}
              </Box>
            </Flex>
          </Stack>
        </CardBody>
      </Card>

      {/* 移除仓库确认 AlertDialog */}
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
    </>
  )
}
