'use client'

import {
  Avatar,
  Box,
  Button,
  Heading,
  HStack,
  Icon,
  Text,
  VStack,
} from '@chakra-ui/react'
import { FiMessageSquare } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import MentionTextarea, { renderWithMentions } from '@/components/MentionTextarea'

interface TaskCommentsProps {
  comments: any[]
  newComment: string
  onNewCommentChange: (value: string) => void
  onSubmit: () => void
  submitting: boolean
  /** 是否允许添加评论（viewer 为 false 时隐藏输入框） */
  canComment?: boolean
}

export default function TaskComments({
  comments,
  newComment,
  onNewCommentChange,
  onSubmit,
  submitting,
  canComment = true,
}: TaskCommentsProps) {
  const { t } = useI18n()

  return (
    <Box>
      <HStack mb={3}>
        <Icon as={FiMessageSquare} />
        <Heading size="sm">{t('pms.comments')} ({comments.length})</Heading>
      </HStack>

      <VStack align="stretch" spacing={4} mb={4}>
        {comments.length === 0 ? (
          <Text color="gray.500" textAlign="center" py={4}>{t('pms.noComments')}</Text>
        ) : (
          comments.map((comment) => (
            <Box key={comment.commentId} p={3} bg="gray.50" borderRadius="md">
              <HStack justify="space-between" mb={2}>
                <HStack spacing={2}>
                  <Avatar size="xs" name={comment.authorName} />
                  <Text fontWeight="medium" fontSize="sm">{comment.authorName}</Text>
                </HStack>
                <Text fontSize="xs" color="gray.500">
                  {new Date(comment.createdAt).toLocaleString()}
                </Text>
              </HStack>
              <Text whiteSpace="pre-wrap" fontSize="sm">{renderWithMentions(comment.content)}</Text>
            </Box>
          ))
        )}
      </VStack>

      {canComment && (
        <VStack align="stretch" spacing={3}>
          <MentionTextarea
            value={newComment}
            onChange={onNewCommentChange}
            placeholder={t('pms.addCommentPlaceholder')}
            rows={3}
            size="sm"
          />
          <Button
            colorScheme="blue"
            size="sm"
            onClick={onSubmit}
            isLoading={submitting}
            isDisabled={!newComment.trim()}
            alignSelf="flex-end"
          >
            {t('pms.submitComment')}
          </Button>
        </VStack>
      )}
    </Box>
  )
}
