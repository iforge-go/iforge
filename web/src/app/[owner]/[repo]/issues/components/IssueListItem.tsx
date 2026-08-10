'use client'

import {
  Text,
  HStack,
  Box,
  Icon,
  Badge,
  Checkbox,
  VStack,
  Avatar,
  AvatarGroup,
} from '@chakra-ui/react'
import Link from 'next/link'
import {
  FiCircle,
  FiCheckCircle,
  FiMessageSquare,
  FiLock,
} from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { formatRelativeTime } from '@/lib/time'
import type { Issue, Participant } from '@/lib/types'

// 判断颜色亮度，决定文字颜色
function getTextColor(bgColor: string): string {
  const hex = bgColor.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  const brightness = (r * 299 + g * 587 + b * 114) / 1000
  return brightness > 128 ? 'myGray.800' : 'white'
}

interface IssueListItemProps {
  issue: Issue
  owner: string
  repoName: string
  isSelected: boolean
  canBatchOperate: boolean
  participantsMap: Map<string, Participant>
  onSelectIssue: (id: number, checked: boolean) => void
}

export function IssueListItem({
  issue,
  owner,
  repoName,
  isSelected,
  canBatchOperate,
  participantsMap,
  onSelectIssue,
}: IssueListItemProps) {
  const { t } = useI18n()

  return (
    <Box
      py={3}
      px={4}
      borderBottom="1px"
      borderColor="myGray.100"
      _last={{ borderBottom: 'none' }}
      _hover={{ bg: 'myGray.50' }}
      bg={isSelected ? 'primary.50' : 'transparent'}
    >
      <HStack spacing={3} align="start">
        {canBatchOperate && (
          <Checkbox
            isChecked={isSelected}
            onChange={(e) => onSelectIssue(issue.issueId, e.target.checked)}
            colorScheme="primary"
            mt={1}
          />
        )}
        <Icon
          as={issue.closed ? FiCheckCircle : FiCircle}
          color={issue.closed ? 'purple.500' : 'green.500'}
          w={4}
          h={4}
          mt={1}
          flexShrink={0}
        />
        <VStack align="start" spacing={1} flex={1}>
          <HStack spacing={2} flexWrap="wrap">
            <Link href={`/${owner}/${repoName}/issues/${issue.issueId}`}>
              <Text fontSize="sm" fontWeight="medium" color="myGray.900" _hover={{ color: 'primary.600' }}>
                {issue.title}
              </Text>
            </Link>
            {issue.locked && (
              <Badge colorScheme="red" variant="subtle">
                <HStack spacing={1}>
                  <Icon as={FiLock} w={3} h={3} />
                  <Text>{t('repo.issueLocked')}</Text>
                </HStack>
              </Badge>
            )}
            {issue.labels && issue.labels.length > 0 && (
              <HStack spacing={1}>
                {issue.labels.map((label) => (
                  <Badge
                    key={label.labelId}
                    bg={`#${label.color}`}
                    color={getTextColor(label.color)}
                    fontSize="10px"
                    px={1.5}
                    py={0.5}
                    borderRadius="full"
                    fontWeight="normal"
                  >
                    {label.labelName}
                  </Badge>
                ))}
              </HStack>
            )}
          </HStack>
          <Text fontSize="xs" color="myGray.500">
            {t('repo.openedBy', {
              id: issue.issueId,
              user: issue.openedUserName,
              date: formatRelativeTime(issue.registeredDate, t),
            })}
          </Text>
        </VStack>
        {/* 参与者头像组（固定宽度，左对齐） */}
        <HStack mt={1} flexShrink={0} minW="100px" justify="start">
          {issue.participantUserNames && issue.participantUserNames.length > 0 && (
            <AvatarGroup size="xs" max={5} spacing={-2}>
              {issue.participantUserNames.map((name) => {
                const p = participantsMap.get(name)
                return (
                  <Avatar
                    key={name}
                    name={p?.fullName || name}
                    src={p?.image || undefined}
                    borderWidth="2px"
                    borderColor="white"
                    showBorder
                  />
                )
              })}
            </AvatarGroup>
          )}
        </HStack>
        {/* 评论数（固定宽度，左对齐） */}
        <HStack mt={1} flexShrink={0} minW="40px" justify="start">
          {issue.commentsCount > 0 && (
            <HStack spacing={1}>
              <Icon as={FiMessageSquare} w={3} h={3} color="myGray.400" />
              <Text fontSize="xs" color="myGray.500">{issue.commentsCount}</Text>
            </HStack>
          )}
        </HStack>
      </HStack>
    </Box>
  )
}
