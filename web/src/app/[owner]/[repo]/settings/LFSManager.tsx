'use client'

import {
  Box,
  Heading,
  Text,
  VStack,
  HStack,
  Button,
  Icon,
  Spinner,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
} from '@chakra-ui/react'
import { useGithubToast } from '@/app/providers'
import { useEffect, useState, useCallback } from 'react'
import { api, LFSObject } from '@/lib/api'
import { FiTrash2 } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { formatRelativeTime } from '@/lib/time'

interface LFSManagerProps {
  owner: string
  repoName: string
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}

function truncateOid(oid: string): string {
  if (oid.length <= 16) return oid
  return `${oid.slice(0, 12)}…${oid.slice(-4)}`
}

export default function LFSManager({ owner, repoName }: LFSManagerProps) {
  const toast = useGithubToast()
  const { t } = useI18n()

  const [objects, setObjects] = useState<LFSObject[]>([])
  const [loading, setLoading] = useState(true)
  const [deletingOid, setDeletingOid] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.listLFSObjects(owner, repoName, 1, 100)
      setObjects(res.objects || [])
    } catch {
      setObjects([])
    } finally {
      setLoading(false)
    }
  }, [owner, repoName])

  useEffect(() => {
    loadData()
  }, [loadData])

  const handleConfirmDelete = async () => {
    if (!deletingOid) return
    setDeleting(true)
    try {
      await api.deleteLFSObject(owner, repoName, deletingOid)
      toast({ title: t('repo.lfsObjectDeleted'), status: 'success', duration: 2000 })
      setObjects(objects.filter(o => o.oid !== deletingOid))
    } catch (err: any) {
      toast({ title: t('repo.lfsObjectDeleteFailed'), description: err.message, status: 'error', duration: 3000 })
    } finally {
      setDeleting(false)
      setDeletingOid(null)
    }
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
      <Box px={6} pt={6} pb={2}>
        <VStack align="start" spacing={1}>
          <Heading size="md">{t('repo.lfsManagement')}</Heading>
          <Text fontSize="sm" color="myGray.600">
            {t('repo.lfsManagementDesc')}
          </Text>
        </VStack>
      </Box>
      <Box p={6}>
        {loading ? (
          <HStack justify="center" py={4}><Spinner size="sm" /></HStack>
        ) : objects.length === 0 ? (
          <Text fontSize="sm" color="myGray.500">{t('repo.noLfsObjects')}</Text>
        ) : (
          <Table size="sm" variant="simple">
            <Thead>
              <Tr>
                <Th>OID</Th>
                <Th>{t('repo.size')}</Th>
                <Th>{t('repo.createdAt')}</Th>
                <Th isNumeric>{t('common.actions')}</Th>
              </Tr>
            </Thead>
            <Tbody>
              {objects.map((obj) => (
                <Tr key={obj.oid}>
                  <Td>
                    <Text fontFamily="mono" fontSize="xs" color="myGray.700">
                      {truncateOid(obj.oid)}
                    </Text>
                  </Td>
                  <Td>
                    <Text fontSize="sm">{formatSize(obj.size)}</Text>
                  </Td>
                  <Td>
                    <Text fontSize="sm" color="myGray.600">
                      {formatRelativeTime(obj.createdAt, t)}
                    </Text>
                  </Td>
                  <Td isNumeric>
                    <Button
                      size="xs"
                      variant="ghost"
                      colorScheme="red"
                      leftIcon={<Icon as={FiTrash2} />}
                      onClick={() => setDeletingOid(obj.oid)}
                    >
                      {t('common.delete')}
                    </Button>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Box>

      <Modal isOpen={deletingOid !== null} onClose={() => setDeletingOid(null)}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('repo.deleteLfsObject')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text fontSize="sm" color="myGray.700">
              {t('repo.deleteLfsObjectConfirm')}
            </Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => setDeletingOid(null)}>
              {t('common.cancel')}
            </Button>
            <Button variant="solid" colorScheme="red" onClick={handleConfirmDelete} isLoading={deleting}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  )
}
