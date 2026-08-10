'use client'

import { useState, useEffect, useCallback } from 'react'
import {
  Box,
  HStack,
  VStack,
  Text,
  Button,
  IconButton,
  Icon,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  FormControl,
  FormLabel,
  Input,
  Select,
  Spinner,
  Tabs,
  TabList,
  Tab,
  TabPanels,
  TabPanel,
  useToast,
} from '@chakra-ui/react'
import { FiGitBranch, FiPlus, FiX, FiExternalLink } from 'react-icons/fi'
import NextLink from 'next/link'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { TaskBranch, Branch } from '@/lib/types'

interface TaskBranchesProps {
  projectSlug: string
  taskId: number
  canEdit: boolean
}

interface ProjectRepo {
  userName: string
  repositoryName: string
}

export default function TaskBranches({ projectSlug, taskId, canEdit }: TaskBranchesProps) {
  const { t } = useI18n()
  const toast = useToast()

  const [branches, setBranches] = useState<TaskBranch[]>([])
  const [loading, setLoading] = useState(false)

  // Modal state
  const [isOpen, setIsOpen] = useState(false)
  const [repos, setRepos] = useState<ProjectRepo[]>([])
  const [reposLoading, setReposLoading] = useState(false)
  const [selectedRepoFull, setSelectedRepoFull] = useState('') // "owner/repo"
  const [repoBranches, setRepoBranches] = useState<Branch[]>([])
  const [branchesLoading, setBranchesLoading] = useState(false)
  const [selectedBranch, setSelectedBranch] = useState('') // associate mode
  const [newBranchName, setNewBranchName] = useState('') // create mode
  const [baseBranch, setBaseBranch] = useState('') // create mode
  const [submitting, setSubmitting] = useState(false)
  const [activeTab, setActiveTab] = useState(0) // 0 = associate, 1 = create

  const loadBranches = useCallback(async () => {
    if (!projectSlug || !taskId) return
    setLoading(true)
    try {
      const res = await api.getTaskBranches(projectSlug, taskId)
      setBranches(res.branches || [])
    } catch {
      // 静默失败，避免drawer打开时报错刷屏
    } finally {
      setLoading(false)
    }
  }, [projectSlug, taskId])

  useEffect(() => {
    loadBranches()
  }, [loadBranches])

  // 打开弹窗：加载项目关联的仓库列表
  const openModal = async () => {
    setIsOpen(true)
    setSelectedRepoFull('')
    setSelectedBranch('')
    // 创建模式默认分支名：feature/task-{taskId}，便于追溯
    setNewBranchName(`feature/task-${taskId}`)
    setBaseBranch('')
    setRepoBranches([])
    setActiveTab(0)
    setReposLoading(true)
    try {
      const list = await api.getProjectRepositories(projectSlug)
      setRepos(list || [])
      setReposLoading(false)
      // 默认选择第一个仓库并加载其分支
      if (list && list.length > 0) {
        await handleRepoChange(`${list[0].userName}/${list[0].repositoryName}`)
      }
    } catch (error: any) {
      setReposLoading(false)
      toast({ title: t('pms.loadReposFailed'), description: error.message, status: 'error', duration: 3000 })
    }
  }

  // 选择仓库后加载其分支列表
  const handleRepoChange = async (repoFull: string) => {
    setSelectedRepoFull(repoFull)
    setRepoBranches([])
    if (!repoFull) {
      setSelectedBranch('')
      setBaseBranch('')
      return
    }
    const [owner, repo] = repoFull.split('/')
    setBranchesLoading(true)
    try {
      const list = await api.listBranches(owner, repo)
      setRepoBranches(list || [])
      // 默认分支：基准分支和关联标签的预选分支都用它
      const def = list.find((b) => b.isDefault)
      const defaultName = def ? def.name : list[0]?.name || ''
      setBaseBranch(defaultName)
      setSelectedBranch(defaultName)
    } catch (error: any) {
      toast({ title: t('pms.loadBranchesFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setBranchesLoading(false)
    }
  }

  const handleAssociate = async () => {
    if (!selectedRepoFull || !selectedBranch) return
    setSubmitting(true)
    try {
      await api.associateTaskBranch(projectSlug, taskId, {
        repoFullName: selectedRepoFull,
        branchName: selectedBranch,
      })
      toast({ title: t('pms.branchLinkedSuccess'), status: 'success', duration: 2000 })
      setIsOpen(false)
      await loadBranches()
    } catch (error: any) {
      const msg = (error as any).status === 409 ? t('pms.branchAlreadyLinked') : error.message
      toast({ title: msg, status: 'error', duration: 3000 })
    } finally {
      setSubmitting(false)
    }
  }

  const handleCreate = async () => {
    if (!selectedRepoFull || !newBranchName.trim()) return
    setSubmitting(true)
    try {
      await api.createAndAssociateTaskBranch(projectSlug, taskId, {
        repoFullName: selectedRepoFull,
        branchName: newBranchName.trim(),
        from: baseBranch || undefined,
      })
      toast({ title: t('pms.branchCreatedSuccess'), status: 'success', duration: 2000 })
      setIsOpen(false)
      await loadBranches()
    } catch (error: any) {
      const msg = (error as any).status === 409 ? t('pms.branchAlreadyLinked') : error.message
      toast({ title: msg, status: 'error', duration: 3000 })
    } finally {
      setSubmitting(false)
    }
  }

  const handleUnlink = async (branchId: number) => {
    try {
      await api.removeTaskBranch(projectSlug, taskId, branchId)
      toast({ title: t('pms.branchUnlinkedSuccess'), status: 'success', duration: 2000 })
      await loadBranches()
    } catch (error: any) {
      toast({ title: error.message, status: 'error', duration: 3000 })
    }
  }

  return (
    <Box>
      <HStack justify="space-between" mb={2}>
        <HStack spacing={2}>
          <Icon as={FiGitBranch} color="gray.500" />
          <Text fontSize="sm" fontWeight="medium" color="gray.700">
            {t('pms.branches')}
          </Text>
          {branches.length > 0 && (
            <Text fontSize="xs" color="gray.500">({branches.length})</Text>
          )}
        </HStack>
        {canEdit && (
          <Button
            size="xs"
            variant="ghost"
            colorScheme="blue"
            leftIcon={<Icon as={FiPlus} />}
            onClick={openModal}
          >
            {t('pms.addBranch')}
          </Button>
        )}
      </HStack>

      {loading ? (
        <Spinner size="xs" />
      ) : branches.length === 0 ? (
        <Text fontSize="sm" color="gray.400">{t('pms.noBranches')}</Text>
      ) : (
        <VStack align="stretch" spacing={1}>
          {branches.map((b) => {
            const [owner, repo] = b.repoFullName.split('/')
            // 分支名含 "/" 时（如 feature/task-15），需保留 "/" 作为路径分隔符
            // 供 tree 路由的 [...path] catch-all 正确匹配，仅对每段单独编码。
            const treeHref = `/${owner}/${repo}/tree/${b.branchName.split('/').map(encodeURIComponent).join('/')}`
            return (
              <HStack
                key={b.id}
                spacing={2}
                bg="gray.50"
                borderRadius="md"
                px={2}
                py={1.5}
              >
                <Icon as={FiGitBranch} w={3.5} h={3.5} color="gray.500" />
                <HStack spacing={1} fontSize="xs" flex={1} minW={0}>
                  <Text color="gray.500" flexShrink={0}>{b.repoFullName}</Text>
                  <Text color="gray.400">:</Text>
                  <NextLink href={treeHref} target="_blank">
                    <HStack spacing={1} _hover={{ color: 'blue.500' }}>
                      <Text fontWeight="medium" color="gray.700" _hover={{ color: 'blue.500' }} isTruncated>
                        {b.branchName}
                      </Text>
                      <Icon as={FiExternalLink} w={3} h={3} color="gray.400" />
                    </HStack>
                  </NextLink>
                </HStack>
                {canEdit && (
                  <IconButton
                    aria-label={t('pms.unlinkBranch')}
                    icon={<Icon as={FiX} />}
                    size="xs"
                    variant="ghost"
                    colorScheme="gray"
                    minW="20px"
                    w="20px"
                    h="20px"
                    onClick={() => handleUnlink(b.id)}
                  />
                )}
              </HStack>
            )
          })}
        </VStack>
      )}

      {/* 关联/创建分支弹窗 */}
      <Modal isOpen={isOpen} onClose={() => setIsOpen(false)} size="lg">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('pms.addBranch')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack align="stretch" spacing={4}>
              {/* 仓库选择（两个模式共享） */}
              <FormControl isRequired>
                <FormLabel fontSize="sm">{t('pms.selectRepo')}</FormLabel>
                {reposLoading ? (
                  <HStack spacing={2}><Spinner size="xs" /><Text fontSize="sm" color="gray.500">{t('common.loading')}</Text></HStack>
                ) : repos.length === 0 ? (
                  <Text fontSize="sm" color="gray.500">{t('pms.noReposLinked')}</Text>
                ) : (
                  <Select
                    size="sm"
                    value={selectedRepoFull}
                    onChange={(e) => handleRepoChange(e.target.value)}
                    placeholder={t('pms.selectRepo')}
                  >
                    {repos.map((r) => {
                      const full = `${r.userName}/${r.repositoryName}`
                      return <option key={full} value={full}>{full}</option>
                    })}
                  </Select>
                )}
              </FormControl>

              {selectedRepoFull && (
                <Tabs
                  variant="enclosed"
                  colorScheme="blue"
                  size="sm"
                  index={activeTab}
                  onChange={(idx) => setActiveTab(idx)}
                >
                  <TabList>
                    <Tab>{t('pms.createAndAssociate')}</Tab>
                    <Tab>{t('pms.associateBranch')}</Tab>
                  </TabList>
                  <TabPanels>
                    {/* 创建新分支（默认） */}
                    <TabPanel>
                      <VStack align="stretch" spacing={3}>
                        <FormControl isRequired>
                          <FormLabel fontSize="sm">{t('pms.branchName')}</FormLabel>
                          <Input
                            size="sm"
                            value={newBranchName}
                            onChange={(e) => setNewBranchName(e.target.value)}
                            placeholder={t('pms.branchNamePlaceholder')}
                          />
                        </FormControl>
                        <FormControl>
                          <FormLabel fontSize="sm">{t('pms.baseBranch')}</FormLabel>
                          {branchesLoading ? (
                            <HStack spacing={2}><Spinner size="xs" /><Text fontSize="sm" color="gray.500">{t('common.loading')}</Text></HStack>
                          ) : (
                            <Select
                              size="sm"
                              value={baseBranch}
                              onChange={(e) => setBaseBranch(e.target.value)}
                            >
                              {repoBranches.map((b) => (
                                <option key={b.name} value={b.name}>
                                  {b.name}{b.isDefault ? ` (${t('pms.useDefaultBranch')})` : ''}
                                </option>
                              ))}
                            </Select>
                          )}
                        </FormControl>
                      </VStack>
                    </TabPanel>
                    {/* 关联已有分支 */}
                    <TabPanel>
                      <VStack align="stretch" spacing={3}>
                        <FormControl isRequired>
                          <FormLabel fontSize="sm">{t('pms.selectBranch')}</FormLabel>
                          {branchesLoading ? (
                            <HStack spacing={2}><Spinner size="xs" /><Text fontSize="sm" color="gray.500">{t('common.loading')}</Text></HStack>
                          ) : repoBranches.length === 0 ? (
                            <Text fontSize="sm" color="gray.500">{t('pms.noBranchesInRepo')}</Text>
                          ) : (
                            <Select
                              size="sm"
                              value={selectedBranch}
                              onChange={(e) => setSelectedBranch(e.target.value)}
                              placeholder={t('pms.selectBranch')}
                            >
                              {repoBranches.map((b) => (
                                <option key={b.name} value={b.name}>
                                  {b.name}{b.isDefault ? ` (${t('pms.useDefaultBranch')})` : ''}
                                </option>
                              ))}
                            </Select>
                          )}
                        </FormControl>
                      </VStack>
                    </TabPanel>
                  </TabPanels>
                </Tabs>
              )}
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="outline" mr={3} onClick={() => setIsOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              colorScheme="blue"
              isLoading={submitting}
              isDisabled={
                selectedRepoFull === '' ||
                (activeTab === 0 ? newBranchName.trim() === '' : selectedBranch === '')
              }
              onClick={activeTab === 0 ? handleCreate : handleAssociate}
            >
              {activeTab === 0 ? t('pms.createBranch') : t('pms.associateBranch')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  )
}
