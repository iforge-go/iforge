'use client'

import {
  Box,
  Text,
  Button,
  VStack,
  HStack,
  Icon,
  Badge,
  Textarea,
  Avatar,
  IconButton,
} from '@chakra-ui/react'
import { useMemo } from 'react'
import {
  FiCheckCircle,
  FiAlertTriangle,
  FiMessageSquare,
  FiSend,
  FiEye,
  FiThumbsUp,
  FiThumbsDown,
  FiTrash2,
  FiEdit2,
  FiGitMerge,
} from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { MarkdownEditor } from '@/components/MarkdownEditor'
import { MarkdownRenderer } from '@/components/MarkdownRenderer'
import { MergeRequest, Comment, Review, ReviewStatus, Participant } from '@/lib/api'

type TimelineItem =
  | { kind: 'comment'; data: Comment; date: string }
  | { kind: 'review'; data: Review; date: string }

interface MrConversationProps {
  mr: MergeRequest
  comments: Comment[]
  reviews: Review[]
  participantMap: Map<string, Participant>
  currentUser: { userName: string } | null
  userRole: 'owner' | 'member' | 'viewer' | '' | null
  canMerge: boolean
  canReview: boolean
  canEditMR: boolean
  dateLocale: string
  // Description editing
  isEditingDesc: boolean
  editDesc: string
  onEditDescChange: (value: string) => void
  onStartEditDesc: () => void
  onCancelEditDesc: () => void
  onSaveDesc: () => void
  editSaving: boolean
  // Comment editing
  editingCommentId: number | null
  editCommentContent: string
  onEditCommentChange: (value: string) => void
  onStartEditComment: (commentId: number, content: string) => void
  onCancelEditComment: () => void
  onSaveComment: () => void
  // Comment deletion
  onRequestDeleteComment: (commentId: number) => void
  // Review deletion
  onRequestDeleteReview: (reviewId: number) => void
  // Add comment
  newComment: string
  onNewCommentChange: (value: string) => void
  onSubmitComment: () => void
  submittingComment: boolean
  // Submit review
  reviewContent: string
  onReviewContentChange: (value: string) => void
  onSubmitReview: (status: ReviewStatus) => void
  submittingReview: boolean
}

