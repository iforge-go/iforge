'use client'

import { useState, useEffect } from 'react'
import {
  Avatar,
  Box,
  Button,
  HStack,
  Input,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverBody,
  Text,
  VStack,
  Icon,
} from '@chakra-ui/react'
import { FiChevronDown, FiX } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { api } from '@/lib/api'
import { UserSearchResult } from '@/lib/types'

interface UserSelectProps {
  value: string | null
  onChange: (user: { userName: string; fullName?: string; image?: string | null } | null) => void
  placeholder?: string
  size?: 'sm' | 'md'
  clearable?: boolean
  disabled?: boolean
  /** 触发按钮宽度，默认 "full" 占满父容器；详情页紧凑场景可传 "auto" */
  buttonWidth?: string
}

/**
 * 单选用户搜索选择器。
 * 基于 Popover + debounce 搜索，用于 story/issue 等单选 assignee 场景。
 * 多选 assignee 场景请用 TaskAssignees。
 */
export default function UserSelect({
  value,
  onChange,
  placeholder,
  size = 'md',
  clearable = true,
  disabled = false,
  buttonWidth = 'full',
}: UserSelectProps) {
  const { t } = useI18n()
  const [isOpen, setIsOpen] = useState(false)
  const [keyword, setKeyword] = useState('')
  const [results, setResults] = useState<UserSearchResult[]>([])
  const [loading, setLoading] = useState(false)

  // debounce 搜索
  useEffect(() => {
    if (!keyword.trim()) {
      setResults([])
      return
    }
    setLoading(true)
    const timer = setTimeout(async () => {
      try {
        const r = await api.searchUsers(keyword, 20)
        setResults(r.users || [])
      } catch {
        setResults([])
      } finally {
        setLoading(false)
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [keyword])

  const handleClose = () => {
    setIsOpen(false)
    setKeyword('')
  }

  return (
    <Popover placement="bottom-start" isOpen={isOpen} onOpen={() => setIsOpen(true)} onClose={handleClose}>
      <PopoverTrigger>
        <Button
          size={size}
          variant="outline"
          w={buttonWidth}
          justifyContent="flex-start"
          fontWeight="normal"
          isDisabled={disabled}
          rightIcon={value && clearable ? undefined : <Icon as={FiChevronDown} />}
        >
          {value ? (
            <HStack spacing={2} flex={1} justify="flex-start" w="full">
              <Avatar size="xs" name={value} />
              <Text fontSize={size === 'sm' ? 'sm' : 'md'}>{value}</Text>
              {clearable && (
                <Box
                  as="span"
                  role="button"
                  tabIndex={0}
                  aria-label="Clear"
                  ml="auto"
                  display="inline-flex"
                  alignItems="center"
                  justifyContent="center"
                  p={size === 'sm' ? '2px' : 1}
                  borderRadius="sm"
                  cursor="pointer"
                  color="myGray.500"
                  _hover={{ bg: 'blackAlpha.100', color: 'myGray.700' }}
                  onClick={(e: React.MouseEvent<HTMLDivElement>) => {
                    e.stopPropagation()
                    onChange(null)
                  }}
                  onKeyDown={(e: React.KeyboardEvent<HTMLDivElement>) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.stopPropagation()
                      e.preventDefault()
                      onChange(null)
                    }
                  }}
                >
                  <Icon as={FiX} fontSize={size === 'sm' ? 'xs' : 'sm'} />
                </Box>
              )}
            </HStack>
          ) : (
            <Text color="gray.400">{placeholder || t('pms.assigneePlaceholder')}</Text>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent w="260px">
        <PopoverBody p={2}>
          <VStack spacing={2} align="stretch">
            <Input
              size="sm"
              placeholder={t('pms.searchUsers')}
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              autoFocus
            />
            {loading && <Text fontSize="xs" color="gray.500">{t('common.loading')}</Text>}
            {!loading && keyword.trim() && (
              <VStack spacing={1} align="stretch" maxH="220px" overflowY="auto">
                {results.map((u) => (
                  <Button
                    key={u.userName}
                    variant="ghost"
                    justifyContent="flex-start"
                    size="sm"
                    h="auto"
                    py={2}
                    onClick={() => {
                      onChange({ userName: u.userName, fullName: u.fullName, image: u.image })
                      handleClose()
                    }}
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
                {results.length === 0 && (
                  <Text fontSize="xs" color="gray.500" p={2}>{t('pms.noUsersFound')}</Text>
                )}
              </VStack>
            )}
          </VStack>
        </PopoverBody>
      </PopoverContent>
    </Popover>
  )
}
