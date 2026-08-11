'use client'

import {
  Box,
  Text,
  VStack,
  HStack,
  Icon,
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  Button,
  Input,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useState, useRef } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import { FiChevronRight, FiSave, FiX, FiPlus, FiLoader } from 'react-icons/fi'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { parseBranchAndPath, shouldWaitForBranches } from '@/lib/branchPath'

export default function NewFilePage() {
  const params = useParams()
  const router = useRouter()
  const owner = params.owner as string
  const repoName = params.repo as string
  const pathSegments = params.path as string[]

  const { branches, refreshData } = useRepo()
  const toast = useGithubToast()
  const { t } = useI18n()

  // All hooks must be called before any conditional returns
  const [fileName, setFileName] = useState('')
  const [content, setContent] = useState('')
  const [commitMessage, setCommitMessage] = useState('')
  const [saving, setSaving] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [confirmOpen, setConfirmOpen] = useState(false)

  const defaultBranch = branches.find(b => b.isDefault)?.name || branches[0]?.name || 'main'
  const { ref, subPath: dirPath } = parseBranchAndPath(pathSegments, branches, defaultBranch)

  // Wait for branches to load when path has multiple segments (may contain "/" in branch name)
  if (shouldWaitForBranches(pathSegments, branches)) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minH="200px">
        <VStack spacing={3}>
          <FiLoader className="animate-spin" size={24} />
          <Text color="myGray.500" fontSize="sm">{t('repo.loadingBranches')}</Text>
        </VStack>
      </Box>
    )
  }

  const fullPath = dirPath ? `${dirPath}/${fileName}` : fileName

  const handleSave = async () => {
    if (!fileName.trim()) {
      toast({
        title: t('repo.fileNameRequired'),
        status: 'error',
        duration: 3000,
      })
      return
    }

    const message = commitMessage.trim() || `Create ${fileName}`

    try {
      setSaving(true)
      await api.createFile(owner, repoName, ref, fullPath, content, message)
      toast({
        title: t('repo.fileCreated'),
        status: 'success',
        duration: 2000,
      })
      await refreshData()
      window.location.href = `/${owner}/${repoName}`
    } catch (err) {
      toast({
        title: t('repo.createFailed'),
        description: err instanceof Error ? err.message : t('repo.unknownError'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSaving(false)
    }
  }

  const handleCancel = () => {
    if (fileName || content) {
      setConfirmOpen(true)
    } else {
      router.push(`/${owner}/${repoName}/tree/${ref}${dirPath ? '/' + dirPath : ''}`)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Tab') {
      e.preventDefault()
      const textarea = textareaRef.current
      if (!textarea) return

      const start = textarea.selectionStart
      const end = textarea.selectionEnd
      const newContent = content.substring(0, start) + '  ' + content.substring(end)
      setContent(newContent)

      setTimeout(() => {
        textarea.selectionStart = textarea.selectionEnd = start + 2
      }, 0)
    }
  }

  const lineCount = Math.max(content.split('\n').length, 1)

  const pathParts = dirPath ? dirPath.split('/').filter(Boolean) : []
  const breadcrumbs = [
    { label: repoName, href: `/${owner}/${repoName}` },
    ...pathParts.map((part, index) => ({
      label: part,
      href: `/${owner}/${repoName}/tree/${ref}/${pathParts.slice(0, index + 1).join('/')}`,
    })),
    { label: t('repo.newFile'), href: undefined },
  ]

  return (
    <VStack spacing={6} align="stretch">
      <Breadcrumb separator={<Icon as={FiChevronRight} color="myGray.400" />} fontSize="sm">
        {breadcrumbs.map((crumb, index) => (
          <BreadcrumbItem key={index} isCurrentPage={index === breadcrumbs.length - 1}>
            {index === breadcrumbs.length - 1 ? (
              <BreadcrumbLink color="myGray.900" fontWeight="medium">
                {crumb.label}
              </BreadcrumbLink>
            ) : (
              <BreadcrumbLink as={Link} href={crumb.href} color="myGray.600">
                {crumb.label}
              </BreadcrumbLink>
            )}
          </BreadcrumbItem>
        ))}
      </Breadcrumb>

      <Box borderWidth="1px" borderColor="myGray.200" borderRadius="lg" overflow="hidden" bg="white">
        <Box p={4} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
          <HStack spacing={2}>
            <Icon as={FiPlus} w={5} h={5} color="myGray.500" />
            <Text fontSize="lg" fontWeight="medium" color="myGray.900">
              {t('repo.newFile')}
            </Text>
          </HStack>
        </Box>

        <Box p={4} borderBottom="1px" borderColor="myGray.200">
          <HStack spacing={2}>
            {dirPath && (
              <Text fontSize="sm" color="myGray.500" fontFamily="mono">
                {dirPath}/
              </Text>
            )}
            <Input
              value={fileName}
              onChange={(e) => setFileName(e.target.value)}
              placeholder={t('repo.fileNamePlaceholder')}
              size="sm"
              flex={1}
              fontFamily="mono"
              fontWeight="medium"
            />
          </HStack>
        </Box>

        <Box position="relative" display="flex">
          <Box
            w="50px"
            bg="myGray.50"
            borderRight="1px"
            borderColor="myGray.200"
            py={4}
            fontSize="sm"
            fontFamily="mono"
            color="myGray.400"
            textAlign="right"
            userSelect="none"
            overflow="hidden"
          >
            {Array.from({ length: lineCount }, (_, i) => (
              <Box key={i} px={2} h="20px" lineHeight="20px">
                {i + 1}
              </Box>
            ))}
          </Box>
          <textarea
            ref={textareaRef}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t('repo.fileContentPlaceholder')}
            style={{
              border: 'none',
              outline: 'none',
              resize: 'vertical',
              fontSize: '12px',
              fontFamily: 'monospace',
              padding: '16px',
              lineHeight: '20px',
              width: '100%',
              height: `${Math.max(lineCount, 3) * 20 + 32}px`,
              overflow: 'auto',
              background: 'transparent',
            }}
            rows={Math.max(lineCount, 3)}
          />
        </Box>
      </Box>

      <Box borderWidth="1px" borderColor="myGray.200" borderRadius="lg" overflow="hidden" bg="white">
        <Box p={4}>
          <VStack spacing={4} align="stretch">
            <Box>
              <Text fontSize="sm" fontWeight="medium" color="myGray.700" mb={2}>
                {t('repo.commitMessage')}
              </Text>
              <Input
                value={commitMessage}
                onChange={(e) => setCommitMessage(e.target.value)}
                placeholder={`Create ${fileName || 'new file'}`}
                size="sm"
              />
            </Box>

            <HStack justify="flex-end" spacing={3}>
              <Button
                leftIcon={<Icon as={FiX} />}
                variant="whiteBase"
                size="sm"
                onClick={handleCancel}
              >
                {t('common.cancel')}
              </Button>
              <Button
                leftIcon={<Icon as={FiSave} />}
                colorScheme="blue"
                size="sm"
                onClick={handleSave}
                isLoading={saving}
                loadingText={t('repo.creating')}
                isDisabled={!fileName.trim()}
              >
                {t('repo.submitNewFile')}
              </Button>
            </HStack>
          </VStack>
        </Box>
      </Box>

      <Modal isOpen={confirmOpen} onClose={() => setConfirmOpen(false)}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('repo.discardChanges')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>{t('repo.discardChangesConfirm')}</Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={() => setConfirmOpen(false)}>
              {t('repo.continueEditing')}
            </Button>
            <Button
              colorScheme="red"
              onClick={() => {
                setConfirmOpen(false)
                router.push(`/${owner}/${repoName}/tree/${ref}${dirPath ? '/' + dirPath : ''}`)
              }}
            >
              {t('repo.discard')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </VStack>
  )
}
