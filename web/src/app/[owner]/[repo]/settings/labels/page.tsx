'use client'

import {
  Box,
  VStack,
  HStack,
  Text,
  Button,
  Input,
  Badge,
  IconButton,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  FormControl,
  FormLabel,
  useDisclosure,
  SimpleGrid,
} from '@chakra-ui/react'
import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import { FiEdit2, FiTrash2, FiPlus } from 'react-icons/fi'
import { api, Label } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { ConfirmDialog } from '@/components/ConfirmDialog'

// 判断颜色亮度，决定文字颜色
function getTextColor(bgColor: string): string {
  const hex = bgColor.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  const brightness = (r * 299 + g * 587 + b * 114) / 1000
  return brightness > 128 ? 'myGray.800' : 'white'
}

// 预设颜色
const PRESET_COLORS = [
  'e99695', 'f9d0c4', 'fef2c0', 'c2e0c6', 'bfdadc',
  'c5def5', 'b3d7ff', '20b1ea', '006b75', '5319e7',
  'e99695', 'd93f0b', '0e8a16', '006b75', '1d76db',
  '5319e7', 'b60205', 'd4c5f9', 'fbca04', '006b75',
]

export default function LabelsPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const toast = useGithubToast()
  const { isOpen, onOpen, onClose } = useDisclosure()
  const { t } = useI18n()
  const { userRole } = useRepo()

  const [labels, setLabels] = useState<Label[]>([])
  const [loading, setLoading] = useState(true)
  const [editingLabel, setEditingLabel] = useState<Label | null>(null)
  const [newLabelName, setNewLabelName] = useState('')
  const [newLabelColor, setNewLabelColor] = useState('006b75')
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [confirmAction, setConfirmAction] = useState<(() => void) | null>(null)

  // Developer 及以上权限可以管理标签
  const canManageLabels = userRole === 'owner' || userRole === 'member'

  const loadLabels = async () => {
    try {
      setLoading(true)
      const data = await api.listLabels(owner, repoName)
      setLabels(data || [])
    } catch (error: any) {
      toast({
        title: t('repo.loadLabelsFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadLabels()
  }, [owner, repoName])

  const handleCreateLabel = async () => {
    if (!newLabelName.trim()) {
      toast({
        title: t('repo.enterLabelName'),
        status: 'warning',
        duration: 2000,
      })
      return
    }

    try {
      await api.createLabel(owner, repoName, newLabelName, newLabelColor)
      toast({
        title: t('repo.labelCreated'),
        status: 'success',
        duration: 2000,
      })
      setNewLabelName('')
      setNewLabelColor('006b75')
      onClose()
      loadLabels()
    } catch (error: any) {
      toast({
        title: t('repo.createLabelFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleUpdateLabel = async () => {
    if (!editingLabel || !newLabelName.trim()) {
      toast({
        title: t('repo.enterLabelName'),
        status: 'warning',
        duration: 2000,
      })
      return
    }

    try {
      await api.updateLabel(owner, repoName, editingLabel.labelId, {
        labelName: newLabelName,
        color: newLabelColor,
      })
      toast({
        title: t('repo.labelUpdated'),
        status: 'success',
        duration: 2000,
      })
      setEditingLabel(null)
      setNewLabelName('')
      setNewLabelColor('006b75')
      onClose()
      loadLabels()
    } catch (error: any) {
      toast({
        title: t('repo.updateLabelFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleDeleteLabel = (id: number) => {
    setConfirmAction(() => async () => {
      try {
        await api.deleteLabel(owner, repoName, id)
        toast({
          title: t('repo.labelDeleted'),
          status: 'success',
          duration: 2000,
        })
        loadLabels()
      } catch (error: any) {
        toast({
          title: t('repo.deleteLabelFailed'),
          description: error.message,
          status: 'error',
          duration: 3000,
        })
      }
    })
    setConfirmOpen(true)
  }

  const handleConfirm = () => {
    confirmAction?.()
    setConfirmOpen(false)
    setConfirmAction(null)
  }

  const openCreateModal = () => {
    setEditingLabel(null)
    setNewLabelName('')
    setNewLabelColor('006b75')
    onOpen()
  }

  const openEditModal = (label: Label) => {
    setEditingLabel(label)
    setNewLabelName(label.labelName)
    setNewLabelColor(label.color)
    onOpen()
  }

  return (
    <Box>
      <VStack spacing={6} align="stretch">
        <HStack justify="space-between">
          <Text fontSize="lg" fontWeight="bold">
            {t('repo.labelsManagementTitle')}
          </Text>
          {canManageLabels && (
            <Button leftIcon={<FiPlus />} colorScheme="blue" size="sm" onClick={openCreateModal}>
              {t('repo.newLabel')}
            </Button>
          )}
        </HStack>

        {loading ? (
          <Text>{t('common.loading')}</Text>
        ) : labels.length === 0 ? (
          <Box textAlign="center" py={10}>
            <Text color="myGray.500">{t('repo.noLabels')}</Text>
          </Box>
        ) : (
          <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} spacing={4}>
            {labels.map((label) => (
              <Box
                key={label.labelId}
                borderWidth="1px"
                borderRadius="lg"
                p={4}
                _hover={{ shadow: 'md' }}
                transition="all 0.2s"
              >
                <HStack justify="space-between" align="center">
                  <Badge
                    bg={`#${label.color}`}
                    color={getTextColor(label.color)}
                    px={3}
                    py={1}
                    borderRadius="md"
                    fontSize="sm"
                  >
                    {label.labelName}
                  </Badge>
                  <HStack spacing={1}>
                    <IconButton
                      aria-label={t('common.edit')}
                      icon={<FiEdit2 />}
                      size="sm"
                      variant="ghost"
                      onClick={() => openEditModal(label)}
                    />
                    <IconButton
                      aria-label={t('common.delete')}
                      icon={<FiTrash2 />}
                      size="sm"
                      variant="ghost"
                      colorScheme="red"
                      onClick={() => handleDeleteLabel(label.labelId)}
                    />
                  </HStack>
                </HStack>
                <Text fontSize="xs" color="myGray.500" mt={2}>
                  #{label.color}
                </Text>
              </Box>
            ))}
          </SimpleGrid>
        )}
      </VStack>

      <ConfirmDialog
        isOpen={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={handleConfirm}
        title={t('common.confirm')}
        message={t('repo.confirmDeleteLabel')}
      />

      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{editingLabel ? t('repo.editLabel') : t('repo.newLabel')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl>
                <FormLabel>{t('repo.labelName')}</FormLabel>
                <Input
                  value={newLabelName}
                  onChange={(e) => setNewLabelName(e.target.value)}
                  placeholder={t('repo.labelNamePlaceholder')}
                />
              </FormControl>

              <FormControl>
                <FormLabel>{t('repo.labelColor')}</FormLabel>
                <VStack align="stretch" spacing={3}>
                  <HStack>
                    <Badge
                      bg={`#${newLabelColor}`}
                      color={getTextColor(newLabelColor)}
                      px={3}
                      py={1}
                      borderRadius="md"
                    >
                      {t('repo.preview')}
                    </Badge>
                    <Text fontSize="sm" color="myGray.500">
                      #{newLabelColor}
                    </Text>
                  </HStack>
                  <SimpleGrid columns={10} spacing={2}>
                    {PRESET_COLORS.map((color) => (
                      <Box
                        key={color}
                        w="30px"
                        h="30px"
                        bg={`#${color}`}
                        borderRadius="md"
                        cursor="pointer"
                        border={newLabelColor === color ? '2px solid' : '1px solid'}
                        borderColor={newLabelColor === color ? 'blue.500' : 'myGray.200'}
                        onClick={() => setNewLabelColor(color)}
                        _hover={{ transform: 'scale(1.1)' }}
                        transition="all 0.2s"
                      />
                    ))}
                  </SimpleGrid>
                  <HStack>
                    <Text fontSize="sm" color="myGray.500">
                      {t('repo.custom')}:
                    </Text>
                    <Input
                      size="sm"
                      w="100px"
                      value={newLabelColor}
                      onChange={(e) => setNewLabelColor(e.target.value.replace('#', ''))}
                      placeholder="006b75"
                    />
                  </HStack>
                </VStack>
              </FormControl>
            </VStack>
          </ModalBody>

          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button
              colorScheme="blue"
              onClick={editingLabel ? handleUpdateLabel : handleCreateLabel}
            >
              {editingLabel ? t('common.update') : t('common.create')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      <ConfirmDialog
        isOpen={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={handleConfirm}
        title={t('common.confirm')}
        message={t('repo.confirmDeleteLabel')}
      />
    </Box>
  )
}
