'use client'

import {
  Heading,
  Text,
  VStack,
  HStack,
  Box,
  Badge,
  Avatar,
  Button,
  Input,
  IconButton,
  Icon,
} from '@chakra-ui/react'
import { FiTag, FiFlag, FiCheckCircle, FiLock, FiEdit2, FiTrash2 } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { MarkdownEditor } from '@/components/MarkdownEditor'
import { MarkdownRenderer } from '@/components/MarkdownRenderer'
import { formatRelativeTime } from '@/lib/time'
import { CommentEditor } from './CommentEditor'
import { Issue, Comment, Participant, User } from '@/lib/api'

// 判断颜色亮度，决定文字颜色
function getTextColor(bgColor: string): string {
  const hex = bgColor.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  const brightness = (r * 299 + g * 587 + b * 114) / 1000
  return brightness > 128 ? 'myGray.800' : 'white'
}

interface IssueTimelineProps {
  issue: Issue
  comments: Comment[]
  participantMap: Map<string, Participant>
  canEditIssue: boolean
  canManageIssue: boolean
  currentUser: User | null
  userRole: 'owner' | 'member' | 'viewer' | '' | null
  // Title editing
  isEditingTitle: boolean
  editTitle: string
  onEditTitleChange: (value: string) => void
  onStartEditTitle: () => void
  onCancelEditTitle: () => void
  onSaveTitle: () => void
  editSaving: boolean
  // Content editing
  isEditingContent: boolean
  editContent: string
  onEditContentChange: (value: string) => void
  onStartEditContent: () => void
  onCancelEditContent: () => void
  onSaveContent: () => void
  // Comment editing
  editingCommentId: number | null
  editCommentContent: string
  onEditCommentChange: (value: string) => void
  onStartEditComment: (commentId: number, content: string) => void
  onCancelEditComment: () => void
  onSaveComment: () => void
  // Comment deletion
  onRequestDeleteComment: (commentId: number) => void
  // Add comment
  newComment: string
  onNewCommentChange: (value: string) => void
  onSubmitComment: () => void
  submitting: boolean
}

