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
  AvatarGroup,
  Badge,
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
  Textarea,
  Switch,
  Divider,
} from '@chakra-ui/react'
import { useEffect, useState, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { api, Organization } from '@/lib/api'
import { FiPlus, FiSettings } from 'react-icons/fi'
import { AiOutlineBank } from 'react-icons/ai'
import { useGithubToast } from '@/app/providers'
import Link from 'next/link'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import { getLocalizedErrorMessage } from '@/lib/errorMessages'

function OrganizationsContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { t } = useI18n()
  const { user } = useCurrentUser()
  const [organizations, setOrganizations] = useState<Organization[]>([])
  const [loading, setLoading] = useState(true)
  const isLoggedIn = !!user
  const isAdmin = !!user?.isAdmin
  const [managedOrganizations, setManagedOrganizations] = useState<string[]>([])
  const { isOpen, onOpen, onClose } = useDisclosure()
  const toast = useGithubToast()

  useEffect(() => {
    if (!user) {
      setManagedOrganizations([])
      return
    }
    if (user.isAdmin) {
      setManagedOrganizations([])
    } else {
      api.listManagedOrganizations().then(names => {
        setManagedOrganizations(names || [])
      }).catch(() => {
        setManagedOrganizations([])
      })
    }
  }, [user])

  const [newOrganization, setNewOrganization] = useState({
    name: '',
    description: '',
    isPrivate: false,
  })

  useEffect(() => {
    loadOrganizations()
  }, [])

  useEffect(() => {
    // 检查 URL 参数，如果 modal=open 则自动打开弹框
    if (searchParams.get('modal') === 'open') {
      onOpen()
      // 清除 URL 参数
      router.replace('/organizations', { scroll: false })
    }
  }, [searchParams])

  const loadOrganizations = async () => {
    try {
      // Show all organizations (aligned with GitBucket)
      const data = await api.listOrganizations()
      setOrganizations(data || [])
    } catch (error) {
      console.error('Failed to load organizations:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreateOrganization = async () => {
    if (!newOrganization.name.trim()) {
      toast({
        title: t('organizations.nameRequired'),
        status: 'warning',
        duration: 2000,
      })
      return
    }

    try {
      await api.createOrganization(newOrganization.name, newOrganization.description, newOrganization.isPrivate)
      toast({
        title: t('organizations.createSuccess'),
        status: 'success',
        duration: 2000,
      })
      onClose()
      setNewOrganization({ name: '', description: '', isPrivate: false })
      loadOrganizations()
    } catch (error: any) {
      toast({
        title: t('organizations.createFailed'),
        description: getLocalizedErrorMessage(error, t),
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

  return (
    <Box bg="myGray.50">
      <Container maxW="container.xl" py={8}>
        <VStack spacing={6} align="stretch">
          <HStack justify="space-between">
            <Heading size="lg" color="myGray.900">
              {t('organizations.title')}
            </Heading>
            {isLoggedIn && isAdmin && (
              <Button leftIcon={<Icon as={FiPlus} />} variant="primary" onClick={onOpen}>
                {t('organizations.newOrganization')}
              </Button>
            )}
          </HStack>

          {organizations.length === 0 ? (
            <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
              <Box py={12}>
                <VStack spacing={4}>
                  <Icon as={AiOutlineBank} w={12} h={12} color="myGray.400" />
                  <Text color="myGray.600">{t('organizations.noOrganizations')}</Text>
                  {isLoggedIn && (
                    <Button variant="primary" onClick={onOpen}>
                      {t('organizations.createFirst')}
                    </Button>
                  )}
                </VStack>
              </Box>
            </Box>
          ) : (
            <VStack spacing={3} align="stretch">
              {organizations.map((organization) => (
                <Box key={organization.userName} borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
                  <Box py={4} px={4}>
                    <HStack justify="space-between" align="start">
                      <HStack spacing={4} align="start">
                        <Avatar size="lg" name={organization.fullName || organization.userName} src={organization.image} />
                        <VStack align="start" spacing={1}>
                          <HStack spacing={2}>
                            <Link href={`/${organization.userName}`}>
                              <Text
                                fontWeight="medium"
                                color="primary.600"
                                fontSize="lg"
                                _hover={{ textDecoration: 'underline' }}
                              >
                                {organization.fullName || organization.userName}
                              </Text>
                            </Link>
                            <Badge colorScheme="blue" variant="subtle">
                              {t('organizations.organizationBadge')}
                            </Badge>
                          </HStack>
                          {organization.description && (
                            <Text fontSize="sm" color="myGray.600">
                              {organization.description}
                            </Text>
                          )}
                          <HStack spacing={3} fontSize="xs" color="myGray.500">
                            <Text>@{organization.userName}</Text>
                          </HStack>
                        </VStack>
                      </HStack>
                      {isLoggedIn && (isAdmin || managedOrganizations.includes(organization.userName)) && (
                        <Button
                          size="sm"
                          variant="whiteBase"
                          leftIcon={<Icon as={FiSettings} />}
                          onClick={() => router.push(`/organizations/${organization.userName}`)}
                        >
                          {t('organizations.manage')}
                        </Button>
                      )}
                    </HStack>
                  </Box>
                </Box>
              ))}
            </VStack>
          )}
        </VStack>
      </Container>

      {/* 新建组织对话框 */}
      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('organizations.createTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('organizations.organizationName')}</FormLabel>
                <Input
                  value={newOrganization.name}
                  onChange={(e) => setNewOrganization({ ...newOrganization, name: e.target.value })}
                  placeholder={t('organizations.organizationNamePlaceholder')}
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('organizations.description')}</FormLabel>
                <Textarea
                  value={newOrganization.description}
                  onChange={(e) => setNewOrganization({ ...newOrganization, description: e.target.value })}
                  placeholder={t('organizations.descriptionPlaceholder')}
                  rows={3}
                />
              </FormControl>
              <FormControl display="flex" alignItems="center">
                <FormLabel htmlFor="is-private" mb="0">
                  {t('organizations.privateOrganization')}
                </FormLabel>
                <Switch
                  id="is-private"
                  isChecked={newOrganization.isPrivate}
                  onChange={(e) => setNewOrganization({ ...newOrganization, isPrivate: e.target.checked })}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button variant="primary" onClick={handleCreateOrganization}>
              {t('common.create')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  )
}

export default function OrganizationsPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <OrganizationsContent />
    </Suspense>
  )
}
