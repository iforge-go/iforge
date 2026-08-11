'use client'

import {
  Box,
  Heading,
  Text,
  Button,
  VStack,
  HStack,
  Badge,
  Icon,
  Code,
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
  useDisclosure,
  Spinner,
} from '@chakra-ui/react'
import { useEffect, useState, useCallback } from 'react'
import { useParams } from 'next/navigation'
import { api, Tag } from '@/lib/api'
import { FiTag, FiPlus, FiTrash2 } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { useGithubToast } from '@/app/providers'

export default function TagsPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'
  const toast = useGithubToast()

  const [tags, setTags] = useState<Tag[]>([])
  const [loading, setLoading] = useState(true)

  const [tagName, setTagName] = useState('')
  const [tagTarget, setTagTarget] = useState('')
  const [tagMessage, setTagMessage] = useState('')
  const [creating, setCreating] = useState(false)
  const { isOpen: isCreateOpen, onOpen: onCreateOpen, onClose: onCreateClose } = useDisclosure()

  const [deletingTag, setDeletingTag] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()

  const loadTags = useCallback(() => {
    setLoading(true)
    api.listTags(owner, repoName)
      .then(data => setTags(data || []))
      .finally(() => setLoading(false))
  }, [owner, repoName])

  useEffect(() => {
    loadTags()
  }, [loadTags])

  const handleCreateTag = async () => {
    if (!tagName.trim()) {
      toast({ title: t('repo.tagNameLabel'), status: 'warning', duration: 2000 })
      return
    }
    setCreating(true)
    try {
      await api.createTag(owner, repoName, {
        tagName: tagName.trim(),
        target: tagTarget.trim() || undefined,
        message: tagMessage.trim() || undefined,
      })
      toast({ title: t('repo.tagCreatedSuccess'), status: 'success', duration: 2000 })
      onCreateClose()
      setTagName('')
      setTagTarget('')
      setTagMessage('')
      loadTags()
    } catch (error: any) {
      toast({ title: t('repo.createFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setCreating(false)
    }
  }

  const handleDeleteTag = async () => {
    if (!deletingTag) return
    setDeleting(true)
    try {
      await api.deleteTag(owner, repoName, deletingTag)
      toast({ title: t('repo.tagDeletedSuccess'), status: 'success', duration: 2000 })
      onDeleteClose()
      setDeletingTag(null)
      loadTags()
    } catch (error: any) {
      toast({ title: t('repo.deleteBranchFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setDeleting(false)
    }
  }

  const openDeleteModal = (name: string) => {
    setDeletingTag(name)
    onDeleteOpen()
  }

  if (loading) {
    return (
      <Text>{t('common.loading')}</Text>
    )
  }

  return (
    <VStack spacing={6} align="stretch">
      <HStack justify="space-between">
        <Heading size="lg" color="myGray.900">
          {t('repo.tags')}
        </Heading>
        <Button leftIcon={<Icon as={FiPlus} />} variant="primary" size="sm" onClick={onCreateOpen}>
          {t('repo.createTagTitle')}
        </Button>
      </HStack>

      <VStack spacing={3} align="stretch">
        {tags.length === 0 ? (
          <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
            <Box py={12}>
              <VStack spacing={4}>
                <Icon as={FiTag} w={12} h={12} color="myGray.400" />
                <Text color="myGray.600">{t('repo.noTags')}</Text>
              </VStack>
            </Box>
          </Box>
        ) : (
          tags.map((tag) => (
            <Box key={tag.name} borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
              <Box py={4} px={4}>
                <HStack justify="space-between">
                  <HStack spacing={3}>
                    <Icon as={FiTag} color="primary.500" />
                    <Text fontWeight="medium" color="myGray.900">
                      {tag.name}
                    </Text>
                    <Badge colorScheme="blue" variant="subtle">
                      {tag.isRelease ? t('repo.release') : t('repo.tag')}
                    </Badge>
                  </HStack>
                  <HStack spacing={3}>
                    <Code colorScheme="gray" fontSize="xs">
                      {tag.commitId.substring(0, 7)}
                    </Code>
                    <Text fontSize="xs" color="myGray.500">
                      {new Date(tag.createdAt).toLocaleDateString(dateLocale)}
                    </Text>
                    <Button
                      leftIcon={<Icon as={FiTrash2} />}
                      variant="ghost"
                      size="sm"
                      colorScheme="red"
                      onClick={() => openDeleteModal(tag.name)}
                    >
                      {t('common.delete')}
                    </Button>
                  </HStack>
                </HStack>
              </Box>
            </Box>
          ))
        )}
      </VStack>

      <Modal isOpen={isCreateOpen} onClose={onCreateClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('repo.createTagTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4} align="stretch">
              <FormControl isRequired>
                <FormLabel>{t('repo.tagNameLabel')}</FormLabel>
                <Input
                  value={tagName}
                  onChange={(e) => setTagName(e.target.value)}
                  placeholder={t('repo.tagNamePlaceholder')}
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('repo.tagTargetLabel')}</FormLabel>
                <Input
                  value={tagTarget}
                  onChange={(e) => setTagTarget(e.target.value)}
                  placeholder={t('repo.tagTargetPlaceholder')}
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('repo.tagMessageLabel')}</FormLabel>
                <Textarea
                  value={tagMessage}
                  onChange={(e) => setTagMessage(e.target.value)}
                  placeholder={t('repo.tagMessagePlaceholder')}
                  rows={3}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onCreateClose}>{t('common.cancel')}</Button>
            <Button variant="primary" onClick={handleCreateTag} isLoading={creating}>
              {creating ? <Spinner size="sm" /> : t('repo.createTagTitle')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      <Modal isOpen={isDeleteOpen} onClose={onDeleteClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('repo.deleteTagTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text fontSize="sm" color="myGray.600">
              {t('repo.deleteTagConfirm', { name: deletingTag || '' })}
            </Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onDeleteClose}>{t('common.cancel')}</Button>
            <Button colorScheme="red" variant="solid" onClick={handleDeleteTag} isLoading={deleting}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </VStack>
  )
}