export function IssueTimeline({
  issue,
  comments,
  participantMap,
  canEditIssue,
  canManageIssue,
  currentUser,
  userRole,
  isEditingTitle,
  editTitle,
  onEditTitleChange,
  onStartEditTitle,
  onCancelEditTitle,
  onSaveTitle,
  editSaving,
  isEditingContent,
  editContent,
  onEditContentChange,
  onStartEditContent,
  onCancelEditContent,
  onSaveContent,
  editingCommentId,
  editCommentContent,
  onEditCommentChange,
  onStartEditComment,
  onCancelEditComment,
  onSaveComment,
  onRequestDeleteComment,
  newComment,
  onNewCommentChange,
  onSubmitComment,
  submitting,
}: IssueTimelineProps) {
  const { t } = useI18n()

  return (
    <VStack spacing={6} align="stretch" flex={1}>
      {/* Issue 标题和状态 */}
      <VStack spacing={4} align="stretch">
        <HStack spacing={3} flexWrap="wrap" align="center">
          {isEditingTitle ? (
            <HStack spacing={2} flex={1}>
              <Input
                value={editTitle}
                onChange={(e) => onEditTitleChange(e.target.value)}
                size="lg"
                fontSize="xl"
                fontWeight="bold"
                autoFocus
                onKeyDown={(e) => { if (e.key === 'Enter') onSaveTitle(); if (e.key === 'Escape') onCancelEditTitle() }}
              />
              <Button size="sm" variant="primary" onClick={onSaveTitle} isLoading={editSaving}>{t('common.save')}</Button>
              <Button size="sm" variant="ghost" onClick={onCancelEditTitle} isDisabled={editSaving}>{t('common.cancel')}</Button>
            </HStack>
          ) : (
            <>
              <Heading size="lg" color="myGray.900">
                {issue.title}
              </Heading>
              {canEditIssue && (
                <IconButton aria-label={t('repo.editTitle')} icon={<Icon as={FiEdit2} />} size="sm" variant="ghost" onClick={onStartEditTitle} />
              )}
            </>
          )}
          <Badge
            colorScheme={issue.closed ? 'purple' : 'green'}
            fontSize="md"
            px={3}
            py={1}
          >
            {issue.closed ? t('repo.closed') : t('repo.open')}
          </Badge>
          {issue.locked && (
            <Badge colorScheme="red" variant="subtle" fontSize="md" px={3} py={1}>
              <HStack spacing={1}>
                <Icon as={FiLock} w={3} h={3} />
                <Text>{t('repo.issueLocked')}</Text>
              </HStack>
            </Badge>
          )}
        </HStack>
        <Text fontSize="sm" color="myGray.600">
          {t('repo.openedBy', { id: issue.issueId, user: issue.openedUserName, date: formatRelativeTime(issue.registeredDate, t) })}
        </Text>
      </VStack>

      {/* Timeline: avatar inside card header, vertical line through card bottom */}
      <Box>
        {/* Issue content */}
        <Box mb={4} position="relative">
          {/* Vertical timeline line at avatar center */}
          <Box position="absolute" left="27px" top="0" bottom="-16px" w="2px" bg="myGray.200" zIndex={0} />
          <Box borderWidth="1px" borderRadius="md" bg="white" borderColor="myGray.200" position="relative" zIndex={1}>
            <Box bg="myGray.50" px={4} py={2} borderBottomWidth="1px" borderColor="myGray.200">
              <HStack spacing={2} align="center" justify="space-between">
                <HStack spacing={2} align="center">
                  <Avatar size="xs" name={participantMap.get(issue.openedUserName)?.fullName || issue.openedUserName} src={participantMap.get(issue.openedUserName)?.image || undefined} />
                  <Text fontWeight="bold" fontSize="sm" color="myGray.900">{issue.openedUserName}</Text>
                  <Text fontSize="sm" color="myGray.500">
                    {t('repo.openedIssue')}{' '}
                    {formatRelativeTime(issue.registeredDate, t)}
                  </Text>
                </HStack>
                {canEditIssue && !isEditingContent && (
                  <IconButton aria-label={t('repo.editContent')} icon={<Icon as={FiEdit2} />} size="xs" variant="ghost" onClick={onStartEditContent} />
                )}
              </HStack>
            </Box>
            <Box p={4}>
              {isEditingContent ? (
                <VStack spacing={3} align="stretch">
                  <MarkdownEditor
                    value={editContent}
                    onChange={onEditContentChange}
                    placeholder={t('repo.editContent')}
                    height={200}
                  />
                  <HStack spacing={2} justify="flex-end">
                    <Button size="sm" variant="primary" onClick={onSaveContent} isLoading={editSaving}>{t('common.save')}</Button>
                    <Button size="sm" variant="ghost" onClick={onCancelEditContent} isDisabled={editSaving}>{t('common.cancel')}</Button>
                  </HStack>
                </VStack>
              ) : issue.content ? (
                <MarkdownRenderer content={issue.content} />
              ) : (
                <Text color="myGray.500" fontSize="sm">{t('repo.noDescription')}</Text>
              )}
            </Box>
          </Box>
        </Box>

        {/* Comments and activity events */}
        {comments.map((comment, idx) => {
          const isLast = idx === comments.length - 1
          if (comment.action && comment.action !== 'comment') {
            return (
              <Box key={comment.commentId} mb={isLast ? 0 : 4} position="relative">
                {/* Vertical line through icon center */}
                <Box position="absolute" left="27px" top="0" bottom="-16px" w="2px" bg="myGray.200" zIndex={0} />
                <HStack align="center" spacing={2} ml="16px" position="relative" zIndex={1}>
                  <Box
                    w="24px"
                    h="24px"
                    flexShrink={0}
                    borderRadius="full"
                    bg="white"
                    borderWidth="2px"
                    borderColor="myGray.200"
                    display="flex"
                    alignItems="center"
                    justifyContent="center"
                  >
                    <Icon
                      as={
                        comment.action === 'add_label' || comment.action === 'remove_label'
                          ? FiTag
                          : comment.action === 'add_milestone' || comment.action === 'remove_milestone'
                          ? FiFlag
                          : FiCheckCircle
                      }
                      color="myGray.500"
                      w={3}
                      h={3}
                    />
                  </Box>
                  <Text fontSize="sm" color="myGray.600">
                    <Text as="span" fontWeight="bold" color="myGray.900">{comment.commentedUserName}</Text>
                    {' '}
                    {comment.action === 'add_label' && `${t('repo.addLabelAction')} `}
                    {comment.action === 'remove_label' && `${t('repo.removeLabelAction')} `}
                    {comment.action === 'add_milestone' && `${t('repo.addMilestoneAction')} `}
                    {comment.action === 'remove_milestone' && `${t('repo.removeMilestoneAction')} `}
                    {comment.action === 'close' && t('repo.closeAction')}
                    {comment.action === 'reopen' && t('repo.reopenAction')}
                    {comment.content && (
                      (() => {
                        const parts = comment.content!.split('|')
                        const labelName = parts[0]
                        const labelColor = parts[1]
                        if (labelColor) {
                          return (
                            <Box
                              as="span"
                              display="inline-block"
                              px={2}
                              py={0}
                              borderRadius="full"
                              bg={`#${labelColor}`}
                              color={getTextColor(labelColor)}
                              fontSize="xs"
                              fontWeight="bold"
                              verticalAlign="middle"
                            >
                              {labelName}
                            </Box>
                          )
                        }
                        return (
                          <Text as="span" fontWeight="medium" color="primary.600">{comment.content}</Text>
                        )
                      })()
                    )}
                    <Text as="span" ml={2} color="myGray.400">
                      {formatRelativeTime(comment.registeredDate, t)}
                    </Text>
                  </Text>
                </HStack>
              </Box>
            )
          }

          return (
            <Box key={comment.commentId} mb={isLast ? 0 : 4} position="relative">
              {/* Vertical line through avatar center */}
              <Box position="absolute" left="27px" top="0" bottom="-16px" w="2px" bg="myGray.200" zIndex={0} />
              <Box borderWidth="1px" borderRadius="md" bg="white" borderColor="myGray.200" position="relative" zIndex={1}>
                <Box bg="myGray.50" px={4} py={2} borderBottomWidth="1px" borderColor="myGray.200">
                  <HStack spacing={2} align="center" justify="space-between">
                    <HStack spacing={2} align="center">
                      <Avatar size="xs" name={participantMap.get(comment.commentedUserName)?.fullName || comment.commentedUserName} src={participantMap.get(comment.commentedUserName)?.image || undefined} />
                      <Text fontWeight="bold" fontSize="sm" color="myGray.900">{comment.commentedUserName}</Text>
                      <Text fontSize="sm" color="myGray.500">
                        {t('repo.commentedAt', { date: formatRelativeTime(comment.registeredDate, t) })}
                      </Text>
                    </HStack>
                    {editingCommentId !== comment.commentId && (
                      <HStack spacing={1}>
                        {((currentUser && currentUser.userName === comment.commentedUserName) || canManageIssue) && (
                          <IconButton aria-label={t('repo.editComment')} icon={<Icon as={FiEdit2} />} size="xs" variant="ghost" onClick={() => onStartEditComment(comment.commentId, comment.content)} />
                        )}
                        {((currentUser && currentUser.userName === comment.commentedUserName) || userRole === 'owner') && (
                          <IconButton aria-label={t('common.delete')} icon={<Icon as={FiTrash2} />} size="xs" variant="ghost" colorScheme="red" onClick={() => onRequestDeleteComment(comment.commentId)} />
                        )}
                      </HStack>
                    )}
                  </HStack>
                </Box>
                <Box p={4}>
                  {editingCommentId === comment.commentId ? (
                    <VStack spacing={3} align="stretch">
                      <MarkdownEditor
                        value={editCommentContent}
                        onChange={onEditCommentChange}
                        placeholder={t('repo.editComment')}
                        height={150}
                      />
                      <HStack spacing={2} justify="flex-end">
                        <Button size="sm" variant="primary" onClick={onSaveComment} isLoading={editSaving}>{t('common.save')}</Button>
                        <Button size="sm" variant="ghost" onClick={onCancelEditComment} isDisabled={editSaving}>{t('common.cancel')}</Button>
                      </HStack>
                    </VStack>
                  ) : (
                    <MarkdownRenderer content={comment.content} />
                  )}
                </Box>
              </Box>
            </Box>
          )
        })}

        {/* 添加评论 */}
        <CommentEditor
          locked={!!issue.locked}
          value={newComment}
          onChange={onNewCommentChange}
          onSubmit={onSubmitComment}
          submitting={submitting}
        />
      </Box>
    </VStack>
  )
}
