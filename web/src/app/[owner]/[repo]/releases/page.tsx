'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  Button,
  VStack,
  HStack,
  Badge,
  Icon,
  useDisclosure,
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
  Textarea,
  IconButton,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  Code,
  Spinner,
  Divider,
} from '@chakra-ui/react'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { api, Release, ReleaseAsset } from '@/lib/api'
import { FiTag, FiPlus, FiEdit, FiTrash2, FiMoreVertical, FiDownload, FiPaperclip } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { useGithubToast } from '@/app/providers'
import { sanitizeHtml } from '@/lib/sanitize'

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

export default function ReleasesPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const { t, locale } = useI18n()
  const toast = useGithubToast()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'

  const [releases, setReleases] = useState<Release[]>([])
  const [loading, setLoading] = useState(true)
  const { isOpen: isCreateOpen, onOpen: onCreateOpen, onClose: onCreateClose } = useDisclosure()
  const { isOpen: isEditOpen, onOpen: onEditOpen, onClose: onEditClose } = useDisclosure()
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()

  const [selectedRelease, setSelectedRelease] = useState<Release | null>(null)
  const [createData, setCreateData] = useState({
    tag: '',
    name: '',
    content: '',
  })
  const [editData, setEditData] = useState({
    name: '',
    content: '',
  })
  const [submitting, setSubmitting] = useState(false)

  // Asset state
  const [assetsMap, setAssetsMap] = useState<Record<string, ReleaseAsset[]>>({})
  const [uploadTargetTag, setUploadTargetTag] = useState<string | null>(null)
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [uploadLabel, setUploadLabel] = useState('')
  const [uploading, setUploading] = useState(false)
  const { isOpen: isUploadOpen, onOpen: onUploadOpen, onClose: onUploadClose } = useDisclosure()
  const [deletingAsset, setDeletingAsset] = useState<{ tag: string; asset: ReleaseAsset } | null>(null)
  const { isOpen: isAssetDeleteOpen, onOpen: onAssetDeleteOpen, onClose: onAssetDeleteClose } = useDisclosure()

  const loadReleases = async () => {
    try {
      const data = await api.listReleases(owner, repoName)
      setReleases(data || [])
    } catch (error) {
      console.error('Failed to load releases:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadReleases()
  }, [owner, repoName])

  const handleCreate = async () => {
    if (!createData.tag || !createData.name) {
      toast({
        title: t('release.tagAndNameRequired'),
        status: 'error',
        duration: 3000,
      })
      return
    }

    setSubmitting(true)
    try {
      await api.createRelease(owner, repoName, createData)
      toast({
        title: t('release.created'),
        status: 'success',
        duration: 3000,
      })
      onCreateClose()
      setCreateData({ tag: '', name: '', content: '' })
      loadReleases()
    } catch (error) {
      toast({
        title: t('release.createFailed'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSubmitting(false)
    }
  }

  const handleEdit = (release: Release) => {
    setSelectedRelease(release)
    setEditData({
      name: release.name,
      content: release.content || '',
    })
    onEditOpen()
  }

  const handleUpdate = async () => {
    if (!selectedRelease || !editData.name) {
      toast({
        title: t('release.nameRequired'),
        status: 'error',
        duration: 3000,
      })
      return
    }

    setSubmitting(true)
    try {
      await api.updateRelease(owner, repoName, selectedRelease.tag, editData)
      toast({
        title: t('release.updated'),
        status: 'success',
        duration: 3000,
      })
      onEditClose()
      loadReleases()
    } catch (error) {
      toast({
        title: t('release.updateFailed'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = (release: Release) => {
    setSelectedRelease(release)
    onDeleteOpen()
  }

  const confirmDelete = async () => {
    if (!selectedRelease) return

    setSubmitting(true)
    try {
      await api.deleteRelease(owner, repoName, selectedRelease.tag)
      toast({
        title: t('release.deleted'),
        status: 'success',
        duration: 3000,
      })
      onDeleteClose()
      loadReleases()
    } catch (error) {
      toast({
        title: t('release.deleteFailed'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSubmitting(false)
    }
  }

  // Assets
  const loadAssets = async (tag: string) => {
    try {
      const assets = await api.listReleaseAssets(owner, repoName, tag)
      setAssetsMap(prev => ({ ...prev, [tag]: assets || [] }))
    } catch {
      // Silently fail
    }
  }

  useEffect(() => {
    releases.forEach(release => {
      loadAssets(release.tag)
    })
  }, [releases])

  const openUploadModal = (tag: string) => {
    setUploadTargetTag(tag)
    setUploadFile(null)
    setUploadLabel('')
    onUploadOpen()
  }

  const handleUploadAsset = async () => {
    if (!uploadFile || !uploadTargetTag) {
      toast({ title: t('release.selectFile'), status: 'warning', duration: 2000 })
      return
    }
    setUploading(true)
    try {
      await api.uploadReleaseAsset(owner, repoName, uploadTargetTag, uploadFile, uploadLabel || undefined)
      toast({ title: t('release.assetUploadSuccess'), status: 'success', duration: 2000 })
      onUploadClose()
      setUploadFile(null)
      setUploadLabel('')
      await loadAssets(uploadTargetTag)
    } catch (error: any) {
      toast({ title: t('release.assetUploadFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setUploading(false)
    }
  }

  const openAssetDeleteModal = (tag: string, asset: ReleaseAsset) => {
    setDeletingAsset({ tag, asset })
    onAssetDeleteOpen()
  }

  const handleDeleteAsset = async () => {
    if (!deletingAsset) return
    setSubmitting(true)
    try {
      await api.deleteReleaseAsset(owner, repoName, deletingAsset.tag, deletingAsset.asset.assetId)
      toast({ title: t('release.assetDeleted'), status: 'success', duration: 2000 })
      onAssetDeleteClose()
      await loadAssets(deletingAsset.tag)
    } catch (error: any) {
      toast({ title: t('release.assetDeleteFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setSubmitting(false)
    }
  }

  const handleDownloadAsset = (tag: string, asset: ReleaseAsset) => {
    const url = api.getAssetDownloadUrl(owner, repoName, tag, asset.assetId)
    const authToken = api.getToken()
    // Use fetch with auth header to download
    fetch(url, {
      headers: authToken ? { Authorization: authToken } : {},
      credentials: 'include',
    })
      .then(response => response.blob())
      .then(blob => {
        const downloadUrl = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = downloadUrl
        a.download = asset.fileName
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        window.URL.revokeObjectURL(downloadUrl)
      })
      .catch(() => {
        toast({ title: t('release.assetUploadFailed'), status: 'error', duration: 2000 })
      })
  }

  if (loading) {
    return (
      <Container maxW="container.xl" py={8}>
        <Text>{t('common.loading')}</Text>
      </Container>
    )
  }

  return (
    <Container maxW="container.xl" py={8}>
      <VStack spacing={6} align="stretch">
        <HStack justify="space-between">
          <Heading size="lg" color="myGray.900">
            {t('repo.releases')}
          </Heading>
          <Button leftIcon={<FiPlus />} colorScheme="primary" onClick={onCreateOpen}>
            {t('release.newRelease')}
          </Button>
        </HStack>

        {releases.length === 0 ? (
          <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
            <Box py={12}>
              <VStack spacing={4}>
                <Icon as={FiTag} w={12} h={12} color="myGray.400" />
                <Text color="myGray.600">{t('release.noReleases')}</Text>
                <Button leftIcon={<FiPlus />} colorScheme="primary" onClick={onCreateOpen}>
                  {t('release.createFirst')}
                </Button>
              </VStack>
            </Box>
          </Box>
        ) : (
          <VStack spacing={4} align="stretch">
            {releases.map((release) => (
              <Box key={release.tag} borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
                <Box py={4} px={6}>
                  <VStack align="stretch" spacing={3}>
                    <HStack justify="space-between" align="start">
                      <VStack align="start" spacing={2} flex={1}>
                        <HStack spacing={3}>
                          <Heading size="md" color="myGray.900">
                            {release.name}
                          </Heading>
                          <Badge colorScheme="green">{t('release.latest')}</Badge>
                        </HStack>
                        <HStack spacing={2}>
                          <Badge colorScheme="purple" fontSize="sm">
                            {release.tag}
                          </Badge>
                          <Text fontSize="sm" color="myGray.500">
                            {new Date(release.registeredDate).toLocaleDateString(dateLocale)}
                          </Text>
                        </HStack>
                      </VStack>
                      <Menu>
                        <MenuButton
                          as={IconButton}
                          icon={<FiMoreVertical />}
                          variant="ghost"
                          size="sm"
                        />
                        <MenuList>
                          <MenuItem icon={<FiEdit />} onClick={() => handleEdit(release)}>
                            {t('common.edit')}
                          </MenuItem>
                          <MenuItem icon={<FiTrash2 />} onClick={() => handleDelete(release)} color="red.500">
                            {t('common.delete')}
                          </MenuItem>
                        </MenuList>
                      </Menu>
                    </HStack>

                    {release.content && (
                      <Box
                        fontSize="sm"
                        color="myGray.700"
                        dangerouslySetInnerHTML={{ __html: sanitizeHtml(release.content) }}
                      />
                    )}

                    {/* Assets */}
                    <Divider />
                    <VStack align="stretch" spacing={2}>
                      <HStack justify="space-between">
                        <HStack spacing={2}>
                          <Icon as={FiPaperclip} color="myGray.500" />
                          <Text fontSize="sm" fontWeight="medium" color="myGray.700">
                            {t('release.assets')}
                          </Text>
                        </HStack>
                        <Button
                          size="xs"
                          variant="ghost"
                          leftIcon={<Icon as={FiPlus} />}
                          onClick={() => openUploadModal(release.tag)}
                        >
                          {t('release.uploadAsset')}
                        </Button>
                      </HStack>
                      {(assetsMap[release.tag] || []).length === 0 ? (
                        <Text fontSize="xs" color="myGray.500">{t('release.noAssets')}</Text>
                      ) : (
                        <VStack align="stretch" spacing={1}>
                          {(assetsMap[release.tag] || []).map((asset) => (
                            <HStack
                              key={asset.assetId}
                              justify="space-between"
                              p={2}
                              bg="myGray.50"
                              borderRadius="md"
                              spacing={3}
                            >
                              <HStack spacing={2} flex={1} minW={0}>
                                <Icon as={FiPaperclip} color="myGray.400" w={3} h={3} flexShrink={0} />
                                <Text fontSize="sm" color="myGray.900" noOfLines={1}>
                                  {asset.fileName}
                                </Text>
                                {asset.label && (
                                  <Badge colorScheme="gray" variant="subtle" fontSize="xs">
                                    {asset.label}
                                  </Badge>
                                )}
                                <Text fontSize="xs" color="myGray.500" flexShrink={0}>
                                  {formatFileSize(asset.size)}
                                </Text>
                              </HStack>
                              <HStack spacing={1} flexShrink={0}>
                                <IconButton
                                  aria-label={t('release.download')}
                                  icon={<Icon as={FiDownload} />}
                                  variant="ghost"
                                  size="xs"
                                  onClick={() => handleDownloadAsset(release.tag, asset)}
                                />
                                <IconButton
                                  aria-label={t('common.delete')}
                                  icon={<Icon as={FiTrash2} />}
                                  variant="ghost"
                                  size="xs"
                                  colorScheme="red"
                                  onClick={() => openAssetDeleteModal(release.tag, asset)}
                                />
                              </HStack>
                            </HStack>
                          ))}
                        </VStack>
                      )}
                    </VStack>
                  </VStack>
                </Box>
              </Box>
            ))}
          </VStack>
        )}
      </VStack>

      {/* Create Release Modal */}
      <Modal isOpen={isCreateOpen} onClose={onCreateClose} size="xl">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('release.newRelease')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('release.tag')}</FormLabel>
                <Input
                  value={createData.tag}
                  onChange={(e) => setCreateData({ ...createData, tag: e.target.value })}
                  placeholder="v1.0.0"
                />
              </FormControl>
              <FormControl isRequired>
                <FormLabel>{t('release.title')}</FormLabel>
                <Input
                  value={createData.name}
                  onChange={(e) => setCreateData({ ...createData, name: e.target.value })}
                  placeholder={t('release.titlePlaceholder')}
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('release.content')}</FormLabel>
                <Textarea
                  value={createData.content}
                  onChange={(e) => setCreateData({ ...createData, content: e.target.value })}
                  placeholder={t('release.contentPlaceholder')}
                  rows={6}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onCreateClose}>
              {t('common.cancel')}
            </Button>
            <Button colorScheme="primary" onClick={handleCreate} isLoading={submitting}>
              {t('release.create')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Edit Release Modal */}
      <Modal isOpen={isEditOpen} onClose={onEditClose} size="xl">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('release.editRelease')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('release.title')}</FormLabel>
                <Input
                  value={editData.name}
                  onChange={(e) => setEditData({ ...editData, name: e.target.value })}
                  placeholder={t('release.titlePlaceholder')}
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('release.content')}</FormLabel>
                <Textarea
                  value={editData.content}
                  onChange={(e) => setEditData({ ...editData, content: e.target.value })}
                  placeholder={t('release.contentPlaceholder')}
                  rows={6}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onEditClose}>
              {t('common.cancel')}
            </Button>
            <Button colorScheme="primary" onClick={handleUpdate} isLoading={submitting}>
              {t('common.save')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal isOpen={isDeleteOpen} onClose={onDeleteClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('release.deleteRelease')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>
              {t('release.confirmDelete', { name: selectedRelease?.name ?? '' })}
            </Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onDeleteClose}>
              {t('common.cancel')}
            </Button>
            <Button colorScheme="red" onClick={confirmDelete} isLoading={submitting}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Upload Asset Modal */}
      <Modal isOpen={isUploadOpen} onClose={onUploadClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('release.uploadAsset')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('release.selectFile')}</FormLabel>
                <Input
                  type="file"
                  p={1}
                  onChange={(e) => setUploadFile(e.target.files?.[0] || null)}
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('release.assetLabel')}</FormLabel>
                <Input
                  value={uploadLabel}
                  onChange={(e) => setUploadLabel(e.target.value)}
                  placeholder={t('release.assetLabelPlaceholder')}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onUploadClose}>
              {t('common.cancel')}
            </Button>
            <Button colorScheme="primary" onClick={handleUploadAsset} isLoading={uploading}>
              {uploading ? <Spinner size="sm" /> : t('release.uploadAsset')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Delete Asset Confirmation Modal */}
      <Modal isOpen={isAssetDeleteOpen} onClose={onAssetDeleteClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('release.deleteAssetTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text fontSize="sm" color="myGray.600">
              {t('release.deleteAssetConfirm', { name: deletingAsset?.asset.fileName || '' })}
            </Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onAssetDeleteClose}>
              {t('common.cancel')}
            </Button>
            <Button colorScheme="red" onClick={handleDeleteAsset} isLoading={submitting}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Container>
  )
}
