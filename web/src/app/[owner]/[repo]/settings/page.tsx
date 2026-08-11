'use client'

import {
  Text,
  Button,
  VStack,
  HStack,
  Box,
  Heading,
  FormControl,
  FormLabel,
  Input,
  Textarea,
  Switch,
  Select,
  Badge,
} from '@chakra-ui/react'
import { useParams, useRouter } from 'next/navigation'
import { useState, useEffect } from 'react'
import Link from 'next/link'
import { api, Branch } from '@/lib/api'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import { FiChevronRight } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import CollaboratorManager from './CollaboratorManager'
import DangerZone from './DangerZone'
import WebhookManager from './WebhookManager'
import DeployKeyManager from './DeployKeyManager'
import MirrorManager from './MirrorManager'
import PriorityManager from './PriorityManager'
import CustomFieldManager from './CustomFieldManager'
import LFSManager from './LFSManager'
import SecretManager from './SecretManager'

export default function SettingsPage() {
  const params = useParams()
  const router = useRouter()
  const toast = useGithubToast()
  const { repo, refreshData, userRole } = useRepo()
  const { t } = useI18n()

  const owner = params.owner as string
  const repoName = params.repo as string

  const [description, setDescription] = useState(repo?.description || '')
  const [isPrivate, setIsPrivate] = useState(repo?.isPrivate || false)
  const [defaultBranch, setDefaultBranch] = useState(repo?.defaultBranch || '')
  const [issuesOption, setIssuesOption] = useState(repo?.options?.issuesOption || 'PUBLIC')
  const [branches, setBranches] = useState<Branch[]>([])
  const [saving, setSaving] = useState(false)
  const [loadingBranches, setLoadingBranches] = useState(true)
  const { user: currentUser } = useCurrentUser()
  const [isArchived, setIsArchived] = useState(repo?.isArchived || false)
  const [isTemplate, setIsTemplate] = useState(repo?.isTemplate || false)

  useEffect(() => {
    if (repo) {
      setDescription(repo.description || '')
      setIsPrivate(repo.isPrivate || false)
      setDefaultBranch(repo.defaultBranch || '')
      setIssuesOption(repo.options?.issuesOption || 'PUBLIC')
      setIsArchived(repo.isArchived || false)
      setIsTemplate(repo.isTemplate || false)
    }
  }, [repo])

  useEffect(() => {
    const loadBranches = async () => {
      try {
        const branchesData = await api.listBranches(owner, repoName)
        setBranches(branchesData || [])
      } catch (error) {
        console.error('Failed to load branches:', error)
      } finally {
        setLoadingBranches(false)
      }
    }

    loadBranches()
  }, [owner, repoName])

  // 权限检查：只有 Owner 可以访问设置页面
  if (userRole !== 'owner') {
    return (
      <VStack spacing={6} align="stretch">
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200" p={8}>
          <VStack spacing={4}>
            <Heading size="md" color="myGray.900">{t('common.accessDenied')}</Heading>
            <Text color="myGray.600">{t('repo.settingsPermissionRequired')}</Text>
          </VStack>
        </Box>
      </VStack>
    )
  }

  const handleSave = async () => {
    if (!repo) return

    setSaving(true)
    try {
      await api.updateRepo(owner, repoName, {
        description,
        isPrivate,
        defaultBranch,
        options: {
          ...repo.options!,
          issuesOption: issuesOption,
        },
      })
      toast({
        title: t('repo.settingsSaved'),
        status: 'success',
        duration: 2000,
      })
      await refreshData()
    } catch (error: any) {
      toast({
        title: t('repo.saveFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSaving(false)
    }
  }

  if (!repo) {
    return <Text>{t('repo.repoNotExist')}</Text>
  }

  return (
    <VStack spacing={6} align="stretch">
      {/* 归档提示条 */}
      {isArchived && (
        <Box bg="yellow.50" borderWidth="1px" borderColor="yellow.200" borderRadius="md" px={4} py={3}>
          <HStack spacing={2}>
            <Badge colorScheme="yellow">{t('repo.archivedBadge')}</Badge>
            <Text fontSize="sm" color="yellow.700">
              {t('repo.archivedBanner')}
            </Text>
          </HStack>
        </Box>
      )}

      {/* 基本信息 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
        <Box px={6} pt={6} pb={2}>
          <Heading size="md">{t('repo.basicInfo')}</Heading>
          <Text fontSize="sm" color="myGray.600" mt={1}>
            {t('repo.basicInfoDesc')}
          </Text>
        </Box>
        <Box p={6}>
          <VStack spacing={5}>
            <FormControl>
              <FormLabel>{t('repo.repoName')}</FormLabel>
              <HStack spacing={2}>
                <Input value={repoName} isReadOnly bg="myGray.50" />
                {isArchived && <Badge colorScheme="yellow">{t('repo.archivedBadge')}</Badge>}
                {isTemplate && <Badge colorScheme="purple">{t('repo.templateBadge')}</Badge>}
              </HStack>
            </FormControl>

            <FormControl>
              <FormLabel>{t('repo.description')}</FormLabel>
              <Textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder={t('repo.descriptionPlaceholder')}
                rows={3}
              />
            </FormControl>

            <FormControl>
              <FormLabel>{t('repo.defaultBranch')}</FormLabel>
              {loadingBranches ? (
                <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
              ) : (
                <Select
                  value={defaultBranch}
                  onChange={(e) => setDefaultBranch(e.target.value)}
                  placeholder={t('repo.selectDefaultBranch')}
                >
                  {branches.map((branch) => (
                    <option key={branch.name} value={branch.name}>
                      {branch.name} {branch.isDefault && t('repo.currentDefault')}
                    </option>
                  ))}
                </Select>
              )}
            </FormControl>

            <FormControl display="flex" alignItems="center">
              <FormLabel mb={0}>{t('repo.privateRepo')}</FormLabel>
              <Switch
                isChecked={isPrivate}
                onChange={(e) => setIsPrivate(e.target.checked)}
                colorScheme="primary"
              />
              <Text fontSize="sm" color="myGray.600" ml={2}>
                {t('repo.privateRepoDesc')}
              </Text>
            </FormControl>

            <HStack justify="flex-end" w="full">
              <Button variant="primary" onClick={handleSave} isLoading={saving}>
                {t('repo.saveChanges')}
              </Button>
            </HStack>
          </VStack>
        </Box>
      </Box>

      {/* Issue 设置 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
        <Box px={6} pt={6} pb={2}>
          <Heading size="md">{t('repo.issuesSettings')}</Heading>
          <Text fontSize="sm" color="myGray.600" mt={1}>
            {t('repo.issuesSettingsDesc')}
          </Text>
        </Box>
        <Box p={6}>
          <VStack spacing={5}>
            <FormControl>
              <FormLabel>{t('repo.issuesOption')}</FormLabel>
              <Select
                value={issuesOption}
                onChange={(e) => setIssuesOption(e.target.value)}
              >
                <option value="ALL">{t('repo.issuesOptionAll')}</option>
                <option value="PUBLIC">{t('repo.issuesOptionPublic')}</option>
                <option value="PRIVATE">{t('repo.issuesOptionPrivate')}</option>
                <option value="DISABLE">{t('repo.issuesOptionDisable')}</option>
              </Select>
              <Text fontSize="sm" color="myGray.600" mt={2}>
                {issuesOption === 'ALL' && t('repo.issuesOptionAllDesc')}
                {issuesOption === 'PUBLIC' && t('repo.issuesOptionPublicDesc')}
                {issuesOption === 'PRIVATE' && t('repo.issuesOptionPrivateDesc')}
                {issuesOption === 'DISABLE' && t('repo.issuesOptionDisableDesc')}
              </Text>
            </FormControl>

            <HStack justify="flex-end" w="full">
              <Button variant="primary" onClick={handleSave} isLoading={saving}>
                {t('repo.saveChanges')}
              </Button>
            </HStack>
          </VStack>
        </Box>
      </Box>

      {/* 协作者管理 */}
      {currentUser && (
        <CollaboratorManager
          owner={owner}
          repoName={repoName}
          currentUser={currentUser}
        />
      )}

      {/* 标签管理 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
        <Box px={6} pt={6} pb={2}>
          <Heading size="md">{t('repo.labelsManagement')}</Heading>
          <Text fontSize="sm" color="myGray.600" mt={1}>
            {t('repo.labelsManagementDesc')}
          </Text>
        </Box>
        <Box p={6}>
          <HStack justify="space-between" align="center">
            <VStack align="start" spacing={0}>
              <Text fontWeight="medium">{t('repo.labels')}</Text>
              <Text fontSize="sm" color="myGray.600">
                {t('repo.labelsDesc')}
              </Text>
            </VStack>
            <Link href={`/${owner}/${repoName}/settings/labels`}>
              <Button rightIcon={<FiChevronRight />} variant="outline">
                {t('repo.manageLabels')}
              </Button>
            </Link>
          </HStack>
        </Box>
      </Box>

      {/* 优先级管理 */}
      <PriorityManager owner={owner} repoName={repoName} />

      {/* 自定义字段管理 */}
      <CustomFieldManager owner={owner} repoName={repoName} />

      {/* Webhook 管理 */}
      <WebhookManager owner={owner} repoName={repoName} />

      {/* 部署密钥管理 */}
      <DeployKeyManager owner={owner} repoName={repoName} />

      {/* CI/CD Secrets 管理 */}
      <SecretManager owner={owner} repoName={repoName} />

      {/* 镜像同步管理 */}
      <MirrorManager owner={owner} repoName={repoName} />

      {/* LFS 管理 */}
      <LFSManager owner={owner} repoName={repoName} />

      {/* 危险区域 */}
      <DangerZone
        owner={owner}
        repoName={repoName}
        isArchived={isArchived}
        isTemplate={isTemplate}
        onRenamed={(newName) => router.push(`/${owner}/${newName}/settings`)}
        onTransferred={(newOwner) => router.push(`/${newOwner}/${repoName}`)}
        onArchived={(archived) => {
          setIsArchived(archived)
          refreshData()
        }}
        onTemplateChanged={(templated) => {
          setIsTemplate(templated)
          refreshData()
        }}
        onDeleted={() => router.push(`/${owner}`)}
      />
    </VStack>
  )
}
