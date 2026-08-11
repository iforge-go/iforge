'use client'

import {
  Box,
  Heading,
  Text,
  VStack,
  HStack,
  Button,
  Input,
  Switch,
  FormControl,
  FormLabel,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
} from '@chakra-ui/react'
import { useState, useRef } from 'react'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useGithubToast } from '@/app/providers'

interface DangerZoneProps {
  owner: string
  repoName: string
  isArchived: boolean
  isTemplate: boolean
  onRenamed: (newName: string) => void
  onTransferred: (newOwner: string) => void
  onArchived: (isArchived: boolean) => void
  onTemplateChanged: (isTemplate: boolean) => void
  onDeleted: () => void
}

type DialogType = 'rename' | 'transfer' | 'archive' | 'unarchive' | 'delete' | null

export default function DangerZone({
  owner,
  repoName,
  isArchived,
  isTemplate,
  onRenamed,
  onTransferred,
  onArchived,
  onTemplateChanged,
  onDeleted,
}: DangerZoneProps) {
  const { t } = useI18n()
  const toast = useGithubToast()
  const cancelRef = useRef<any>(null)

  const [dialogType, setDialogType] = useState<DialogType>(null)
  const [submitting, setSubmitting] = useState(false)

  // Rename state
  const [newRepoName, setNewRepoName] = useState('')

  // Transfer state
  const [newOwner, setNewOwner] = useState('')
  const [transferConfirm, setTransferConfirm] = useState('')

  // Delete confirm
  const [deleteConfirm, setDeleteConfirm] = useState('')

  const expectedConfirmText = `${owner}/${repoName}`
  const canTransfer = newOwner.trim() && transferConfirm === expectedConfirmText
  const canDelete = deleteConfirm === expectedConfirmText
  const canRename = newRepoName.trim() && newRepoName !== repoName && /^[a-zA-Z0-9._-]+$/.test(newRepoName.trim())

  const closeDialog = () => {
    setDialogType(null)
    setNewRepoName('')
    setNewOwner('')
    setTransferConfirm('')
    setDeleteConfirm('')
  }

  const handleRename = async () => {
    if (!canRename) return
    setSubmitting(true)
    try {
      await api.renameRepo(owner, repoName, newRepoName.trim())
      toast({ title: t('repo.renameSuccess'), status: 'success', duration: 2000 })
      onRenamed(newRepoName.trim())
      closeDialog()
    } catch (error: any) {
      toast({ title: t('repo.renameFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setSubmitting(false)
    }
  }

  const handleTransfer = async () => {
    if (!canTransfer) return
    setSubmitting(true)
    try {
      await api.transferRepo(owner, repoName, newOwner.trim())
      toast({ title: t('repo.transferSuccess'), status: 'success', duration: 2000 })
      onTransferred(newOwner.trim())
      closeDialog()
    } catch (error: any) {
      toast({ title: t('repo.transferFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setSubmitting(false)
    }
  }

  const handleArchive = async (archive: boolean) => {
    setSubmitting(true)
    try {
      await api.archiveRepo(owner, repoName, archive)
      toast({ title: t('repo.archiveSuccess'), status: 'success', duration: 2000 })
      onArchived(archive)
      closeDialog()
    } catch (error: any) {
      toast({ title: t('repo.archiveFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setSubmitting(false)
    }
  }

  const handleTemplateToggle = async (checked: boolean) => {
    try {
      await api.setTemplateRepo(owner, repoName, checked)
      toast({ title: t('repo.templateSuccess'), status: 'success', duration: 2000 })
      onTemplateChanged(checked)
    } catch (error: any) {
      toast({ title: t('repo.updateFailed'), description: error.message, status: 'error', duration: 3000 })
    }
  }

  const handleDelete = async () => {
    if (!canDelete) return
    setSubmitting(true)
    try {
      await api.deleteRepo(owner, repoName)
      toast({ title: t('repo.repoDeleted'), status: 'success', duration: 2000 })
      onDeleted()
      closeDialog()
    } catch (error: any) {
      toast({ title: t('repo.deleteFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setSubmitting(false)
    }
  }

  const getDialogTitle = () => {
    switch (dialogType) {
      case 'rename': return t('repo.renameRepo')
      case 'transfer': return t('repo.transferRepo')
      case 'archive': return t('repo.archiveRepo')
      case 'unarchive': return t('repo.unarchiveRepo')
      case 'delete': return t('repo.deleteRepo')
      default: return ''
    }
  }

  return (
    <>
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="red.200">
        <Box px={6} pt={6} pb={2}>
          <Heading size="md" color="red.600">{t('repo.dangerZone')}</Heading>
          <Text fontSize="sm" color="myGray.600" mt={1}>
            {t('repo.dangerZoneDesc')}
          </Text>
        </Box>
        <Box p={6}>
          <VStack spacing={0} align="stretch" divider={<Box borderBottom="1px solid" borderColor="myGray.100" />}>
            {/* Template repository */}
            <HStack justify="space-between" align="center" pb={4}>
              <VStack align="start" spacing={0}>
                <Text fontWeight="medium">{t('repo.templateRepo')}</Text>
                <Text fontSize="sm" color="myGray.600">
                  {t('repo.templateRepoDesc')}
                </Text>
              </VStack>
              <Switch
                isChecked={isTemplate}
                onChange={(e) => handleTemplateToggle(e.target.checked)}
                colorScheme="primary"
              />
            </HStack>

            {/* Archive / Unarchive */}
            <HStack justify="space-between" align="center" py={4}>
              <VStack align="start" spacing={0}>
                <Text fontWeight="medium">
                  {isArchived ? t('repo.unarchiveRepo') : t('repo.archiveRepo')}
                </Text>
                <Text fontSize="sm" color="myGray.600">
                  {isArchived ? t('repo.unarchiveRepoDesc') : t('repo.archiveRepoDesc')}
                </Text>
              </VStack>
              <Button
                colorScheme="red"
                variant="outline"
                onClick={() => setDialogType(isArchived ? 'unarchive' : 'archive')}
              >
                {isArchived ? t('repo.unarchiveRepo') : t('repo.archiveRepo')}
              </Button>
            </HStack>

            {/* Rename */}
            <HStack justify="space-between" align="center" py={4}>
              <VStack align="start" spacing={0}>
                <Text fontWeight="medium">{t('repo.renameRepo')}</Text>
                <Text fontSize="sm" color="myGray.600">
                  {t('repo.renameRepoDesc')}
                </Text>
              </VStack>
              <Button colorScheme="red" variant="outline" onClick={() => setDialogType('rename')}>
                {t('repo.renameRepo')}
              </Button>
            </HStack>

            {/* Transfer */}
            <HStack justify="space-between" align="center" py={4}>
              <VStack align="start" spacing={0}>
                <Text fontWeight="medium">{t('repo.transferRepo')}</Text>
                <Text fontSize="sm" color="myGray.600">
                  {t('repo.transferRepoDesc')}
                </Text>
              </VStack>
              <Button colorScheme="red" variant="outline" onClick={() => setDialogType('transfer')}>
                {t('repo.transferRepo')}
              </Button>
            </HStack>

            {/* Delete */}
            <HStack justify="space-between" align="center" pt={4}>
              <VStack align="start" spacing={0}>
                <Text fontWeight="medium">{t('repo.deleteThisRepo')}</Text>
                <Text fontSize="sm" color="myGray.600">
                  {t('repo.deleteThisRepoDesc')}
                </Text>
              </VStack>
              <Button colorScheme="red" variant="solid" onClick={() => setDialogType('delete')}>
                {t('repo.deleteRepo')}
              </Button>
            </HStack>
          </VStack>
        </Box>
      </Box>

      {/* Unified AlertDialog */}
      <AlertDialog
        isOpen={dialogType !== null}
        leastDestructiveRef={cancelRef}
        onClose={closeDialog}
      >
        <AlertDialogContent>
          <AlertDialogHeader>{getDialogTitle()}</AlertDialogHeader>
          <AlertDialogBody>
            {dialogType === 'rename' && (
              <VStack spacing={3} align="stretch">
                <Text fontSize="sm" color="myGray.600">
                  {t('repo.renameRepoDesc')}
                </Text>
                <FormControl>
                  <FormLabel>{t('repo.renameNewName')}</FormLabel>
                  <Input
                    value={newRepoName}
                    onChange={(e) => setNewRepoName(e.target.value)}
                    placeholder={repoName}
                    autoFocus
                  />
                </FormControl>
              </VStack>
            )}

            {dialogType === 'transfer' && (
              <VStack spacing={3} align="stretch">
                <Text fontSize="sm" color="myGray.600">
                  {t('repo.transferRepoDesc')}
                </Text>
                <FormControl>
                  <FormLabel>{t('repo.transferNewOwner')}</FormLabel>
                  <Input
                    value={newOwner}
                    onChange={(e) => setNewOwner(e.target.value)}
                    placeholder="new-owner"
                    autoFocus
                  />
                </FormControl>
                <Text fontSize="sm" color="red.600">
                  {t('repo.transferWarning')}
                </Text>
                <FormControl>
                  <FormLabel>{t('repo.transferConfirmPrompt', { repo: expectedConfirmText })}</FormLabel>
                  <Input
                    value={transferConfirm}
                    onChange={(e) => setTransferConfirm(e.target.value)}
                    placeholder={expectedConfirmText}
                  />
                </FormControl>
              </VStack>
            )}

            {(dialogType === 'archive' || dialogType === 'unarchive') && (
              <Text>
                {dialogType === 'archive'
                  ? t('repo.archiveConfirm')
                  : t('repo.unarchiveConfirm')}
              </Text>
            )}

            {dialogType === 'delete' && (
              <VStack spacing={3} align="stretch">
                <Text>
                  {t('repo.confirmDeleteRepo')} <strong>{expectedConfirmText}</strong>{t('repo.confirmDeleteRepoSuffix')}
                </Text>
                <FormControl>
                  <FormLabel>{t('repo.deleteConfirmPrompt', { repo: expectedConfirmText })}</FormLabel>
                  <Input
                    value={deleteConfirm}
                    onChange={(e) => setDeleteConfirm(e.target.value)}
                    placeholder={expectedConfirmText}
                    autoFocus
                  />
                </FormControl>
              </VStack>
            )}
          </AlertDialogBody>
          <AlertDialogFooter>
            <Button ref={cancelRef} onClick={closeDialog}>
              {t('common.cancel')}
            </Button>
            {dialogType === 'rename' && (
              <Button colorScheme="red" onClick={handleRename} isLoading={submitting} ml={3} isDisabled={!canRename}>
                {t('repo.renameRepo')}
              </Button>
            )}
            {dialogType === 'transfer' && (
              <Button colorScheme="red" onClick={handleTransfer} isLoading={submitting} ml={3} isDisabled={!canTransfer}>
                {t('repo.transferRepo')}
              </Button>
            )}
            {dialogType === 'archive' && (
              <Button colorScheme="red" onClick={() => handleArchive(true)} isLoading={submitting} ml={3}>
                {t('repo.archiveRepo')}
              </Button>
            )}
            {dialogType === 'unarchive' && (
              <Button colorScheme="primary" onClick={() => handleArchive(false)} isLoading={submitting} ml={3}>
                {t('repo.unarchiveRepo')}
              </Button>
            )}
            {dialogType === 'delete' && (
              <Button colorScheme="red" onClick={handleDelete} isLoading={submitting} ml={3} isDisabled={!canDelete}>
                {t('repo.delete')}
              </Button>
            )}
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
