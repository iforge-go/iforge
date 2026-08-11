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
  Avatar,
  Badge,
  Tabs,
  TabList,
  Tab,
  TabPanels,
  TabPanel,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  useDisclosure,
  FormControl,
  FormLabel,
  Input,
  Select,
  IconButton,
  Divider,
} from '@chakra-ui/react'
import { useEffect, useState, useCallback } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { api, Organization, Repository, OrganizationMember } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import {
  FiBook,
  FiSettings,
  FiPlus,
  FiTrash2,
  FiEdit2,
  FiUserPlus,
  FiUserMinus,
} from 'react-icons/fi'
import { AiOutlineBank } from 'react-icons/ai'

export default function OrganizationDetailPage() {
  const params = useParams()
  const router = useRouter()
  const organizationName = params.organizationName as string
  const toast = useGithubToast()
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'

  // 检查登录状态
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const { user: currentUser } = useCurrentUser()
  const [isOrganizationManager, setIsOrganizationManager] = useState(false)

  useEffect(() => {
    const token = localStorage.getItem('token')
    if (!token) {
      router.push('/login')
      return
    }
    setIsAuthenticated(true)
  }, [router])

  const [organization, setOrganization] = useState<Organization | null>(null)
  const [members, setMembers] = useState<OrganizationMember[]>([])
  const [repos, setRepos] = useState<Repository[]>([])
  const [loading, setLoading] = useState(true)

  // 模态框状态
  const { isOpen: isEditOpen, onOpen: onEditOpen, onClose: onEditClose } = useDisclosure()
  const { isOpen: isAddMemberOpen, onOpen: onAddMemberOpen, onClose: onAddMemberClose } = useDisclosure()
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()

  const [editForm, setEditForm] = useState({
    name: '',
    description: '',
    isPrivate: false,
  })
  const [newMember, setNewMember] = useState({
    userName: '',
    role: 'member',
  })

  const loadOrganizationData = useCallback(async () => {
    try {
      const organizationData = await api.getOrganization(organizationName)
      setOrganization(organizationData)
      setEditForm({
        name: organizationData.userName,
        description: organizationData.description || '',
        isPrivate: false,
      })

      // 加载成员和仓库
      const [membersData, reposData] = await Promise.all([
        api.getOrganizationMembers(organizationName),
        api.getOrganizationRepos(organizationName),
      ])
      setMembers(membersData || [])
      setRepos(reposData || [])
    } catch {
      toast({
        title: t('organization.loadFailed'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }, [organizationName, t, toast])

  useEffect(() => {
    loadOrganizationData()
  }, [loadOrganizationData])

  useEffect(() => {
    if (currentUser && members.length > 0) {
      const isManager = members.some(
        (m: any) => m.userName === currentUser.userName && m.isManager
      )
      setIsOrganizationManager(isManager || currentUser.isAdmin)
    }
  }, [currentUser, members])

  const handleUpdateOrganization = async () => {
    try {
      await api.updateOrganization(organizationName, editForm)
      toast({
        title: t('organization.updated'),
        status: 'success',
        duration: 2000,
      })
      onEditClose()
      loadOrganizationData()
    } catch (error: any) {
      toast({
        title: t('organization.updateFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleAddMember = async () => {
    if (!newMember.userName.trim()) {
      toast({
        title: t('organization.enterUsername'),
        status: 'warning',
        duration: 2000,
      })
      return
    }

    try {
      await api.addOrganizationMember(organizationName, newMember.userName, newMember.role)
      toast({
        title: t('organization.memberAdded'),
        status: 'success',
        duration: 2000,
      })
      onAddMemberClose()
      setNewMember({ userName: '', role: 'member' })
      loadOrganizationData()
    } catch (error: any) {
      toast({
        title: t('organization.addFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleRemoveMember = async (userName: string) => {
    try {
      await api.removeOrganizationMember(organizationName, userName)
      toast({
        title: t('organization.memberRemoved'),
        status: 'success',
        duration: 2000,
      })
      loadOrganizationData()
    } catch (error: any) {
      toast({
        title: t('organization.removeFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleDeleteOrganization = async () => {
    try {
      await api.deleteOrganization(organizationName)
      toast({
        title: t('organization.deleted'),
        status: 'success',
        duration: 2000,
      })
      router.push('/organizations')
    } catch (error: any) {
      toast({
        title: t('organization.deleteFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  if (loading) {
    return (
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={8}>
          <Text>{t('common.loading')}</Text>
        </Container>
      </Box>
    )
  }

  if (!organization) {
    return (
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={8}>
          <Text>{t('organization.notExists')}</Text>
        </Container>
      </Box>
    )
  }

  if (!isAuthenticated) {
    return null
  }

  return (
    <Box bg="myGray.50">
      <Container maxW="container.xl" py={8}>
        <VStack spacing={6} align="stretch">
          {/* 组织头部 */}
          <HStack justify="space-between" align="start">
            <HStack spacing={4} align="start">
              <Avatar size="xl" name={organization.fullName || organization.userName} src={organization.image} />
              <VStack align="start" spacing={2}>
                <HStack spacing={2}>
                  <Heading size="lg" color="myGray.900">
                    {organization.fullName || organization.userName}
                  </Heading>
                </HStack>
                {organization.description && (
                  <Text color="myGray.600">{organization.description}</Text>
                )}
                <HStack spacing={4} fontSize="sm" color="myGray.500">
                  <HStack spacing={1}>
                    <Icon as={AiOutlineBank} />
                    <Text>{t('organization.memberCount', { count: members.length })}</Text>
                  </HStack>
                  <HStack spacing={1}>
                    <Icon as={FiBook} />
                    <Text>{t('organization.repoCount', { count: repos.length })}</Text>
                  </HStack>
                </HStack>
              </VStack>
            </HStack>
            {isOrganizationManager && (
              <HStack spacing={2}>
                <Button
                  leftIcon={<Icon as={FiEdit2} />}
                  variant="whiteBase"
                  onClick={onEditOpen}
                >
                  {t('organization.edit')}
                </Button>
                {currentUser?.isAdmin && (
                  <Button
                    leftIcon={<Icon as={FiTrash2} />}
                    variant="danger"
                    onClick={onDeleteOpen}
                  >
                    {t('organization.delete')}
                  </Button>
                )}
              </HStack>
            )}
          </HStack>

          <Divider />

          {/* 标签页 */}
          <Tabs variant="line" colorScheme="primary">
            <TabList>
              <Tab>
                <HStack spacing={2}>
                  <Icon as={FiBook} />
                  <Text>{t('organization.repos')}</Text>
                </HStack>
              </Tab>
              <Tab>
                <HStack spacing={2}>
                  <Icon as={AiOutlineBank} />
                  <Text>{t('organization.members')}</Text>
                </HStack>
              </Tab>
              {isOrganizationManager && (
                <Tab>
                  <HStack spacing={2}>
                    <Icon as={FiSettings} />
                    <Text>{t('organization.settings')}</Text>
                  </HStack>
                </Tab>
              )}
            </TabList>

            <TabPanels>
              {/* 仓库标签页 */}
              <TabPanel px={0}>
                <VStack spacing={4} align="stretch">
                  <HStack justify="space-between">
                    <Heading size="md">{t('organization.repoList')}</Heading>
                    {isOrganizationManager && (
                      <Button
                        leftIcon={<Icon as={FiPlus} />}
                        variant="primary"
                        size="sm"
                        onClick={() => router.push(`/new?owner=${organizationName}`)}
                      >
                        {t('organization.newRepo')}
                      </Button>
                    )}
                  </HStack>

                  {repos.length === 0 ? (
                    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" py={12}>
                      <VStack spacing={4}>
                        <Icon as={FiBook} w={12} h={12} color="myGray.400" />
                        <Text color="myGray.600">{t('organization.noRepos')}</Text>
                        <Button
                          leftIcon={<Icon as={FiPlus} />}
                          variant="primary"
                          onClick={() => router.push(`/new?owner=${organizationName}`)}
                        >
                          {t('organization.createRepo')}
                        </Button>
                      </VStack>
                    </Box>
                  ) : (
                    <VStack spacing={3} align="stretch">
                      {repos.map((repo) => (
                        <Box key={`${repo.userName}/${repo.repositoryName}`} borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" py={4} px={6}>
                          <HStack justify="space-between" align="start">
                            <HStack spacing={4} align="start">
                              <Icon as={FiBook} color="myGray.500" mt={1} />
                              <VStack align="start" spacing={1}>
                                <HStack spacing={2}>
                                  <Text
                                    fontWeight="medium"
                                    color="primary.600"
                                    cursor="pointer"
                                    _hover={{ textDecoration: 'underline' }}
                                    onClick={() =>
                                      router.push(`/${repo.userName}/${repo.repositoryName}`)
                                    }
                                  >
                                    {repo.userName}/{repo.repositoryName}
                                  </Text>
                                  {repo.isPrivate && (
                                    <Badge colorScheme="yellow" variant="subtle">
                                      {t('repo.private')}
                                    </Badge>
                                  )}
                                </HStack>
                                {repo.description && (
                                  <Text fontSize="sm" color="myGray.600">
                                    {repo.description}
                                  </Text>
                                )}
                                <HStack spacing={3} fontSize="xs" color="myGray.500">
                                  <Text>{repo.defaultBranch}</Text>
                                </HStack>
                              </VStack>
                            </HStack>
                          </HStack>
                        </Box>
                      ))}
                    </VStack>
                  )}
                </VStack>
              </TabPanel>

              {/* 成员标签页 */}
              <TabPanel px={0}>
                <VStack spacing={4} align="stretch">
                  <HStack justify="space-between">
                    <Heading size="md">{t('organization.memberList')}</Heading>
                    {isOrganizationManager && (
                      <Button
                        leftIcon={<Icon as={FiUserPlus} />}
                        variant="primary"
                        size="sm"
                        onClick={onAddMemberOpen}
                      >
                        {t('organization.addMember')}
                      </Button>
                    )}
                  </HStack>

                  <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
                    <Table variant="simple" borderColor="myGray.200">
                      <Thead bg="myGray.100">
                        <Tr>
                          <Th borderColor="myGray.200">{t('organization.user')}</Th>
                          <Th borderColor="myGray.200">{t('organization.role')}</Th>
                          <Th borderColor="myGray.200">{t('organization.joinedAt')}</Th>
                          <Th borderColor="myGray.200">{t('organization.actions')}</Th>
                        </Tr>
                      </Thead>
                      <Tbody>
                        {members.map((member) => (
                          <Tr key={member.userName}>
                            <Td borderColor="myGray.100">
                              <HStack spacing={3}>
                                <Avatar size="sm" name={member.fullName || member.userName} src={member.image || undefined} />
                                <VStack align="start" spacing={0}>
                                  <Text fontWeight="medium">
                                    {member.fullName || member.userName}
                                  </Text>
                                  <Text fontSize="xs" color="myGray.500">
                                    @{member.userName}
                                  </Text>
                                </VStack>
                              </HStack>
                            </Td>
                            <Td borderColor="myGray.100">
                              <Badge
                                colorScheme={member.isManager ? 'purple' : 'gray'}
                              >
                                {member.isManager ? t('organization.admin') : t('organization.member')}
                              </Badge>
                            </Td>
                            <Td borderColor="myGray.100">
                              <Text fontSize="sm">
                                {member.joinedDate && new Date(member.joinedDate).getFullYear() > 1
                                  ? new Date(member.joinedDate).toLocaleDateString(dateLocale)
                                  : '-'}
                              </Text>
                            </Td>
                            <Td borderColor="myGray.100">
                              {isOrganizationManager && (
                                <IconButton
                                  aria-label={t('organization.removeMember')}
                                  icon={<Icon as={FiUserMinus} />}
                                  size="sm"
                                  variant="ghost"
                                  colorScheme="red"
                                  onClick={() => handleRemoveMember(member.userName)}
                                />
                              )}
                            </Td>
                          </Tr>
                        ))}
                      </Tbody>
                    </Table>
                  </Box>
                </VStack>
              </TabPanel>

              {/* 设置标签页 */}
              {isOrganizationManager && (
              <TabPanel px={0}>
                <VStack spacing={6} align="stretch">
                  <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
                    <VStack spacing={4} align="stretch">
                      <Heading size="md">{t('organization.organizationSettings')}</Heading>
                      <FormControl>
                        <FormLabel>{t('organization.organizationName')}</FormLabel>
                        <Input
                          value={editForm.name}
                          onChange={(e) =>
                            setEditForm({ ...editForm, name: e.target.value })
                          }
                        />
                      </FormControl>
                      <FormControl>
                        <FormLabel>{t('organization.description')}</FormLabel>
                        <Input
                          value={editForm.description}
                          onChange={(e) =>
                            setEditForm({ ...editForm, description: e.target.value })
                          }
                        />
                      </FormControl>
                      <Button
                        variant="primary"
                        onClick={handleUpdateOrganization}
                        alignSelf="flex-end"
                      >
                        {t('organization.saveChanges')}
                      </Button>
                    </VStack>
                  </Box>

                  <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
                    <VStack spacing={4} align="stretch">
                      <Heading size="md" color="red.600">
                        {t('organization.dangerZone')}
                      </Heading>
                      <Text fontSize="sm" color="myGray.600">
                        {t('organization.dangerZoneDesc')}
                      </Text>
                      <Button
                        variant="danger"
                        onClick={onDeleteOpen}
                        alignSelf="flex-end"
                      >
                        {t('organization.deleteOrganization')}
                      </Button>
                    </VStack>
                  </Box>
                </VStack>
              </TabPanel>
              )}
            </TabPanels>
          </Tabs>
        </VStack>
      </Container>

      {/* 编辑组织对话框 */}
      <Modal isOpen={isEditOpen} onClose={onEditClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('organization.editOrganization')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('organization.organizationName')}</FormLabel>
                <Input
                  value={editForm.name}
                  onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('organization.description')}</FormLabel>
                <Input
                  value={editForm.description}
                  onChange={(e) =>
                    setEditForm({ ...editForm, description: e.target.value })
                  }
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onEditClose}>
              {t('organization.cancel')}
            </Button>
            <Button variant="primary" onClick={handleUpdateOrganization}>
              {t('organization.save')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* 添加成员对话框 */}
      <Modal isOpen={isAddMemberOpen} onClose={onAddMemberClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('organization.addMember')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('organization.username')}</FormLabel>
                <Input
                  value={newMember.userName}
                  onChange={(e) =>
                    setNewMember({ ...newMember, userName: e.target.value })
                  }
                  placeholder={t('organization.enterUsername')}
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('organization.role')}</FormLabel>
                <Select
                  value={newMember.role}
                  onChange={(e) => setNewMember({ ...newMember, role: e.target.value })}
                >
                  <option value="member">{t('organization.member')}</option>
                  <option value="admin">{t('organization.admin')}</option>
                </Select>
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onAddMemberClose}>
              {t('organization.cancel')}
            </Button>
            <Button variant="primary" onClick={handleAddMember}>
              {t('organization.add')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* 删除确认对话框 */}
      <Modal isOpen={isDeleteOpen} onClose={onDeleteClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('organization.confirmDelete')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>
              {t('organization.confirmDeleteMessage', { organizationName: organization.fullName || organization.userName })}
            </Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onDeleteClose}>
              {t('organization.cancel')}
            </Button>
            <Button variant="danger" onClick={handleDeleteOrganization}>
              {t('organization.confirmDelete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  )
}
