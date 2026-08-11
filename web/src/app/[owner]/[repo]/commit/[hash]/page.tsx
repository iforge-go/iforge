'use client'

import {
  Box,
  Heading,
  Text,
  VStack,
  HStack,
  Icon,
  Avatar,
  Code,
  Badge,
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  Button,
  useClipboard,
  Flex,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useEffect, useState, Suspense } from 'react'
import { useParams, useSearchParams } from 'next/navigation'
import { api, CommitInfo, CombinedCommitStatus, Deployment } from '@/lib/api'
import { FiCopy, FiCheck, FiChevronRight, FiCheckCircle, FiXCircle, FiClock, FiAlertCircle, FiPackage, FiExternalLink, FiFileText } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { FileTree } from '@/components/FileTree'
import { SplitDiffView } from '@/components/SplitDiffView'
import { parsePatch } from '@/lib/diff'
import { formatRelativeTime } from '@/lib/time'
import { useI18n } from '@/contexts/I18nContext'

function CommitContent() {
  const params = useParams()
  const searchParams = useSearchParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const commitHash = params.hash as string
  const branch = searchParams.get('branch') || 'main'
  const toast = useGithubToast()
  const { t } = useI18n()

  const [commit, setCommit] = useState<CommitInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedFile, setSelectedFile] = useState<string>('')
  const [combinedStatus, setCombinedStatus] = useState<CombinedCommitStatus | null>(null)
  const [deployments, setDeployments] = useState<Deployment[]>([])

  useEffect(() => {
    api.getCommit(owner, repoName, commitHash)
      .then((data) => {
        setCommit(data)
        if (data.files && data.files.length > 0) {
          setSelectedFile(data.files[0].filename)
        }
      })
      .catch((err) => {
        setError(err.message || 'Failed to load commit')
      })
      .finally(() => {
        setLoading(false)
      })
    api.getCombinedCommitStatus(owner, repoName, commitHash)
      .then((data) => setCombinedStatus(data || {}))
      .catch(() => {})
    api.listDeploymentsByCommit(owner, repoName, commitHash)
      .then((data) => setDeployments(data.items || []))
      .catch(() => {})
  }, [owner, repoName, commitHash])

  const { onCopy, hasCopied } = useClipboard(commitHash)

  const handleCopy = () => {
    onCopy()
    toast({
      title: t('repo.copied'),
      description: t('repo.commitHashCopied'),
      status: 'success',
      duration: 2000,
    })
  }

  if (loading) {
    return (
      <Box py={8}>
        <Text color="myGray.600">{t('common.loading')}</Text>
      </Box>
    )
  }

  if (error || !commit) {
    return (
      <Box py={8}>
        <Text color="red.500">{error || t('repo.commitNotExist')}</Text>
      </Box>
    )
  }

  const shortHash = commitHash.substring(0, 7)
  const files = commit.files || []
  const totalAdditions = files.reduce((sum, f) => sum + f.additions, 0)
  const totalDeletions = files.reduce((sum, f) => sum + f.deletions, 0)

  const selectedFileData = files.find((f) => f.filename === selectedFile)
  const fileDiff = selectedFileData?.patch
    ? parsePatch(selectedFileData.filename, selectedFileData.status, selectedFileData.patch)
    : null

  return (
    <VStack spacing={6} align="stretch">
      {/* Breadcrumb */}
      <Breadcrumb separator={<FiChevronRight />} fontSize="sm" color="myGray.600">
        <BreadcrumbItem>
          <BreadcrumbLink as={Link} href={`/${owner}/${repoName}`}>
            {repoName}
          </BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbItem>
          <BreadcrumbLink as={Link} href={`/${owner}/${repoName}/commits/${branch}`}>
            {t('repo.commits')}
          </BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbItem isCurrentPage>
          <BreadcrumbLink>{shortHash}</BreadcrumbLink>
        </BreadcrumbItem>
      </Breadcrumb>

      {/* Commit Header */}
      <Box borderWidth="1px" borderRadius="lg" p={6} bg="white" borderColor="myGray.200">
        <HStack justify="space-between" flexWrap="wrap" gap={4} mb={3}>
          <HStack spacing={2}>
            <Heading size="sm" color="myGray.900">
              Commit {shortHash}
            </Heading>
          </HStack>
          <HStack spacing={2}>
            <Button
              as={Link}
              href={`/${owner}/${repoName}/tree/${branch}`}
              size="sm"
              variant="outline"
              borderColor="myGray.200"
            >
              {t('repo.browseFiles')}
            </Button>
            <Code colorScheme="gray" fontSize="sm" fontFamily="mono">
              {commitHash}
            </Code>
            <Button
              leftIcon={<Icon as={hasCopied ? FiCheck : FiCopy} />}
              size="sm"
              variant="outline"
              onClick={handleCopy}
            >
              {hasCopied ? t('repo.copied') : t('repo.copy')}
            </Button>
          </HStack>
        </HStack>

        {/* Commit Message */}
        <Text fontSize="sm" fontWeight="medium" color="myGray.900" whiteSpace="pre-wrap" mb={3}>
          {commit.message}
        </Text>

        {/* Author and Time */}
        <HStack justify="space-between" flexWrap="wrap" gap={4}>
          <HStack spacing={3}>
            <Avatar size="sm" name={commit.author} />
            <VStack spacing={0} align="start">
              <Text fontWeight="medium" color="myGray.900">
                {commit.author}
              </Text>
              <Text fontSize="sm" color="myGray.500">
                {commit.email}
              </Text>
            </VStack>
          </HStack>
          <Text fontSize="sm" color="myGray.600">
            {t('repo.committedAt')} {formatRelativeTime(commit.timestamp, t)}
          </Text>
        </HStack>
      </Box>

      {/* CI Checks */}
      {combinedStatus && combinedStatus.statuses && combinedStatus.statuses.length > 0 && (
        <Box borderWidth="1px" borderRadius="lg" bg="white" borderColor="myGray.200" overflow="hidden">
          <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
            <HStack spacing={2}>
              <Icon
                as={combinedStatus.state === 'success' ? FiCheckCircle : combinedStatus.state === 'pending' ? FiClock : FiXCircle}
                color={combinedStatus.state === 'success' ? 'green.500' : combinedStatus.state === 'pending' ? 'yellow.500' : 'red.500'}
                w={5}
                h={5}
              />
              <Text fontSize="sm" fontWeight="medium" color="myGray.900">
                {t('repo.ciChecks')}
              </Text>
              <Text fontSize="sm" color="myGray.600">
                {combinedStatus.state === 'success'
                  ? t('repo.allChecksPassed')
                  : combinedStatus.state === 'pending'
                  ? t('repo.checksPending')
                  : t('repo.someChecksFailed')}
              </Text>
            </HStack>
          </Box>
          <VStack spacing={0} align="stretch">
            {combinedStatus.statuses.map((status) => {
              const icon =
                status.state === 'success' ? FiCheckCircle :
                status.state === 'pending' ? FiClock :
                status.state === 'error' ? FiAlertCircle : FiXCircle
              const color =
                status.state === 'success' ? 'green.500' :
                status.state === 'pending' ? 'yellow.500' :
                status.state === 'error' ? 'orange.500' : 'red.500'
              const label =
                status.state === 'success' ? t('repo.statusSuccess') :
                status.state === 'pending' ? t('repo.statusPending') :
                status.state === 'error' ? t('repo.statusError') : t('repo.statusFailure')
              return (
                <HStack key={status.context} px={4} py={2} borderBottom="1px" borderColor="myGray.100" _last={{ borderBottom: 'none' }} spacing={3}>
                  <Icon as={icon} color={color} w={4} h={4} flexShrink={0} />
                  <VStack align="start" spacing={0} flex={1} minW={0}>
                    <Text fontSize="sm" fontWeight="medium" color="myGray.900" noOfLines={1}>
                      {status.context}
                    </Text>
                    {status.description && (
                      <Text fontSize="xs" color="myGray.500" noOfLines={1}>
                        {status.description}
                      </Text>
                    )}
                  </VStack>
                  <Badge fontSize="xs" colorScheme={status.state === 'success' ? 'green' : status.state === 'pending' ? 'yellow' : 'red'}>
                    {label}
                  </Badge>
                  {status.targetUrl && (
                    <Button as="a" href={status.targetUrl} target="_blank" size="xs" variant="ghost">
                      {t('common.viewAll')}
                    </Button>
                  )}
                </HStack>
              )
            })}
          </VStack>
        </Box>
      )}

      {/* Deployments (全链路追溯) */}
      {deployments.length > 0 && (
        <Box borderWidth="1px" borderRadius="lg" bg="white" borderColor="myGray.200" overflow="hidden">
          <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
            <HStack spacing={2}>
              <Icon as={FiPackage} color="purple.500" w={5} h={5} />
              <Text fontSize="sm" fontWeight="medium" color="myGray.900">
                {t('cicd.deployments')}
              </Text>
              <Badge colorScheme="gray" fontSize="xs">{deployments.length}</Badge>
            </HStack>
          </Box>
          <VStack spacing={0} align="stretch">
            {deployments.map((dep) => {
              const depColor =
                dep.status === 'success' ? 'green.500' :
                dep.status === 'failed' ? 'red.500' :
                dep.status === 'in_progress' ? 'blue.500' : 'myGray.400'
              const depIcon =
                dep.status === 'success' ? FiCheckCircle :
                dep.status === 'failed' ? FiXCircle :
                dep.status === 'in_progress' ? FiClock : FiAlertCircle
              const depLabel =
                dep.status === 'success' ? t('cicd.deploySuccess') :
                dep.status === 'failed' ? t('cicd.deployFailed') :
                dep.status === 'in_progress' ? t('cicd.deployInProgress') :
                t('cicd.deployCanceled')
              return (
                <HStack key={dep.id} px={4} py={2} borderBottom="1px" borderColor="myGray.100" _last={{ borderBottom: 'none' }} spacing={3}>
                  <Icon as={depIcon} color={depColor} w={4} h={4} flexShrink={0} />
                  <VStack align="start" spacing={0} flex={1} minW={0}>
                    <HStack spacing={2}>
                      <Text fontSize="sm" fontWeight="medium" color="myGray.900">
                        #{dep.id}
                      </Text>
                      <Text fontSize="xs" color="myGray.500">
                        {t('cicd.deployedBy')}: {dep.deployedBy}
                      </Text>
                      <Text fontSize="xs" color="myGray.500">
                        · {formatRelativeTime(dep.deployedAt, t)}
                      </Text>
                    </HStack>
                  </VStack>
                  <Badge fontSize="xs" colorScheme={dep.status === 'success' ? 'green' : dep.status === 'failed' ? 'red' : dep.status === 'in_progress' ? 'blue' : 'gray'}>
                    {depLabel}
                  </Badge>
                  {dep.pipelineId && (
                    <Button
                      as={Link}
                      href={`/${owner}/${repoName}/pipelines/${dep.pipelineId}`}
                      size="xs"
                      variant="ghost"
                      rightIcon={<Icon as={FiExternalLink} w={3} h={3} />}
                    >
                      Pipeline #{dep.pipelineId}
                    </Button>
                  )}
                </HStack>
              )
            })}
          </VStack>
        </Box>
      )}

      {/* File Changes Summary */}
      {files.length > 0 && (
        <HStack justify="space-between" px={2}>
          <Text fontSize="sm" color="myGray.600">
            {t('repo.filesChanged', { count: files.length })}
          </Text>
          <HStack spacing={2}>
            <Badge colorScheme="green" fontSize="sm">+{totalAdditions}</Badge>
            <Badge colorScheme="red" fontSize="sm">-{totalDeletions}</Badge>
          </HStack>
        </HStack>
      )}

      {/* Split View: File Tree + Diff */}
      {files.length > 0 ? (
        <Flex gap={4} align="start">
          {/* Left: File Tree */}
          <Box w="300px" flexShrink={0}>
            <FileTree
              files={files.map((f) => f.filename)}
              selectedFile={selectedFile}
              onSelectFile={setSelectedFile}
            />
          </Box>

          {/* Right: Diff View */}
          <Box flex={1} minW={0}>
            {fileDiff ? (
              <SplitDiffView
                filename={fileDiff.filename}
                status={fileDiff.status}
                additions={fileDiff.additions}
                deletions={fileDiff.deletions}
                hunks={fileDiff.hunks}
              />
            ) : (
              <Box borderWidth="1px" borderRadius="md" borderColor="myGray.200" p={8} textAlign="center" bg="white">
                <Icon as={FiFileText} w={12} h={12} color="myGray.400" />
                <Text color="myGray.600" mt={2}>
                  {t('repo.cannotDisplayDiff')}
                </Text>
              </Box>
            )}
          </Box>
        </Flex>
      ) : (
        <Box borderWidth="1px" borderRadius="lg" p={8} textAlign="center" bg="white" borderColor="myGray.200">
          <Icon as={FiFileText} w={12} h={12} color="myGray.400" />
          <Text color="myGray.600" mt={2}>
            {t('repo.noFileChanges')}
          </Text>
        </Box>
      )}
    </VStack>
  )
}

export default function CommitPage() {
  const { t } = useI18n()
  return (
    <Suspense fallback={<Text>{t('common.loading')}</Text>}>
      <CommitContent />
    </Suspense>
  )
}
