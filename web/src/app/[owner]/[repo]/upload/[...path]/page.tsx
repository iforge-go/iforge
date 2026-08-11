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
  Select,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useState, useRef, useCallback, useEffect } from 'react'
import { useParams } from 'next/navigation'
import { api } from '@/lib/api'
import { parseBranchAndPath } from '@/lib/branchPath'
import { FiUpload, FiChevronRight, FiFile, FiX } from 'react-icons/fi'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

export default function UploadFilePage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const pathSegments = params.path as string[]

  const { branches, refreshData } = useRepo()
  const toast = useGithubToast()
  const { t } = useI18n()

  const defaultBranch = branches.find(b => b.isDefault)?.name || branches[0]?.name || 'main'
  // 分支名可能含 "/"（如 feature/task-15），用已知分支列表反向匹配前缀来正确分离分支名和文件路径
  const { ref, subPath: dirPath } = parseBranchAndPath(pathSegments, branches, defaultBranch)

  const [selectedBranch, setSelectedBranch] = useState(ref)

  // branches 延迟加载时，ref 会在 branches 就绪后变为正确值（如从 'feature' 变为 'feature/task-15'）。
  // 此时需同步 selectedBranch，避免下拉框停留在错误的分支上。
  useEffect(() => {
    if (branches.length > 0 && branches.some(b => b.name === ref)) {
      setSelectedBranch(ref)
    }
  }, [ref, branches])

  const [files, setFiles] = useState<File[]>([])
  const [commitMessage, setCommitMessage] = useState('')
  const [uploading, setUploading] = useState(false)
  const [dragActive, setDragActive] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true)
    } else if (e.type === 'dragleave') {
      setDragActive(false)
    }
  }, [])

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)

    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      setFiles(prev => [...prev, ...Array.from(e.dataTransfer.files)])
    }
  }, [])

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      setFiles(prev => [...prev, ...Array.from(e.target.files as ArrayLike<File>)])
    }
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  const removeFile = (index: number) => {
    setFiles(prev => prev.filter((_, i) => i !== index))
  }

  const handleUpload = async () => {
    if (files.length === 0) {
      toast({
        title: t('repo.selectFiles'),
        status: 'warning',
        duration: 2000,
      })
      return
    }

    try {
      setUploading(true)
      const message = commitMessage.trim() || `Upload ${files.length} file${files.length > 1 ? 's' : ''}`

      for (const file of files) {
        await api.uploadFile(owner, repoName, selectedBranch, dirPath, message, file)
      }

      toast({
        title: t('repo.filesUploaded'),
        status: 'success',
        duration: 2000,
      })
      await refreshData()
      window.location.href = `/${owner}/${repoName}/tree/${selectedBranch}${dirPath ? '/' + dirPath : ''}`
    } catch (err) {
      toast({
        title: t('repo.uploadFailed'),
        description: err instanceof Error ? err.message : t('repo.unknownError'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setUploading(false)
    }
  }

  const pathParts = dirPath ? dirPath.split('/').filter(Boolean) : []
  const breadcrumbs = [
    { label: repoName, href: `/${owner}/${repoName}` },
    ...pathParts.map((part, index) => ({
      label: part,
      href: `/${owner}/${repoName}/tree/${ref}/${pathParts.slice(0, index + 1).join('/')}`,
    })),
    { label: t('repo.uploadFiles'), href: undefined },
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
            <Icon as={FiUpload} color="primary.500" />
            <Text fontWeight="medium">{t('repo.uploadFiles')}</Text>
          </HStack>
        </Box>

        <Box p={4}>
          <VStack spacing={4} align="stretch">
            {/* Branch selector */}
            <HStack spacing={2}>
              <Text fontSize="sm" color="myGray.600" minW="80px">
                {t('repo.branch')}
              </Text>
              <Select
                size="sm"
                w="200px"
                value={selectedBranch}
                onChange={(e) => setSelectedBranch(e.target.value)}
              >
                {branches.map((b) => (
                  <option key={b.name} value={b.name}>
                    {b.name}
                  </option>
                ))}
              </Select>
            </HStack>

            {/* Drop zone */}
            <Box
              borderWidth="2px"
              borderStyle="dashed"
              borderColor={dragActive ? 'primary.400' : 'myGray.300'}
              borderRadius="md"
              p={8}
              textAlign="center"
              cursor="pointer"
              bg={dragActive ? 'primary.50' : 'myGray.50'}
              transition="all 0.2s"
              onDragEnter={handleDrag}
              onDragLeave={handleDrag}
              onDragOver={handleDrag}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
            >
              <VStack spacing={3}>
                <Icon as={FiUpload} w={10} h={10} color="myGray.400" />
                <Text fontSize="sm" color="myGray.600">
                  {t('repo.dragDropFiles')}
                </Text>
                <Text fontSize="xs" color="myGray.500">
                  {t('repo.orClickToBrowse')}
                </Text>
              </VStack>
              <Input
                ref={fileInputRef}
                type="file"
                multiple
                display="none"
                onChange={handleFileChange}
              />
            </Box>

            {/* File list */}
            {files.length > 0 && (
              <Box borderWidth="1px" borderColor="myGray.200" borderRadius="md" overflow="hidden">
                <VStack spacing={0} align="stretch">
                  {files.map((file, index) => (
                    <HStack
                      key={index}
                      p={3}
                      borderBottom={index < files.length - 1 ? '1px' : 'none'}
                      borderColor="myGray.200"
                      _hover={{ bg: 'myGray.50' }}
                    >
                      <Icon as={FiFile} color="myGray.500" />
                      <Text fontSize="sm" flex={1} noOfLines={1}>
                        {file.name}
                      </Text>
                      <Text fontSize="xs" color="myGray.500">
                        {(file.size / 1024).toFixed(1)} KB
                      </Text>
                      <Button
                        size="xs"
                        variant="ghost"
                        colorScheme="red"
                        onClick={() => removeFile(index)}
                      >
                        <Icon as={FiX} />
                      </Button>
                    </HStack>
                  ))}
                </VStack>
              </Box>
            )}

            {/* Commit message */}
            <HStack spacing={2}>
              <Text fontSize="sm" color="myGray.600" minW="80px">
                {t('repo.commitMessage')}
              </Text>
              <Input
                size="sm"
                flex={1}
                placeholder={t('repo.commitMessagePlaceholder')}
                value={commitMessage}
                onChange={(e) => setCommitMessage(e.target.value)}
              />
            </HStack>

            {/* Actions */}
            <HStack spacing={3} justify="flex-end">
              <Link href={`/${owner}/${repoName}/tree/${ref}${dirPath ? '/' + dirPath : ''}`}>
                <Button variant="ghost" size="sm">
                  {t('common.cancel')}
                </Button>
              </Link>
              <Button
                variant="primary"
                size="sm"
                leftIcon={<Icon as={FiUpload} />}
                onClick={handleUpload}
                isLoading={uploading}
                loadingText={t('repo.uploading')}
                isDisabled={files.length === 0}
              >
                {t('repo.uploadFiles')}
              </Button>
            </HStack>
          </VStack>
        </Box>
      </Box>
    </VStack>
  )
}
