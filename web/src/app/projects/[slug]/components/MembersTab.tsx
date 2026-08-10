'use client'

import { useState, useEffect, useRef } from 'react'
import {
  Box,
  Button,
  Flex,
  Stack,
  Text,
  Badge,
  Avatar,
  HStack,
  VStack,
  IconButton,
  Icon,
  Input,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverBody,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  Spinner,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
  useToast,
} from '@chakra-ui/react'
import { FiPlus, FiTrash2, FiChevronDown } from 'react-icons/fi'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { UserSearchResult } from '@/lib/types'
import { roleLevel } from '@/lib/projectRole'

// 可分配的角色（owner 为固定角色，不可手动分配）
const EDITABLE_ROLES = ['admin', 'member', 'viewer'] as const

interface MembersTabProps {
  projectSlug: string
  members: any[]
  onMembersChange: (members: any[]) => void
  // 是否可管理成员（admin+，由父层从 ProjectContext.canManageProject 传入）
  canManage: boolean
}

export default function MembersTab({ projectSlug, members, onMembersChange, canManage }: MembersTabProps) {
  const { t } = useI18n()
  const toast = useToast()

  const getRoleColor = (role: string) => {
    switch (role) {
      case 'owner': return 'purple'
      case 'admin': return 'blue'
      case 'member': return 'gray'
      case 'viewer': return 'green'
      default: return 'gray'
    }
  }

  const getRoleLabel = (role: string) => {
    const key = `pms.role${role.charAt(0).toUpperCase() + role.slice(1)}`
    const translated = t(key)
    return translated === key ? role : translated
  }

  // Search state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [searchResults, setSearchResults] = useState<UserSearchResult[]>([])
  const [searchLoading, setSearchLoading] = useState(false)
  const [isPopoverOpen, setIsPopoverOpen] = useState(false)

  // Remove dialog state
  const [memberToRemove, setMemberToRemove] = useState<string | null>(null)
  const cancelRef = useRef(null as any)

  // Role update state
  const [updatingRoleUser, setUpdatingRoleUser] = useState<string | null>(null)

  useEffect(() => {
    if (!searchKeyword.trim()) {
      setSearchResults([])
      return
    }
    setSearchLoading(true)
    const timer = setTimeout(async () => {
      try {
        const result = await api.searchUsers(searchKeyword, 20)
        setSearchResults(result.users || [])
      } catch {
        setSearchResults([])
      } finally {
        setSearchLoading(false)
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [searchKeyword])

  const handleAddMember = async (userName: string) => {
    try {
      await api.addProjectMember(projectSlug, { userName, role: 'member' })
      setSearchKeyword('')
      setIsPopoverOpen(false)
      api.getProjectMembers(projectSlug).then((data) => onMembersChange(data || []))
    } catch (error: any) {
      toast({
        title: t('common.error'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleRemoveMember = async () => {
    if (!memberToRemove) return
    try {
      await api.removeProjectMember(projectSlug, memberToRemove)
      setMemberToRemove(null)
      api.getProjectMembers(projectSlug).then((data) => onMembersChange(data || []))
    } catch (error: any) {
      toast({
        title: t('common.error'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleRoleChange = async (userName: string, newRole: string) => {
    // 乐观更新
    const prevMembers = members
    onMembersChange(members.map(m => m.userName === userName ? { ...m, role: newRole } : m))
    setUpdatingRoleUser(userName)
    try {
      await api.updateProjectMemberRole(projectSlug, userName, newRole)
    } catch (error: any) {
      // 回滚
      onMembersChange(prevMembers)
      toast({
        title: t('common.error'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setUpdatingRoleUser(null)
    }
  }

  return (
    <>
      <Flex justify="space-between" align="center" mb={4}>
        <Text fontSize="sm" color="gray.500">{t('pms.teams')} ({members.length})</Text>
        {canManage && (
          <Popover
            placement="bottom-end"
            isOpen={isPopoverOpen}
            onOpen={() => setIsPopoverOpen(true)}
            onClose={() => { setIsPopoverOpen(false); setSearchKeyword('') }}
          >
            <PopoverTrigger>
              <Button size="sm" colorScheme="blue" leftIcon={<FiPlus />}>
                {t('pms.addMember')}
              </Button>
            </PopoverTrigger>
            <PopoverContent maxW="280px" w="280px">
              <PopoverBody p={2}>
                <VStack spacing={2} align="stretch">
                  <Input
                    size="sm"
                    placeholder={t('pms.searchUsers')}
                    value={searchKeyword}
                    onChange={(e) => setSearchKeyword(e.target.value)}
                    autoFocus
                  />
                  {searchLoading && (
                    <Text fontSize="xs" color="gray.500">{t('common.loading')}</Text>
                  )}
                  {!searchLoading && searchKeyword.trim() && (
                    <VStack spacing={0} align="stretch" maxH="240px" overflowY="auto">
                      {searchResults
                        .filter(u => !members.some(m => m.userName === u.userName))
                        .map(u => (
                          <Button
                            key={u.userName}
                            variant="ghost"
                            justifyContent="flex-start"
                            size="sm"
                            onClick={() => handleAddMember(u.userName)}
                          >
                            <HStack spacing={2}>
                              <Avatar size="xs" name={u.fullName || u.userName} src={u.image || undefined} />
                              <VStack spacing={0} align="start">
                                <Text fontSize="sm" fontWeight="medium">{u.userName}</Text>
                                {u.fullName && <Text fontSize="xs" color="gray.500">{u.fullName}</Text>}
                              </VStack>
                            </HStack>
                          </Button>
                        ))}
                      {searchResults
                        .filter(u => !members.some(m => m.userName === u.userName))
                        .length === 0 && (
                        <Text fontSize="xs" color="gray.500" p={2}>{t('pms.noUsersFound')}</Text>
                      )}
                    </VStack>
                  )}
                </VStack>
              </PopoverBody>
            </PopoverContent>
          </Popover>
        )}
      </Flex>
      {members && members.length > 0 ? (
        <Stack spacing={2}>
          {[...members]
            .sort((a, b) => roleLevel(b.role) - roleLevel(a.role))
            .map((member: any) => {
            const isOwner = member.role === 'owner'
            const isUpdating = updatingRoleUser === member.userName
            return (
            <Flex key={member.userName} p={3} bg="gray.50" borderRadius="md" align="center" justify="space-between">
              <Flex align="center" gap={3}>
                <Avatar size="sm" name={member.fullName || member.userName} src={member.image || undefined} />
                <Box>
                  <Text fontWeight="medium" fontSize="sm">{member.userName}</Text>
                  <Text fontSize="xs" color="gray.500">{getRoleLabel(member.role)}</Text>
                </Box>
              </Flex>
              <HStack spacing={2}>
                {canManage && !isOwner ? (
                  <Menu>
                    <MenuButton
                      as={Button}
                      size="xs"
                      variant="outline"
                      colorScheme={getRoleColor(member.role)}
                      rightIcon={isUpdating ? undefined : <Icon as={FiChevronDown} />}
                      isDisabled={isUpdating}
                      px={2}
                      minW="90px"
                      justifyContent="space-between"
                    >
                      {isUpdating ? <Spinner size="xs" /> : getRoleLabel(member.role)}
                    </MenuButton>
                    <MenuList minW="120px">
                      {EDITABLE_ROLES.map(role => (
                        <MenuItem
                          key={role}
                          onClick={() => handleRoleChange(member.userName, role)}
                          fontWeight={member.role === role ? 'bold' : 'normal'}
                          color={member.role === role ? `${getRoleColor(role)}.600` : undefined}
                        >
                          <Box w="8px" h="8px" borderRadius="full" bg={`${getRoleColor(role)}.500`} mr={2} display="inline-block" />
                          {getRoleLabel(role)}
                        </MenuItem>
                      ))}
                    </MenuList>
                  </Menu>
                ) : (
                  <Badge colorScheme={getRoleColor(member.role)} variant="outline">
                    {getRoleLabel(member.role)}
                  </Badge>
                )}
                {canManage && !isOwner && (
                  <IconButton
                    size="xs"
                    variant="ghost"
                    icon={<Icon as={FiTrash2} />}
                    onClick={() => setMemberToRemove(member.userName)}
                    aria-label="Remove member"
                    colorScheme="red"
                  />
                )}
              </HStack>
            </Flex>
            )
          })}
        </Stack>
      ) : (
        <Text color="gray.500">{t('pms.noMembers')}</Text>
      )}

      {/* Remove confirmation dialog */}
      <AlertDialog
        isOpen={memberToRemove !== null}
        leastDestructiveRef={cancelRef}
        onClose={() => setMemberToRemove(null)}
        blockScrollOnMount={false}
      >
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              {t('pms.removeMember')}
            </AlertDialogHeader>
            <AlertDialogBody>
              {t('pms.removeMemberConfirm').replace('{name}', memberToRemove || '')}
            </AlertDialogBody>
            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={() => setMemberToRemove(null)}>
                {t('common.cancel')}
              </Button>
              <Button colorScheme="red" onClick={handleRemoveMember} ml={3}>
                {t('common.delete')}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>
    </>
  )
}
