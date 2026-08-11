'use client'

import {
  Text,
  VStack,
  HStack,
  Box,
  Icon,
  Badge,
  Button,
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
  Select,
  useDisclosure,
  Spinner,
  Tooltip,
} from '@chakra-ui/react'
import { useParams } from 'next/navigation'
import { FiGitBranch, FiPlus, FiTrash2, FiEdit2, FiShield, FiShieldOff } from 'react-icons/fi'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { api } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useEffect, useState } from 'react'
import { useI18n } from '@/contexts/I18nContext'

export default function BranchesPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const { branches, refreshData, userRole } = useRepo()
  const toast = useGithubToast()
  const { t } = useI18n()
  const { isOpen, onOpen, onClose } = useDisclosure()

  // 权限判断
  const canCreateBranch = userRole === 'owner' || userRole === 'member'
  const canManageBranch = userRole === 'owner'

  const [branchName, setBranchName] = useState('')
  const [fromBranch, setFromBranch] = useState(branches.find(b => b.isDefault)?.name || branches[0]?.name || '')
  const [loading, setLoading] = useState(false)
  const [deletingBranch, setDeletingBranch] = useState<string | null>(null)
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()

  const [renamingBranch, setRenamingBranch] = useState<string | null>(null)
  const [newBranchName, setNewBranchName] = useState('')
  const { isOpen: isRenameOpen, onOpen: onRenameOpen, onClose: onRenameClose } = useDisclosure()

  const [protectedBranches, setProtectedBranches] = useState<Set<string>>(new Set())
  const [protectionLoadingBranch, setProtectionLoadingBranch] = useState<string | null>(null)

  useEffect(() => {
    api.listProtectedBranches(owner, repoName)
      .then(list => {
        setProtectedBranches(new Set(list.map(b => b.branchName)))
      })
      .catch(() => {
        // Silently fail (e.g., no permission)
      })
  }, [owner, repoName])

  const handleToggleProtection = async (branchName: string, protect: boolean) => {
    setProtectionLoadingBranch(branchName)
    try {
      if (protect) {
        await api.protectBranch(owner, repoName, branchName)
        setProtectedBranches(prev => new Set([...prev, branchName]))
        toast({ title: t('repo.branchProtectedSuccess'), status: 'success', duration: 2000 })
      } else {
        await api.unprotectBranch(owner, repoName, branchName)
        setProtectedBranches(prev => {
          const newSet = new Set(prev)
          newSet.delete(branchName)
          return newSet
        })
        toast({ title: t('repo.branchUnprotectedSuccess'), status: 'success', duration: 2000 })
      }
    } catch (error: any) {
      toast({ title: t('repo.branchProtectionFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setProtectionLoadingBranch(null)
    }
  }

  const handleCreateBranch = async () => {
    if (!branchName.trim()) {
      toast({ title: t('repo.enterBranchName'), status: 'warning', duration: 2000 })
      return
    }
    if (!fromBranch) {
      toast({ title: t('repo.selectSourceBranch'), status: 'warning', duration: 2000 })
      return
    }

    try {
      setLoading(true)
      await api.createBranch(owner, repoName, branchName, fromBranch)
      toast({ title: t('repo.branchCreatedSuccess'), status: 'success', duration: 2000 })
      onClose()
      setBranchName('')
      setFromBranch(branches.find(b => b.isDefault)?.name || branches[0]?.name || '')
      await refreshData()
    } catch (error: any) {
      toast({ title: t('repo.createBranchFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteBranch = async () => {
    if (!deletingBranch) return
    try {
      setLoading(true)
      await api.deleteBranch(owner, repoName, deletingBranch)
      toast({ title: t('repo.branchDeletedSuccess'), status: 'success', duration: 2000 })
      onDeleteClose()
      setDeletingBranch(null)
      await refreshData()
    } catch (error: any) {
      toast({ title: t('repo.deleteBranchFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setLoading(false)
    }
  }

  const openDeleteModal = (branchName: string) => {
    setDeletingBranch(branchName)
    onDeleteOpen()
  }

  const openRenameModal = (branchName: string) => {
    setRenamingBranch(branchName)
    setNewBranchName(branchName)
    onRenameOpen()
  }

  const handleRenameBranch = async () => {
    if (!newBranchName.trim()) {
      toast({ title: t('repo.enterBranchName'), status: 'warning', duration: 2000 })
      return
    }
    if (newBranchName.trim() === renamingBranch) {
      toast({ title: t('repo.newNameSameAsOld'), status: 'warning', duration: 2000 })
      return
    }
    try {
      setLoading(true)
      await api.renameBranch(owner, repoName, renamingBranch!, newBranchName.trim())
      toast({ title: t('repo.branchRenamedSuccess'), status: 'success', duration: 2000 })
      onRenameClose()
      setRenamingBranch(null)
      setNewBranchName('')
      await refreshData()
    } catch (error: any) {
      toast({ title: t('repo.renameBranchFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setLoading(false)
    }
  }

  return (
    <VStack spacing={4} align="stretch">
      <HStack justify="space-between">
        <Text fontSize="sm" color="myGray.600">
          {t('repo.branchesCountText', { count: branches.length })}
        </Text>
        {canCreateBranch && (
          <Button leftIcon={<Icon as={FiPlus} />} variant="primary" size="sm" onClick={onOpen}>
            {t('repo.newBranch')}
          </Button>
        )}
      </HStack>

      {branches.length === 0 ? (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
          <Box py={12}>
            <VStack spacing={4}>
              <Text color="myGray.500" fontSize="sm">{t('repo.noBranches')}</Text>
            </VStack>
          </Box>
        </Box>
      ) : (
        <VStack spacing={2} align="stretch">
          {branches.map((branch) => {
            const isProtected = protectedBranches.has(branch.name)
            return (
            <Box key={branch.name} borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
              <Box py={3} px={4}>
                <HStack spacing={3}>
                  <Icon as={FiGitBranch} w={4} h={4} color="myGray.500" />
                  <VStack align="start" spacing={0} flex={1}>
                    <HStack spacing={2}>
                      <Text fontSize="sm" fontWeight="medium" color="myGray.900">{branch.name}</Text>
                      {isProtected && (
                        <Badge colorScheme="yellow" variant="subtle">
                          <HStack spacing={1}>
                            <Icon as={FiShield} w={3} h={3} />
                            <Text>{t('repo.branchProtected')}</Text>
                          </HStack>
                        </Badge>
                      )}
                    </HStack>
                    {branch.commitId && (
                      <Text fontSize="xs" color="myGray.500">{branch.commitId.substring(0, 7)}</Text>
                    )}
                  </VStack>
                  <HStack spacing={1}>
                    {canManageBranch && (
                      <Tooltip label={isProtected ? t('repo.unprotectBranch') : t('repo.protectBranch')}>
                        <Button
                          leftIcon={<Icon as={isProtected ? FiShieldOff : FiShield} />}
                          variant="ghost"
                          size="sm"
                          colorScheme={isProtected ? 'orange' : 'yellow'}
                          onClick={() => handleToggleProtection(branch.name, !isProtected)}
                          isLoading={protectionLoadingBranch === branch.name}
                        >
                          {isProtected ? t('repo.unprotectBranch') : t('repo.protectBranch')}
                        </Button>
                      </Tooltip>
                    )}
                    {!branch.isDefault && canManageBranch && (
                      <>
                        <Button
                          leftIcon={<Icon as={FiEdit2} />}
                          variant="ghost"
                          size="sm"
                          colorScheme="blue"
                          onClick={() => openRenameModal(branch.name)}
                        >
                          {t('repo.renameBranch')}
                        </Button>
                        <Button
                          leftIcon={<Icon as={FiTrash2} />}
                          variant="ghost"
                          size="sm"
                          colorScheme="red"
                          onClick={() => openDeleteModal(branch.name)}
                        >
                          {t('common.delete')}
                        </Button>
                      </>
                    )}
                  </HStack>
                </HStack>
              </Box>
            </Box>
            )
          })}
        </VStack>
      )}

      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('repo.createBranchTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4} align="stretch">
              <FormControl isRequired>
                <FormLabel>{t('repo.branchNameLabel')}</FormLabel>
                <Input value={branchName} onChange={(e) => setBranchName(e.target.value)} placeholder={t('repo.branchNamePlaceholder')} />
              </FormControl>
              <FormControl isRequired>
                <FormLabel>{t('repo.sourceBranchLabel')}</FormLabel>
                <Select value={fromBranch} onChange={(e) => setFromBranch(e.target.value)} placeholder={t('repo.selectSourceBranchPlaceholder')}>
                  {branches.map((branch) => (
                    <option key={branch.name} value={branch.name}>{branch.name} {branch.isDefault && t('repo.defaultBranchSuffix')}</option>
                  ))}
                </Select>
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onClose}>{t('common.cancel')}</Button>
            <Button variant="primary" onClick={handleCreateBranch} isLoading={loading}>
              {loading ? <Spinner size="sm" /> : t('repo.createBranchButton')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      <Modal isOpen={isDeleteOpen} onClose={onDeleteClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('repo.deleteBranchTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text fontSize="sm" color="myGray.600">
              {t('repo.deleteBranchConfirm', { name: deletingBranch || '' })}
            </Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onDeleteClose}>{t('common.cancel')}</Button>
            <Button colorScheme="red" variant="solid" onClick={handleDeleteBranch} isLoading={loading}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      <Modal isOpen={isRenameOpen} onClose={onRenameClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('repo.renameBranchTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4} align="stretch">
              <FormControl>
                <FormLabel>{t('repo.oldNameLabel')}</FormLabel>
                <Input value={renamingBranch || ''} isReadOnly bg="myGray.50" />
              </FormControl>
              <FormControl isRequired>
                <FormLabel>{t('repo.newNameLabel')}</FormLabel>
                <Input
                  value={newBranchName}
                  onChange={(e) => setNewBranchName(e.target.value)}
                  placeholder={t('repo.newBranchNamePlaceholder')}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onRenameClose}>{t('common.cancel')}</Button>
            <Button variant="primary" onClick={handleRenameBranch} isLoading={loading}>
              {loading ? <Spinner size="sm" /> : t('repo.confirmRename')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </VStack>
  )
}
