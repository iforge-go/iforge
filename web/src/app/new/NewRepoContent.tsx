'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  Button,
  VStack,
  HStack,
  FormControl,
  FormLabel,
  Input,
  Textarea,
  Switch,
  Icon,
  Radio,
  RadioGroup,
  Stack,
  Select,
  Tabs,
  TabList,
  Tab,
  TabPanels,
  TabPanel,
  Spinner,
} from '@chakra-ui/react'
import { useRouter, useSearchParams } from 'next/navigation'
import { useState, useEffect } from 'react'
import { api, Organization } from '@/lib/api'
import { FiBook, FiDownload, FiCopy } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'

export default function NewRepoContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const toast = useGithubToast()
  const { t } = useI18n()
  const [loading, setLoading] = useState(false)
  const { user } = useCurrentUser()
  const [organizations, setOrganizations] = useState<Organization[]>([])
  const [formData, setFormData] = useState({
    owner: '',
    name: '',
    description: '',
    isPrivate: 'public',
    initReadme: true,
  })

  useEffect(() => {
    const token = localStorage.getItem('token')
    if (!token) {
      router.push('/login')
      return
    }
    const loadData = async () => {
      try {
        const organizationsData = await api.listMyOrganizations()
        setOrganizations(organizationsData || [])
      } catch (error) {
        router.push('/login')
      }
    }
    loadData()
  }, [])

  useEffect(() => {
    if (user) {
      const urlOwner = searchParams.get('owner')
      setFormData(prev => ({ ...prev, owner: urlOwner || user.userName }))
    }
  }, [user])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!formData.name.trim()) {
      toast({ title: t('newRepo.nameRequired'), status: 'warning', duration: 2000 })
      return
    }
    if (!formData.owner.trim()) {
      toast({ title: t('newRepo.ownerRequired'), status: 'warning', duration: 2000 })
      return
    }
    setLoading(true)
    try {
      const repo = await api.createRepo(
        formData.name,
        formData.description,
        formData.isPrivate === 'private',
        formData.owner
      )
      toast({ title: t('newRepo.createSuccess'), status: 'success', duration: 2000 })
      router.push(`/${repo.userName}/${repo.repositoryName}`)
    } catch (error: any) {
      toast({ title: t('newRepo.createFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setLoading(false)
    }
  }

  // Import form state
  const [importForm, setImportForm] = useState({
    sourceUrl: '',
    owner: '',
    repoName: '',
    description: '',
    isPrivate: 'public',
  })
  const [importLoading, setImportLoading] = useState(false)
  const [parsingUrl, setParsingUrl] = useState(false)

  // Sync import owner with create owner when user loads
  useEffect(() => {
    setImportForm((prev) => ({ ...prev, owner: formData.owner }))
  }, [formData.owner])

  const handleParseUrl = async () => {
    const url = importForm.sourceUrl.trim()
    if (!url) return
    setParsingUrl(true)
    try {
      const parsed = await api.parseRepoUrl(url)
      if (parsed.repo && !importForm.repoName) {
        setImportForm((prev) => ({ ...prev, repoName: parsed.repo }))
      }
    } catch {
      // Silent fail - user can still fill manually
    } finally {
      setParsingUrl(false)
    }
  }

  const handleImport = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!importForm.sourceUrl.trim()) {
      toast({ title: t('newRepo.sourceUrl'), status: 'warning', duration: 2000 })
      return
    }
    if (!importForm.repoName.trim()) {
      toast({ title: t('newRepo.nameRequired'), status: 'warning', duration: 2000 })
      return
    }
    if (!importForm.owner.trim()) {
      toast({ title: t('newRepo.ownerRequired'), status: 'warning', duration: 2000 })
      return
    }
    setImportLoading(true)
    try {
      const repo = await api.importRepo({
        owner: importForm.owner,
        repoName: importForm.repoName,
        sourceUrl: importForm.sourceUrl,
        isPrivate: importForm.isPrivate === 'private',
        description: importForm.description,
      })
      toast({ title: t('newRepo.importSuccess'), status: 'success', duration: 2000 })
      router.push(`/${repo.userName}/${repo.repositoryName}`)
    } catch (error: any) {
      toast({ title: t('newRepo.importFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setImportLoading(false)
    }
  }

  // Template form state
  const [templateForm, setTemplateForm] = useState({
    templateRepo: '',
    owner: '',
    newRepoName: '',
    isPrivate: 'public',
  })
  const [templateLoading, setTemplateLoading] = useState(false)

  // Sync template owner with create owner when user loads
  useEffect(() => {
    setTemplateForm((prev) => ({ ...prev, owner: formData.owner }))
  }, [formData.owner])

  const handleCreateFromTemplate = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = templateForm.templateRepo.trim()
    if (!trimmed) {
      toast({ title: t('newRepo.templateRepoRequired'), status: 'warning', duration: 2000 })
      return
    }
    const parts = trimmed.split('/')
    if (parts.length !== 2 || !parts[0] || !parts[1]) {
      toast({ title: t('newRepo.invalidTemplateFormat'), status: 'warning', duration: 2000 })
      return
    }
    if (!templateForm.newRepoName.trim()) {
      toast({ title: t('newRepo.nameRequired'), status: 'warning', duration: 2000 })
      return
    }
    if (!templateForm.owner.trim()) {
      toast({ title: t('newRepo.ownerRequired'), status: 'warning', duration: 2000 })
      return
    }
    setTemplateLoading(true)
    try {
      const repo = await api.createFromTemplate({
        templateOwner: parts[0],
        templateRepo: parts[1],
        newOwner: templateForm.owner,
        newRepo: templateForm.newRepoName.trim(),
        isPrivate: templateForm.isPrivate === 'private',
      })
      toast({ title: t('newRepo.templateSuccess'), status: 'success', duration: 2000 })
      router.push(`/${repo.userName}/${repo.repositoryName}`)
    } catch (error: any) {
      toast({ title: t('newRepo.templateFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setTemplateLoading(false)
    }
  }

  return (
    <Box bg="myGray.50">
      <Container maxW="container.md" py={8}>
        <VStack spacing={6} align="stretch">
          <Heading size="lg" color="myGray.900">
            {t('newRepo.title')}
          </Heading>

          <Tabs variant="line" colorScheme="primary" isLazy>
            <TabList borderBottomColor="myGray.200">
              <Tab>
                <HStack spacing={2}>
                  <Icon as={FiBook} />
                  <Text>{t('newRepo.tabCreate')}</Text>
                </HStack>
              </Tab>
              <Tab>
                <HStack spacing={2}>
                  <Icon as={FiDownload} />
                  <Text>{t('newRepo.tabImport')}</Text>
                </HStack>
              </Tab>
              <Tab>
                <HStack spacing={2}>
                  <Icon as={FiCopy} />
                  <Text>{t('newRepo.tabTemplate')}</Text>
                </HStack>
              </Tab>
            </TabList>

            <TabPanels>
              {/* Create new repo */}
              <TabPanel px={0} pt={6}>
                <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
                  <Box as="form" onSubmit={handleSubmit}>
                    <VStack spacing={5}>
                      <FormControl isRequired>
                        <FormLabel color="myGray.700">{t('newRepo.owner')}</FormLabel>
                        <Select
                          value={formData.owner}
                          onChange={(e) => setFormData({ ...formData, owner: e.target.value })}
                        >
                          {user && (
                            <option value={user.userName}>
                              {user.userName} {t('newRepo.ownerPersonal')}
                            </option>
                          )}
                          {organizations.map((organization) => (
                            <option key={organization.userName} value={organization.userName}>
                              {organization.userName} {t('newRepo.ownerOrganization')}
                            </option>
                          ))}
                        </Select>
                      </FormControl>

                      <FormControl isRequired>
                        <FormLabel color="myGray.700">{t('newRepo.repoName')}</FormLabel>
                        <Input
                          value={formData.name}
                          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                          placeholder={t('newRepo.repoNamePlaceholder')}
                        />
                      </FormControl>

                      <FormControl>
                        <FormLabel color="myGray.700">{t('newRepo.description')}</FormLabel>
                        <Textarea
                          value={formData.description}
                          onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                          placeholder={t('newRepo.descriptionPlaceholder')}
                          rows={3}
                        />
                      </FormControl>

                      <FormControl>
                        <FormLabel color="myGray.700">{t('newRepo.visibility')}</FormLabel>
                        <RadioGroup value={formData.isPrivate} onChange={(v) => setFormData({ ...formData, isPrivate: v })}>
                          <Stack spacing={3}>
                            <HStack
                              p={3}
                              border="1px solid"
                              borderColor={formData.isPrivate === 'public' ? 'primary.500' : 'myGray.200'}
                              borderRadius="md"
                              cursor="pointer"
                              onClick={() => setFormData({ ...formData, isPrivate: 'public' })}
                              align="start"
                            >
                              <Radio value="public" colorScheme="primary" mt={1} />
                              <VStack align="start" spacing={0}>
                                <Text fontWeight="medium" color="myGray.900">{t('newRepo.public')}</Text>
                                <Text fontSize="xs" color="myGray.500">{t('newRepo.publicDesc')}</Text>
                              </VStack>
                            </HStack>
                            <HStack
                              p={3}
                              border="1px solid"
                              borderColor={formData.isPrivate === 'private' ? 'primary.500' : 'myGray.200'}
                              borderRadius="md"
                              cursor="pointer"
                              onClick={() => setFormData({ ...formData, isPrivate: 'private' })}
                              align="start"
                            >
                              <Radio value="private" colorScheme="primary" mt={1} />
                              <VStack align="start" spacing={0}>
                                <Text fontWeight="medium" color="myGray.900">{t('newRepo.private')}</Text>
                                <Text fontSize="xs" color="myGray.500">{t('newRepo.privateDesc')}</Text>
                              </VStack>
                            </HStack>
                          </Stack>
                        </RadioGroup>
                      </FormControl>

                      <HStack justify="flex-end" w="full">
                        <Button variant="whiteBase" onClick={() => router.back()}>
                          {t('common.cancel')}
                        </Button>
                        <Button variant="primary" type="submit" isLoading={loading} leftIcon={<Icon as={FiBook} />}>
                          {t('newRepo.create')}
                        </Button>
                      </HStack>
                    </VStack>
                  </Box>
                </Box>
              </TabPanel>

              {/* Import repo */}
              <TabPanel px={0} pt={6}>
                <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
                  <VStack spacing={2} align="stretch" mb={4}>
                    <Heading size="md">{t('newRepo.importTitle')}</Heading>
                    <Text fontSize="sm" color="myGray.600">
                      {t('newRepo.importDesc')}
                    </Text>
                  </VStack>
                  <Box as="form" onSubmit={handleImport}>
                    <VStack spacing={5}>
                      <FormControl isRequired>
                        <FormLabel color="myGray.700">{t('newRepo.sourceUrl')}</FormLabel>
                        <HStack spacing={2}>
                          <Input
                            value={importForm.sourceUrl}
                            onChange={(e) => setImportForm({ ...importForm, sourceUrl: e.target.value })}
                            onBlur={handleParseUrl}
                            placeholder={t('newRepo.sourceUrlPlaceholder')}
                          />
                          {parsingUrl && <Spinner size="sm" color="primary.500" />}
                        </HStack>
                      </FormControl>

                      <FormControl isRequired>
                        <FormLabel color="myGray.700">{t('newRepo.owner')}</FormLabel>
                        <Select
                          value={importForm.owner}
                          onChange={(e) => setImportForm({ ...importForm, owner: e.target.value })}
                        >
                          {user && (
                            <option value={user.userName}>
                              {user.userName} {t('newRepo.ownerPersonal')}
                            </option>
                          )}
                          {organizations.map((organization) => (
                            <option key={organization.userName} value={organization.userName}>
                              {organization.userName} {t('newRepo.ownerOrganization')}
                            </option>
                          ))}
                        </Select>
                      </FormControl>

                      <FormControl isRequired>
                        <FormLabel color="myGray.700">{t('newRepo.repoName')}</FormLabel>
                        <Input
                          value={importForm.repoName}
                          onChange={(e) => setImportForm({ ...importForm, repoName: e.target.value })}
                          placeholder={t('newRepo.repoNamePlaceholder')}
                        />
                      </FormControl>

                      <FormControl>
                        <FormLabel color="myGray.700">{t('newRepo.description')}</FormLabel>
                        <Textarea
                          value={importForm.description}
                          onChange={(e) => setImportForm({ ...importForm, description: e.target.value })}
                          placeholder={t('newRepo.descriptionPlaceholder')}
                          rows={3}
                        />
                      </FormControl>

                      <FormControl>
                        <FormLabel color="myGray.700">{t('newRepo.visibility')}</FormLabel>
                        <RadioGroup value={importForm.isPrivate} onChange={(v) => setImportForm({ ...importForm, isPrivate: v })}>
                          <Stack spacing={3}>
                            <HStack
                              p={3}
                              border="1px solid"
                              borderColor={importForm.isPrivate === 'public' ? 'primary.500' : 'myGray.200'}
                              borderRadius="md"
                              cursor="pointer"
                              onClick={() => setImportForm({ ...importForm, isPrivate: 'public' })}
                              align="start"
                            >
                              <Radio value="public" colorScheme="primary" mt={1} />
                              <VStack align="start" spacing={0}>
                                <Text fontWeight="medium" color="myGray.900">{t('newRepo.public')}</Text>
                                <Text fontSize="xs" color="myGray.500">{t('newRepo.publicDesc')}</Text>
                              </VStack>
                            </HStack>
                            <HStack
                              p={3}
                              border="1px solid"
                              borderColor={importForm.isPrivate === 'private' ? 'primary.500' : 'myGray.200'}
                              borderRadius="md"
                              cursor="pointer"
                              onClick={() => setImportForm({ ...importForm, isPrivate: 'private' })}
                              align="start"
                            >
                              <Radio value="private" colorScheme="primary" mt={1} />
                              <VStack align="start" spacing={0}>
                                <Text fontWeight="medium" color="myGray.900">{t('newRepo.private')}</Text>
                                <Text fontSize="xs" color="myGray.500">{t('newRepo.privateDesc')}</Text>
                              </VStack>
                            </HStack>
                          </Stack>
                        </RadioGroup>
                      </FormControl>

                      <HStack justify="flex-end" w="full">
                        <Button variant="whiteBase" onClick={() => router.back()}>
                          {t('common.cancel')}
                        </Button>
                        <Button variant="primary" type="submit" isLoading={importLoading} leftIcon={<Icon as={FiDownload} />}>
                          {t('newRepo.importButton')}
                        </Button>
                      </HStack>
                    </VStack>
                  </Box>
                </Box>
              </TabPanel>

              {/* Create from template */}
              <TabPanel px={0} pt={6}>
                <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
                  <VStack spacing={2} align="stretch" mb={4}>
                    <Heading size="md">{t('newRepo.templateTitle')}</Heading>
                    <Text fontSize="sm" color="myGray.600">
                      {t('newRepo.templateDesc')}
                    </Text>
                  </VStack>
                  <Box as="form" onSubmit={handleCreateFromTemplate}>
                    <VStack spacing={5}>
                      <FormControl isRequired>
                        <FormLabel color="myGray.700">{t('newRepo.templateRepo')}</FormLabel>
                        <Input
                          value={templateForm.templateRepo}
                          onChange={(e) => setTemplateForm({ ...templateForm, templateRepo: e.target.value })}
                          placeholder={t('newRepo.templateRepoPlaceholder')}
                        />
                        <Text fontSize="xs" color="myGray.500" mt={1}>
                          {t('newRepo.templateRepoHint')}
                        </Text>
                      </FormControl>

                      <FormControl isRequired>
                        <FormLabel color="myGray.700">{t('newRepo.owner')}</FormLabel>
                        <Select
                          value={templateForm.owner}
                          onChange={(e) => setTemplateForm({ ...templateForm, owner: e.target.value })}
                        >
                          {user && (
                            <option value={user.userName}>
                              {user.userName} {t('newRepo.ownerPersonal')}
                            </option>
                          )}
                          {organizations.map((organization) => (
                            <option key={organization.userName} value={organization.userName}>
                              {organization.userName} {t('newRepo.ownerOrganization')}
                            </option>
                          ))}
                        </Select>
                      </FormControl>

                      <FormControl isRequired>
                        <FormLabel color="myGray.700">{t('newRepo.repoName')}</FormLabel>
                        <Input
                          value={templateForm.newRepoName}
                          onChange={(e) => setTemplateForm({ ...templateForm, newRepoName: e.target.value })}
                          placeholder={t('newRepo.repoNamePlaceholder')}
                        />
                      </FormControl>

                      <FormControl>
                        <FormLabel color="myGray.700">{t('newRepo.visibility')}</FormLabel>
                        <RadioGroup value={templateForm.isPrivate} onChange={(v) => setTemplateForm({ ...templateForm, isPrivate: v })}>
                          <Stack spacing={3}>
                            <HStack
                              p={3}
                              border="1px solid"
                              borderColor={templateForm.isPrivate === 'public' ? 'primary.500' : 'myGray.200'}
                              borderRadius="md"
                              cursor="pointer"
                              onClick={() => setTemplateForm({ ...templateForm, isPrivate: 'public' })}
                              align="start"
                            >
                              <Radio value="public" colorScheme="primary" mt={1} />
                              <VStack align="start" spacing={0}>
                                <Text fontWeight="medium" color="myGray.900">{t('newRepo.public')}</Text>
                                <Text fontSize="xs" color="myGray.500">{t('newRepo.publicDesc')}</Text>
                              </VStack>
                            </HStack>
                            <HStack
                              p={3}
                              border="1px solid"
                              borderColor={templateForm.isPrivate === 'private' ? 'primary.500' : 'myGray.200'}
                              borderRadius="md"
                              cursor="pointer"
                              onClick={() => setTemplateForm({ ...templateForm, isPrivate: 'private' })}
                              align="start"
                            >
                              <Radio value="private" colorScheme="primary" mt={1} />
                              <VStack align="start" spacing={0}>
                                <Text fontWeight="medium" color="myGray.900">{t('newRepo.private')}</Text>
                                <Text fontSize="xs" color="myGray.500">{t('newRepo.privateDesc')}</Text>
                              </VStack>
                            </HStack>
                          </Stack>
                        </RadioGroup>
                      </FormControl>

                      <HStack justify="flex-end" w="full">
                        <Button variant="whiteBase" onClick={() => router.back()}>
                          {t('common.cancel')}
                        </Button>
                        <Button variant="primary" type="submit" isLoading={templateLoading} leftIcon={<Icon as={FiCopy} />}>
                          {t('newRepo.templateCreateButton')}
                        </Button>
                      </HStack>
                    </VStack>
                  </Box>
                </Box>
              </TabPanel>
            </TabPanels>
          </Tabs>
        </VStack>
      </Container>
    </Box>
  )
}
