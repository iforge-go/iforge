'use client'

import type React from 'react'
import {
  Avatar,
  Button,
  HStack,
  IconButton,
  Icon,
  Input,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverBody,
  Text,
  VStack,
} from '@chakra-ui/react'
import { FiPlus, FiX } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { UserSearchResult } from '@/lib/types'

interface TaskAssigneesProps {
  assignees: any[]
  setAssignees: React.Dispatch<React.SetStateAction<any[]>>
  searchKeyword: string
  setSearchKeyword: React.Dispatch<React.SetStateAction<string>>
  searchResults: UserSearchResult[]
  setSearchResults: React.Dispatch<React.SetStateAction<UserSearchResult[]>>
  searchLoading: boolean
  setSearchLoading: React.Dispatch<React.SetStateAction<boolean>>
  isPopoverOpen: boolean
  setIsPopoverOpen: React.Dispatch<React.SetStateAction<boolean>>
}

export default function TaskAssignees({
  assignees,
  setAssignees,
  searchKeyword,
  setSearchKeyword,
  searchResults,
  setSearchResults: _setSearchResults,
  searchLoading,
  setSearchLoading: _setSearchLoading,
  isPopoverOpen,
  setIsPopoverOpen,
}: TaskAssigneesProps) {
  const { t } = useI18n()

  return (
    <HStack spacing={2} flexWrap="wrap">
      {assignees.map((a) => (
        <HStack key={a.userName} spacing={1} bg="myGray.100" borderRadius="full" px={2} py={1}>
          <Avatar size="xs" name={a.fullName || a.userName} src={a.image || undefined} />
          <Text fontSize="xs" color="myGray.700">{a.userName}</Text>
          <IconButton
            size="xs"
            variant="ghost"
            icon={<Icon as={FiX} />}
            onClick={() => setAssignees((prev) => prev.filter((x) => x.userName !== a.userName))}
            aria-label="Remove assignee"
            minW="16px"
            w="16px"
            h="16px"
          />
        </HStack>
      ))}
      <Popover placement="bottom-end" isOpen={isPopoverOpen} onOpen={() => setIsPopoverOpen(true)} onClose={() => setIsPopoverOpen(false)}>
        <PopoverTrigger>
          <Button size="xs" variant="outline" leftIcon={<Icon as={FiPlus} />}>
            {t('common.add')}
          </Button>
        </PopoverTrigger>
        <PopoverContent maxW="220px" w="220px">
          <PopoverBody p={2}>
            <VStack spacing={2} align="stretch">
              <Input
                size="sm"
                placeholder={t('pms.searchUsers')}
                value={searchKeyword}
                onChange={(e) => setSearchKeyword(e.target.value)}
              />
              {searchLoading && (
                <Text fontSize="xs" color="myGray.500">{t('common.loading')}</Text>
              )}
              {!searchLoading && searchKeyword.trim() && (
                <VStack spacing={0} align="stretch" maxH="200px" overflowY="auto">
                  {searchResults
                    .filter(u => !assignees.some(a => a.userName === u.userName))
                    .map(u => (
                      <Button
                        key={u.userName}
                        variant="ghost"
                        justifyContent="flex-start"
                        size="sm"
                        onClick={() => {
                          setAssignees((prev) => [...prev, {
                            userName: u.userName,
                            fullName: u.fullName || u.userName,
                            image: u.image || null,
                          }])
                          setSearchKeyword('')
                          setIsPopoverOpen(false)
                        }}
                      >
                        <HStack spacing={2}>
                          <Avatar size="xs" name={u.fullName || u.userName} src={u.image || undefined} />
                          <VStack spacing={0} align="start">
                            <Text fontSize="sm" fontWeight="medium">{u.userName}</Text>
                            {u.fullName && <Text fontSize="xs" color="myGray.500">{u.fullName}</Text>}
                          </VStack>
                        </HStack>
                      </Button>
                    ))}
                  {searchResults
                    .filter(u => !assignees.some(a => a.userName === u.userName))
                    .length === 0 && (
                    <Text fontSize="xs" color="myGray.500" p={2}>{t('pms.noUsersFound')}</Text>
                  )}
                </VStack>
              )}
            </VStack>
          </PopoverBody>
        </PopoverContent>
      </Popover>
    </HStack>
  )
}
