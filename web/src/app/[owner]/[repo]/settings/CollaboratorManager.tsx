'use client'

import {
  Box,
  Heading,
  Text,
  VStack,
  HStack,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Select,
  Input,
  Button,
  IconButton,
  Avatar,
  Badge,
  Link,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverBody,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
  Collapse,
  useDisclosure,
  Spinner,
} from '@chakra-ui/react'
import { useState, useRef, useEffect, useCallback } from 'react'
import { api, Collaborator, User, UserSearchResult } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useGithubToast } from '@/app/providers'
import { FiTrash2, FiChevronDown, FiChevronUp } from 'react-icons/fi'

interface CollaboratorManagerProps {
  owner: string
  repoName: string
  currentUser: User
}

export default function CollaboratorManager({ owner, repoName, currentUser }: CollaboratorManagerProps) {
  const { t } = useI18n()
  const toast = useGithubToast()

  const [collaborators, setCollaborators] = useState<Collaborator[]>([])
  const [loading, setLoading] = useState(true)

  // Add collaborator state
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<UserSearchResult[]>([])
  const [searching, setSearching] = useState(false)
  const [newRole, setNewRole] = useState('member')
  const [adding, setAdding] = useState(false)
  const [showSuggestions, setShowSuggestions] = useState(false)

  // Remove confirm state
  const [removeTarget, setRemoveTarget] = useState<Collaborator | null>(null)
  const [removing, setRemoving] = useState(false)
  const cancelRef = useRef<any>(null)

  // Permission explainer
  const [showPermissions, setShowPermissions] = useState(false)

  const loadCollaborators = useCallback(async () => {
    try {
      const data = await api.listCollaborators(owner, repoName)
      setCollaborators(data || [])
    } catch (error) {
      console.error('Failed to load collaborators:', error)
    } finally {
      setLoading(false)
    }
  }, [owner, repoName])

  useEffect(() => {
    loadCollaborators()
  }, [loadCollaborators])

  // Debounced user search
  useEffect(() => {
    if (!searchQuery.trim()) {
      setSearchResults([])
      setShowSuggestions(false)
      return
    }

    setSearching(true)
    const timer = setTimeout(async () => {
      try {
        const res = await api.searchUsers(searchQuery.trim())
        const existingNames = new Set(collaborators.map(c => c.collaboratorName))
        existingNames.add(owner)
        existingNames.add(currentUser.userName)
        const filtered = (res.users || []).filter(u => !existingNames.has(u.userName))
        setSearchResults(filtered)
        setShowSuggestions(true)
      } catch (error) {
        console.error('User search failed:', error)
        setSearchResults([])
      } finally {
        setSearching(false)
      }
    }, 300)

    return () => clearTimeout(timer)
  }, [searchQuery, collaborators, owner, currentUser.userName])

  const handleAddCollaborator = async (userName: string) => {
    if (!userName.trim()) return

    setAdding(true)
    try {
      await api.addCollaborator(owner, repoName, userName.trim(), newRole)
      toast({ title: t('repo.collaboratorAdded'), status: 'success', duration: 2000 })
      setSearchQuery('')
      setSearchResults([])
      setShowSuggestions(false)
      await loadCollaborators()
    } catch (error: any) {
      toast({ title: t('repo.addFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setAdding(false)
    }
  }

  const handleRoleChange = async (collaboratorName: string, role: string) => {
    try {
      await api.updateCollaboratorRole(owner, repoName, collaboratorName, role)
      toast({ title: t('repo.roleUpdated'), status: 'success', duration: 2000 })
      setCollaborators(prev =>
        prev.map(c => c.collaboratorName === collaboratorName ? { ...c, role } : c)
      )
    } catch (error: any) {
      toast({ title: t('repo.updateFailed'), description: error.message, status: 'error', duration: 3000 })
    }
  }

  const handleRemoveConfirm = async () => {
    if (!removeTarget) return

    setRemoving(true)
    try {
      await api.removeCollaborator(owner, repoName, removeTarget.collaboratorName)
      toast({ title: t('repo.collaboratorRemoved'), status: 'success', duration: 2000 })
      setCollaborators(prev => prev.filter(c => c.collaboratorName !== removeTarget.collaboratorName))
    } catch (error: any) {
      toast({ title: t('repo.removeFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setRemoving(false)
      setRemoveTarget(null)
    }
  }

  const isCurrentUserOwner = currentUser.userName === owner

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
      <Box px={6} pt={6} pb={2}>
        <Heading size="md">{t('repo.collaborators')}</Heading>
        <Text fontSize="sm" color="myGray.600" mt={1}>
          {t('repo.collaboratorsDesc')}
        </Text>
      </Box>
      <Box p={6}>
        <VStack spacing={4} align="stretch">
          {/* Permission explainer */}
          <Box>
            <Button
              variant="ghost"
              size="sm"
              rightIcon={showPermissions ? <FiChevronUp /> : <FiChevronDown />}
              onClick={() => setShowPermissions(!showPermissions)}
              color="myGray.600"
            >
              {t('repo.permissionExplainer')}
            </Button>
            <Collapse in={showPermissions}>
              <Box mt={2} p={4} bg="myGray.50" borderRadius="md" borderWidth="1px" borderColor="myGray.200">
                <VStack align="start" spacing={2}>
                  <HStack spacing={2}>
                    <Badge colorScheme="purple">Admin</Badge>
                    <Text fontSize="sm" color="myGray.700">{t('repo.permissionAdmin')}</Text>
                  </HStack>
                  <HStack spacing={2}>
                    <Badge colorScheme="blue">Member</Badge>
                    <Text fontSize="sm" color="myGray.700">{t('repo.permissionMember')}</Text>
                  </HStack>
                  <HStack spacing={2}>
                    <Badge colorScheme="gray">Viewer</Badge>
                    <Text fontSize="sm" color="myGray.700">{t('repo.permissionViewer')}</Text>
                  </HStack>
                </VStack>
              </Box>
            </Collapse>
          </Box>

          {/* Collaborator table */}
          {loading ? (
            <HStack justify="center" py={4}>
              <Spinner size="sm" />
              <Text fontSize="sm" color="myGray.500">{t('common.loading')}</Text>
            </HStack>
          ) : (
            <Box borderWidth="1px" borderRadius="md" overflow="hidden" borderColor="myGray.200">
              <Table size="sm">
                <Thead bg="myGray.100">
                  <Tr>
                    <Th borderColor="myGray.200">{t('repo.username')}</Th>
                    <Th borderColor="myGray.200" width="140px">{t('repo.role')}</Th>
                    <Th borderColor="myGray.200" width="60px"></Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {/* Owner row */}
                  <Tr bg="myGray.50">
                    <Td borderColor="myGray.100">
                      <HStack spacing={3}>
                        <Avatar size="xs" name={currentUser.fullName || owner} src={currentUser.image || undefined} />
                        <Link href={`/${owner}`}>
                          <Text as="span" color="primary.600" fontWeight="medium" _hover={{ textDecoration: 'underline' }}>
                            {owner}
                          </Text>
                        </Link>
                        {owner === currentUser.userName && (
                          <Badge colorScheme="blue" fontSize="xs">{t('repo.youBadge')}</Badge>
                        )}
                        <Badge colorScheme="purple" fontSize="xs">{t('repo.ownerBadge')}</Badge>
                      </HStack>
                    </Td>
                    <Td borderColor="myGray.100">
                      <Text fontSize="sm" color="myGray.500">Owner</Text>
                    </Td>
                    <Td borderColor="myGray.100"></Td>
                  </Tr>
                  {/* Collaborator rows */}
                  {collaborators.map((c) => (
                    <Tr key={c.collaboratorName}>
                      <Td borderColor="myGray.100">
                        <HStack spacing={3}>
                          <Avatar size="xs" name={c.fullName || c.collaboratorName} src={c.avatarUrl || undefined} />
                          <Link href={`/${c.collaboratorName}`}>
                            <Text as="span" color="primary.600" fontWeight="medium" _hover={{ textDecoration: 'underline' }}>
                              {c.collaboratorName}
                            </Text>
                          </Link>
                          {c.fullName && c.fullName !== c.collaboratorName && (
                            <Text fontSize="sm" color="myGray.500">{c.fullName}</Text>
                          )}
                          {c.collaboratorName === currentUser.userName && (
                            <Badge colorScheme="blue" fontSize="xs">{t('repo.youBadge')}</Badge>
                          )}
                        </HStack>
                      </Td>
                      <Td borderColor="myGray.100">
                        <Select
                          size="sm"
                          value={c.role}
                          onChange={(e) => handleRoleChange(c.collaboratorName, e.target.value)}
                          isDisabled={c.collaboratorName === currentUser.userName && !isCurrentUserOwner}
                        >
                          <option value="admin">{t('repo.admin')}</option>
                          <option value="member">{t('repo.member')}</option>
                          <option value="viewer">{t('repo.viewer')}</option>
                        </Select>
                      </Td>
                      <Td borderColor="myGray.100">
                        <IconButton
                          aria-label={t('repo.removeCollaborator')}
                          icon={<FiTrash2 />}
                          size="sm"
                          variant="ghost"
                          colorScheme="red"
                          onClick={() => setRemoveTarget(c)}
                        />
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </Box>
          )}

          {/* Add collaborator with search suggestions */}
          <HStack spacing={2} align="flex-start">
            <Box position="relative" flex="1">
              <Popover
                isOpen={showSuggestions && searchResults.length > 0}
                onClose={() => setShowSuggestions(false)}
                placement="bottom-start"
                closeOnBlur
              >
                <PopoverTrigger>
                  <Input
                    placeholder={t('repo.searchUser')}
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    size="sm"
                  />
                </PopoverTrigger>
                <PopoverContent maxW="400px">
                  <PopoverBody p={0}>
                    {searching ? (
                      <HStack justify="center" py={3}>
                        <Spinner size="sm" />
                      </HStack>
                    ) : (
                      <VStack spacing={0} align="stretch">
                        {searchResults.map((user) => (
                          <Button
                            key={user.userName}
                            variant="ghost"
                            justifyContent="flex-start"
                            size="sm"
                            w="100%"
                            onClick={() => handleAddCollaborator(user.userName)}
                            isLoading={adding}
                          >
                            <HStack spacing={3} w="100%">
                              <Avatar size="xs" name={user.fullName || user.userName} src={user.image || undefined} />
                              <VStack align="start" spacing={0}>
                                <Text fontSize="sm" fontWeight="medium">{user.userName}</Text>
                                {user.fullName && user.fullName !== user.userName && (
                                  <Text fontSize="xs" color="myGray.500">{user.fullName}</Text>
                                )}
                              </VStack>
                            </HStack>
                          </Button>
                        ))}
                      </VStack>
                    )}
                  </PopoverBody>
                </PopoverContent>
              </Popover>
            </Box>
            <Select
              value={newRole}
              onChange={(e) => setNewRole(e.target.value)}
              size="sm"
              width="140px"
            >
              <option value="admin">{t('repo.admin')}</option>
              <option value="member">{t('repo.member')}</option>
              <option value="viewer">{t('repo.viewer')}</option>
            </Select>
            <Button
              variant="primary"
              size="sm"
              onClick={() => handleAddCollaborator(searchQuery)}
              isLoading={adding}
              isDisabled={!searchQuery.trim()}
            >
              {t('repo.add')}
            </Button>
          </HStack>
        </VStack>
      </Box>

      {/* Remove confirm dialog */}
      <AlertDialog
        isOpen={!!removeTarget}
        leastDestructiveRef={cancelRef}
        onClose={() => setRemoveTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>{t('repo.removeCollaborator')}</AlertDialogHeader>
          <AlertDialogBody>
            {t('repo.removeCollaboratorConfirm')} <strong>{removeTarget?.collaboratorName}</strong>?
          </AlertDialogBody>
          <AlertDialogFooter>
            <Button ref={cancelRef} onClick={() => setRemoveTarget(null)}>
              {t('common.cancel')}
            </Button>
            <Button colorScheme="red" onClick={handleRemoveConfirm} isLoading={removing} ml={3}>
              {t('common.remove')}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Box>
  )
}