export function MrConversation({
  mr,
  comments,
  reviews,
  participantMap,
  currentUser,
  userRole,
  canMerge,
  canReview,
  canEditMR,
  dateLocale,
  isEditingDesc,
  editDesc,
  onEditDescChange,
  onStartEditDesc,
  onCancelEditDesc,
  onSaveDesc,
  editSaving,
  editingCommentId,
  editCommentContent,
  onEditCommentChange,
  onStartEditComment,
  onCancelEditComment,
  onSaveComment,
  onRequestDeleteComment,
  onRequestDeleteReview,
  newComment,
  onNewCommentChange,
  onSubmitComment,
  submittingComment,
  reviewContent,
  onReviewContentChange,
  onSubmitReview,
  submittingReview,
}: MrConversationProps) {
  const { t } = useI18n()

  // Review summary counts
  const reviewSummary = useMemo(() => {
    const approved = reviews.filter((r) => r.status === 'approved').length
    const changesRequested = reviews.filter((r) => r.status === 'changes_requested').length
    const commented = reviews.filter((r) => r.status === 'commented').length
    return { approved, changesRequested, commented, total: reviews.length }
  }, [reviews])

  // Merge reviews into timeline (reviews + comments sorted by date)
  const timeline: TimelineItem[] = useMemo(() => {
    const items: TimelineItem[] = []
    comments.forEach((c) => items.push({ kind: 'comment', data: c, date: c.registeredDate }))
    reviews.forEach((r) => items.push({ kind: 'review', data: r, date: r.registeredDate }))
    return items.sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime())
  }, [comments, reviews])

  return (
    <>
      {/* Review summary */}
      {reviewSummary.total > 0 && (
        <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
          <HStack spacing={4} flexWrap="wrap">
            <HStack spacing={2}>
              <Icon as={FiEye} color="myGray.500" w={4} h={4} />
              <Text fontSize="sm" fontWeight="medium" color="myGray.700">
                {t('repo.reviews')} ({reviewSummary.total})
              </Text>
            </HStack>
            {reviewSummary.approved > 0 && (
              <HStack spacing={1}>
                <Icon as={FiCheckCircle} color="green.500" w={3.5} h={3.5} />
                <Text fontSize="xs" color="green.700">
                  {t('repo.approvedCount', { count: reviewSummary.approved })}
                </Text>
              </HStack>
            )}
            {reviewSummary.changesRequested > 0 && (
              <HStack spacing={1}>
                <Icon as={FiAlertTriangle} color="red.500" w={3.5} h={3.5} />
                <Text fontSize="xs" color="red.700">
                  {t('repo.changesRequestedCount', { count: reviewSummary.changesRequested })}
                </Text>
              </HStack>
            )}
            {reviewSummary.commented > 0 && (
              <HStack spacing={1}>
                <Icon as={FiMessageSquare} color="myGray.500" w={3.5} h={3.5} />
                <Text fontSize="xs" color="myGray.600">
                  {t('repo.commentedCount', { count: reviewSummary.commented })}
                </Text>
              </HStack>
            )}
          </HStack>
        </Box>
      )}

      {/* Description */}
      <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200">
        <HStack spacing={3} mb={2} justify="space-between">
          <HStack spacing={3}>
            <Avatar size="xs" name={mr.requestUserName} />
            <Text fontSize="sm" fontWeight="medium" color="myGray.900">{mr.requestUserName}</Text>
            <Text fontSize="xs" color="myGray.500">{t('repo.commented', { date: new Date(mr.createdAt).toLocaleDateString(dateLocale) })}</Text>
          </HStack>
          {canEditMR && !isEditingDesc && (
            <IconButton
              aria-label={t('repo.editDescription')}
              icon={<Icon as={FiEdit2} />}
              size="xs"
              variant="ghost"
              onClick={onStartEditDesc}
            />
          )}
        </HStack>
        {isEditingDesc ? (
          <VStack spacing={2} align="stretch">
            <MarkdownEditor value={editDesc} onChange={onEditDescChange} placeholder={t('repo.editDescription')} />
            <HStack spacing={2} justify="flex-end">
              <Button size="sm" variant="primary" onClick={onSaveDesc} isLoading={editSaving}>{t('common.save')}</Button>
              <Button size="sm" variant="ghost" onClick={onCancelEditDesc} isDisabled={editSaving}>{t('common.cancel')}</Button>
            </HStack>
          </VStack>
        ) : mr.content ? (
          <MarkdownRenderer content={mr.content} />
        ) : (
          <Text fontSize="sm" color="myGray.500" fontStyle="italic">{t('repo.noDescriptionProvided')}</Text>
        )}
      </Box>

      {/* Merge event timeline */}
      {mr.merged && mr.mergedCommitIds && (
        <Box px={4} py={3} borderBottom="1px" borderColor="myGray.100">
          <HStack spacing={3}>
            <Avatar size="xs" name={mr.requestUserName} />
            <Icon as={FiGitMerge} color="purple.500" w={4} h={4} />
            <Text fontSize="sm" color="myGray.700">
              <Text as="span" fontWeight="medium">{mr.requestUserName}</Text> {t('repo.mergedCommit')}{' '}
              <Text as="span" fontFamily="mono" color="primary.600">{mr.mergedCommitIds.substring(0, 7)}</Text>{' '}
              {t('repo.into')} <Text as="span" fontWeight="medium" color="primary.600">{mr.branch}</Text>{' '}
              <Text as="span" color="myGray.500">{new Date(mr.updatedAt || mr.createdAt).toLocaleDateString(dateLocale)}</Text>
            </Text>
          </HStack>
        </Box>
      )}

      {/* Timeline: merged reviews + comments sorted by date */}
      {timeline.length > 0 && (
        <VStack spacing={0} align="stretch">
          {timeline.map((item) => {
            if (item.kind === 'comment') {
              const comment = item.data
              return (
                <Box key={`comment-${comment.commentId}`} px={4} py={3} borderBottom="1px" borderColor="myGray.100">
                  <HStack spacing={3} mb={2} justify="space-between">
                    <HStack spacing={3}>
                      <Avatar size="xs" name={participantMap.get(comment.commentedUserName)?.fullName || comment.commentedUserName} src={participantMap.get(comment.commentedUserName)?.image || undefined} />
                      <Text fontSize="sm" fontWeight="medium" color="myGray.900">
                        {comment.commentedUserName}
                      </Text>
                      <Text fontSize="xs" color="myGray.500">
                        {t('repo.commented', { date: new Date(comment.registeredDate).toLocaleDateString(dateLocale) })}
                      </Text>
                    </HStack>
                    {editingCommentId !== comment.commentId && (
                      <HStack spacing={1}>
                        {((currentUser && currentUser.userName === comment.commentedUserName) || canMerge) && (
                          <IconButton
                            aria-label={t('repo.editComment')}
                            icon={<Icon as={FiEdit2} />}
                            size="xs"
                            variant="ghost"
                            onClick={() => onStartEditComment(comment.commentId, comment.content)}
                          />
                        )}
                        {((currentUser && currentUser.userName === comment.commentedUserName) || userRole === 'owner') && (
                          <IconButton
                            aria-label={t('common.delete')}
                            icon={<Icon as={FiTrash2} />}
                            size="xs"
                            variant="ghost"
                            colorScheme="red"
                            onClick={() => onRequestDeleteComment(comment.commentId)}
                          />
                        )}
                      </HStack>
                    )}
                  </HStack>
                  {editingCommentId === comment.commentId ? (
                    <VStack spacing={2} align="stretch">
                      <MarkdownEditor value={editCommentContent} onChange={onEditCommentChange} placeholder={t('repo.editComment')} />
                      <HStack spacing={2} justify="flex-end">
                        <Button size="sm" variant="primary" onClick={onSaveComment} isLoading={editSaving}>{t('common.save')}</Button>
                        <Button size="sm" variant="ghost" onClick={onCancelEditComment} isDisabled={editSaving}>{t('common.cancel')}</Button>
                      </HStack>
                    </VStack>
                  ) : (
                    <MarkdownRenderer content={comment.content} />
                  )}
                </Box>
              )
            }
            // Review event
            const review = item.data
            const isApproved = review.status === 'approved'
            const isChangesRequested = review.status === 'changes_requested'
            return (
              <Box key={`review-${review.reviewId}`} px={4} py={3} borderBottom="1px" borderColor="myGray.100">
                <HStack spacing={3} mb={2} justify="space-between">
                  <HStack spacing={3}>
                    <Avatar size="xs" name={review.reviewer} />
                    <Icon
                      as={isApproved ? FiCheckCircle : isChangesRequested ? FiAlertTriangle : FiMessageSquare}
                      color={isApproved ? 'green.500' : isChangesRequested ? 'red.500' : 'myGray.500'}
                      w={4}
                      h={4}
                    />
                    <Text fontSize="sm" fontWeight="medium" color="myGray.900">
                      {review.reviewer}
                    </Text>
                    <Badge
                      colorScheme={isApproved ? 'green' : isChangesRequested ? 'red' : 'gray'}
                      fontSize="xs"
                    >
                      {isApproved
                        ? t('repo.approved')
                        : isChangesRequested
                        ? t('repo.changesRequested')
                        : t('repo.reviewCommented')}
                    </Badge>
                    <Text fontSize="xs" color="myGray.500">
                      {new Date(review.registeredDate).toLocaleDateString(dateLocale)}
                    </Text>
                  </HStack>
                  <Button
                    size="xs"
                    variant="ghost"
                    colorScheme="red"
                    leftIcon={<Icon as={FiTrash2} />}
                    onClick={() => onRequestDeleteReview(review.reviewId)}
                  >
                    {t('common.delete')}
                  </Button>
                </HStack>
                {review.content && (
                  <Text fontSize="sm" whiteSpace="pre-wrap" color="myGray.700" ml={11}>
                    {review.content}
                  </Text>
                )}
              </Box>
            )
          })}
        </VStack>
      )}

      {/* Add comment */}
      <Box px={4} py={3} bg="white" borderBottom="1px" borderColor="myGray.200">
        <Textarea
          value={newComment}
          onChange={(e) => onNewCommentChange(e.target.value)}
          placeholder={t('repo.addCommentPlaceholder')}
          size="sm"
          rows={3}
          mb={2}
          bg="white"
        />
        <HStack justify="flex-end">
          <Button
            size="sm"
            colorScheme="blue"
            leftIcon={<Icon as={FiSend} />}
            onClick={onSubmitComment}
            isLoading={submittingComment}
            isDisabled={!newComment.trim()}
          >
            {t('repo.comment')}
          </Button>
        </HStack>
      </Box>

      {/* Submit review (Developer/Owner only) */}
      {mr.state === 'open' && canReview && (
        <Box px={4} py={3} bg="myGray.50">
          <Textarea
            value={reviewContent}
            onChange={(e) => onReviewContentChange(e.target.value)}
            placeholder={t('repo.reviewContentPlaceholder')}
            size="sm"
            rows={3}
            mb={2}
            bg="white"
          />
          <HStack spacing={2} justify="flex-end">
            <Button
              size="sm"
              leftIcon={<Icon as={FiMessageSquare} />}
              onClick={() => onSubmitReview('commented')}
              isLoading={submittingReview}
            >
              {t('repo.comment')}
            </Button>
            {canReview && (
              <>
                <Button
                  size="sm"
                  colorScheme="red"
                  leftIcon={<Icon as={FiThumbsDown} />}
                  onClick={() => onSubmitReview('changes_requested')}
                  isLoading={submittingReview}
                >
                  {t('repo.requestChanges')}
                </Button>
                <Button
                  size="sm"
                  colorScheme="green"
                  leftIcon={<Icon as={FiThumbsUp} />}
                  onClick={() => onSubmitReview('approved')}
                  isLoading={submittingReview}
                >
                  {t('repo.approve')}
                </Button>
              </>
            )}
          </HStack>
        </Box>
      )}
    </>
  )
}
