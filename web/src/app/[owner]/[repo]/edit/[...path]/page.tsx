'use client'

import {
  Box,
  Container,
  Text,
  VStack,
  HStack,
  Icon,
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  Spinner,
  Button,
  Textarea,
  Input,
  useDisclosure,
  Modal,
  ModalOverlay,
  ModalContent,
  Heading,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useEffect, useState, useRef, useMemo } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import { parseBranchAndPath, shouldWaitForBranches } from '@/lib/branchPath'
import { FiFile, FiChevronRight, FiSave, FiX } from 'react-icons/fi'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

export default function EditPage() {
  const params = useParams()
  const router = useRouter()
  const owner = params.owner as string
  const repoName = params.repo as string
  const pathSegments = params.path as string[]

  const { branches, userRole } = useRepo()
  const toast = useGithubToast()
  const { t } = useI18n()

  const defaultBranch = branches.find(b => b.isDefault)?.name || branches[0]?.name || 'main'
  // 分支名可能含 "/"（如 feature/task-15），用已知分支列表反向匹配前缀来正确分离分支名和文件路径
  const { ref, subPath: filePath } = parseBranchAndPath(pathSegments, branches, defaultBranch)
  const fileName = filePath.split('/').pop() || filePath

  const [content, setContent] = useState('')
  const [originalContent, setOriginalContent] = useState('')
  const [commitMessage, setCommitMessage] = useState(`Update ${fileName}`)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const { isOpen, onOpen, onClose } = useDisclosure()

  // Developer 及以上权限可以编辑文件
  const canEditFile = userRole === 'owner' || userRole === 'member'
  const branchesLoaded = branches.length > 0

  useEffect(() => {
    const loadFile = async () => {
      try {
        setLoading(true)
        const fileContent = await api.getFileContent(owner, repoName, filePath, ref)
        const lines = fileContent.content.split('\n')
        let end = lines.length
        while (end > 0 && lines[end - 1].trim() === '') end--
        const trimmed = lines.slice(0, Math.max(end, 1)).join('\n')
        setContent(trimmed)
        setOriginalContent(trimmed)
        setError(null)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load file')
      } finally {
        setLoading(false)
      }
    }

    // 多段路径可能含 "/" 分支名，需等 branches 加载后才能正确解析
    if (shouldWaitForBranches(pathSegments, branches)) return
    if (filePath) {
      loadFile()
    }
  }, [owner, repoName, filePath, ref, branchesLoaded])

  const handleSave = async () => {
    if (!commitMessage.trim()) {
      toast({
        title: t('repo.commitMessageRequired'),
        status: 'error',
        duration: 3000,
      })
      return
    }

    try {
      setSaving(true)
      await api.updateFile(owner, repoName, ref, filePath, content, commitMessage)
      toast({
        title: t('repo.fileUpdated'),
        status: 'success',
        duration: 2000,
      })
      router.push(`/${owner}/${repoName}/blob/${ref}/${filePath}`)
    } catch (err) {
      toast({
        title: t('repo.saveFailed'),
        description: err instanceof Error ? err.message : t('repo.unknownError'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSaving(false)
    }
  }

  const handleCancel = () => {
    if (content !== originalContent) {
      onOpen()
    } else {
      router.push(`/${owner}/${repoName}/blob/${ref}/${filePath}`)
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

  const lineCount = useMemo(() => {
    if (!content) return 1
    return Math.max(content.split('\n').length, 1)
  }, [content])

  // 权限检查：只有 Developer 及以上可以编辑文件
  if (!canEditFile) {
    return (
      <VStack spacing={6} align="stretch">
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200" p={8}>
          <VStack spacing={4}>
            <Heading size="md" color="myGray.900">{t('common.accessDenied')}</Heading>
            <Text color="myGray.600">{t('repo.editPermissionRequired')}</Text>
            <Link href={`/${owner}/${repoName}/blob/${ref}/${filePath}`}>
              <Button variant="primary">{t('repo.backToFile')}</Button>
            </Link>
          </VStack>
        </Box>
      </VStack>
    )
  }

  if (loading) {
    return (
      <VStack spacing={4}>
        <Spinner size="xl" />
        <Text color="myGray.600">{t('repo.loadingFile')}</Text>
      </VStack>
    )
  }

  if (error) {
    return (
      <VStack spacing={4}>
        <Text color="red.500">{error}</Text>
        <Link href={`/${owner}/${repoName}`}>
          <Button variant="whiteBase">{t('repo.backToRepo')}</Button>
        </Link>
      </VStack>
    )
  }

  const pathParts = filePath.split('/').filter(Boolean)
  const breadcrumbs = [
    { label: repoName, href: `/${owner}/${repoName}` },
    ...pathParts.slice(0, -1).map((part, index) => ({
      label: part,
      href: `/${owner}/${repoName}/tree/${ref}/${pathParts.slice(0, index + 1).join('/')}`,
    })),
    { label: fileName, href: `/${owner}/${repoName}/blob/${ref}/${filePath}` },
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
            <Icon as={FiFile} w={5} h={5} color="myGray.500" />
            <Text fontSize="lg" fontWeight="medium" color="myGray.900">
              {fileName}
            </Text>
          </HStack>
        </Box>

        <Box
          borderWidth="1px"
          borderColor="myGray.200"
          borderRadius="md"
          overflow="auto"
          bg="white"
          maxH="632px"
          display="flex"
        >
          {/* Line numbers */}
          <Box
            w="50px"
            flexShrink={0}
            bg="myGray.50"
            borderRight="1px"
            borderColor="myGray.200"
            py={4}
            fontSize="sm"
            fontFamily="mono"
            color="myGray.400"
            textAlign="right"
            userSelect="none"
          >
            {Array.from({ length: lineCount }, (_, i) => (
              <Box key={i} px={2} h="20px" lineHeight="20px">
                {i + 1}
              </Box>
            ))}
          </Box>
          {/* Editor */}
          <textarea
            ref={textareaRef}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t('repo.fileContentPlaceholder')}
            style={{
              border: 'none',
              outline: 'none',
              resize: 'none',
              overflow: 'hidden',
              fontSize: '12px',
              fontFamily: 'monospace',
              padding: '16px',
              lineHeight: '20px',
              width: '100%',
              height: `${Math.max(lineCount, 30) * 20 + 32}px`,
              background: 'transparent',
            }}
            rows={lineCount}
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
                placeholder={t('repo.commitMessagePlaceholder')}
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
                loadingText={t('repo.saving')}
              >
                {t('repo.submitChanges')}
              </Button>
            </HStack>
          </VStack>
        </Box>
      </Box>

      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('repo.discardChanges')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>{t('repo.discardChangesConfirm')}</Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onClose}>
              {t('repo.continueEditing')}
            </Button>
            <Button
              colorScheme="red"
              onClick={() => {
                onClose()
                router.push(`/${owner}/${repoName}/blob/${ref}/${filePath}`)
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
