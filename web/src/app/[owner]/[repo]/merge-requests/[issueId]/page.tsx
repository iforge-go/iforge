'use client'

import {
  Box,
  Text,
  Button,
  VStack,
  HStack,
  Icon,
  Spinner,
  Badge,
  Avatar,
  Tabs,
  TabList,
  Tab,
  TabPanels,
  TabPanel,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
} from '@chakra-ui/react'
import { useGithubToast } from '@/app/providers'
import { useParams, useRouter } from 'next/navigation'
import { useState, useEffect, useMemo, useRef } from 'react'
import { api, MergeRequest, Comment, Review, ReviewStatus, Participant } from '@/lib/api'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import {
  FiGitPullRequest,
  FiGitBranch,
  FiCheckCircle,
  FiAlertTriangle,
  FiGitMerge,
  FiXCircle,
  FiRotateCcw,
  FiMessageSquare,
  FiGitCommit,
  FiFileText,
  FiCheckSquare,
} from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import { MrConversation } from './components/MrConversation'
import { MrCommits } from './components/MrCommits'
import { MrFilesChanged } from './components/MrFilesChanged'
import { MrChecks } from './components/MrChecks'

interface CompareResult {
  commits: Array<{ id: string; message: string; author: string; timestamp: string }>
  fileChanges: Array<{ filename: string; additions: number; deletions: number; status: string; patch?: string }>
  baseCommit: string
  headCommit: string
  mergeable?: boolean
}

export default function MergeRequestDetailPage() {
  const params = useParams()
  const router = useRouter()
  const owner = params.owner as string
  const repoName = params.repo as string
  const issueId = parseInt(params.issueId as string)
  const { branches, userRole, refreshData } = useRepo()
  const toast = useGithubToast()
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'

  const [mr, setMr] = useState<MergeRequest | null>(null)
  const [compareResult, setCompareResult] = useState<CompareResult | null>(null)
  const [comments, setComments] = useState<Comment[]>([])
  const [participants, setParticipants] = useState<Participant[]>([])
  const [reviews, setReviews] = useState<Review[]>([])
  const [reviewComments, setReviewComments] = useState<any[]>([])
  const [newComment, setNewComment] = useState('')
  const [submittingComment, setSubmittingComment] = useState(false)
  const [loading, setLoading] = useState(true)
  const [merging, setMerging] = useState(false)
  const { user: currentUser } = useCurrentUser()

  // Review submission state
  const [reviewContent, setReviewContent] = useState('')
  const [submittingReview, setSubmittingReview] = useState(false)
  const [deletingReviewId, setDeletingReviewId] = useState<number | null>(null)

  // Compare 结果缓存：MR 的分支/状态/合并提交未变时跳过 cross-repo fetch+diff
  // key 由 base/head 分支、head 仓库、合并提交、状态组成；结果不变即无需重跑 compare
  const compareCacheRef = useRef<{ key: string; result: CompareResult | null } | null>(null)

  // loadData 互斥锁：React StrictMode 开发模式下 useEffect 会双调用，
  // 导致 compare 等重 API 被并发请求两次。用 ref 挡住第二次并发调用。
  const loadingRef = useRef(false)

  // Inline editing state
  const [isEditingDesc, setIsEditingDesc] = useState(false)
  const [editDesc, setEditDesc] = useState('')
  const [editingCommentId, setEditingCommentId] = useState<number | null>(null)
  const [editCommentContent, setEditCommentContent] = useState('')
  const [editSaving, setEditSaving] = useState(false)
  const [deletingCommentId, setDeletingCommentId] = useState<number | null>(null)

  // 权限判断
  const canMerge = userRole === 'owner' || userRole === 'member'
  const canReview = userRole === 'owner' || userRole === 'member'
  const isMRAuthor = currentUser && mr && currentUser.userName === mr.requestUserName
  const canCloseOrReopen = isMRAuthor || canMerge
  const canEditMR = isMRAuthor || canMerge

  // 构建 participant map
  const participantMap = useMemo(() => {
    const map = new Map<string, Participant>()
    for (const p of participants) {
      map.set(p.userName, p)
    }
    if (mr) {
      const author = map.get(mr.requestUserName)
      if (!author) {
        map.set(mr.requestUserName, {
          userName: mr.requestUserName,
          fullName: mr.requestUserName,
          image: null,
        })
      }
    }
    return map
  }, [participants, mr])

  const allParticipants = useMemo(() => Array.from(participantMap.values()), [participantMap])

  const loadData = async () => {
    // StrictMode 双调用互斥：第二次并发调用直接跳过，避免 compare 被请求两次
    if (loadingRef.current) return
    loadingRef.current = true
    try {
      const mrData = await api.getMergeRequest(owner, repoName, issueId)
      setMr(mrData)
      const isCrossRepo = mrData.requestRepositoryName && mrData.requestUserName &&
        (mrData.requestUserName !== owner || mrData.requestRepositoryName !== repoName)

      // Compare 缓存：MR 的 base/head 分支、head 仓库、合并提交、状态都没变时，
      // compare 结果必然不变（review/评论提交不会改变 commit graph），跳过 cross-repo fetch+diff。
      const compareKey = [
        mrData.branch || 'main',
        mrData.requestBranch || 'dev',
        isCrossRepo ? mrData.requestUserName : '',
        isCrossRepo ? mrData.requestRepositoryName : '',
        mrData.mergedCommitIds || '',
        mrData.state,
      ].join('|')
      const cacheHit = compareCacheRef.current?.key === compareKey
      const cachedResult = cacheHit ? compareCacheRef.current!.result : null

      const comparePromise = (cacheHit || mrData.state !== 'open')
        ? Promise.resolve(cachedResult)
        : api.compare(
            owner, repoName,
            mrData.branch || 'main',
            mrData.requestBranch || 'dev',
            isCrossRepo ? mrData.requestUserName : undefined,
            isCrossRepo ? mrData.requestRepositoryName : undefined,
          ).catch(() => null)

      const [compareData, commentsData, reviewsData] = await Promise.all([
        comparePromise,
        api.listMergeRequestComments(owner, repoName, issueId).catch(() => ({ comments: [], participants: [] })),
        api.listReviews(owner, repoName, issueId).catch(() => []),
      ])
      // 写入缓存（即使 compareData 为 null 也记下 key，避免失败时反复重试）
      if (!cacheHit) {
        compareCacheRef.current = { key: compareKey, result: compareData }
      }
      if (compareData) {
        setCompareResult(compareData)
      }
      setComments(commentsData.comments || [])
      setParticipants(commentsData.participants || [])
      const reviewsList = reviewsData || []
      setReviews(reviewsList)
      // 先关闭 loading：review inline comments 加载较慢且不阻塞页面主体显示，
      // 放到 loading 关闭之后异步加载，避免页面长时间转圈。
      loadingRef.current = false
      setLoading(false)
      // Load inline comments for all reviews (异步加载，不阻塞 loading)
      const commentPromises = reviewsList.map((r) =>
        api.listReviewComments(owner, repoName, issueId, r.reviewId).catch(() => [])
      )
      const commentResults = await Promise.all(commentPromises)
      const allComments = commentResults.flat()
      setReviewComments(allComments)
    } catch {
      toast({
        title: t('repo.loadFailed'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      loadingRef.current = false
      setLoading(false)
    }
  }

  // reloadReviews 只刷新 review 列表及其内联评论，不触碰 MR 本身和 compare 结果。
  // 用于 review 提交/删除后：这些操作不改变 commit graph，无需重跑 compare。
  const reloadReviews = async () => {
    try {
      const reviewsData = await api.listReviews(owner, repoName, issueId).catch(() => [])
      const reviewsList = reviewsData || []
      setReviews(reviewsList)
      const commentPromises = reviewsList.map((r) =>
        api.listReviewComments(owner, repoName, issueId, r.reviewId).catch(() => [])
      )
      const commentResults = await Promise.all(commentPromises)
      setReviewComments(commentResults.flat())
    } catch {
      // 静默失败：toast 已由调用方在成功路径展示
    }
  }

  // reloadComments 只刷新评论列表和参与者，不触碰 MR 本身和 compare 结果。
  // 用于新增评论后：评论不影响 commit graph。
  const reloadComments = async () => {
    try {
      const commentsData = await api.listMergeRequestComments(owner, repoName, issueId).catch(() => ({ comments: [], participants: [] }))
      setComments(commentsData.comments || [])
      setParticipants(commentsData.participants || [])
    } catch {
      // 静默失败
    }
  }

  useEffect(() => {
    loadData()
  }, [owner, repoName, issueId])

  const handleMerge = async (strategy: string) => {
    if (!mr) return
    setMerging(true)
    try {
      await api.mergeMergeRequest(owner, repoName, issueId, strategy)
      toast({
        title: t('repo.mergeSuccess'),
        status: 'success',
        duration: 3000,
      })
      await loadData()
      await refreshData()
    } catch (err: any) {
      toast({
        title: t('repo.mergeFailed'),
        description: err.message || t('repo.unknownError'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setMerging(false)
    }
  }

  const handleClose = async () => {
    if (!mr) return
    try {
      await api.updateMergeRequest(owner, repoName, issueId, { state: 'closed' })
      toast({
        title: t('repo.closedSuccess'),
        status: 'success',
        duration: 3000,
      })
      await loadData()
      await refreshData()
    } catch (err: any) {
      toast({
        title: t('repo.operationFailed'),
        description: err.message || t('repo.unknownError'),
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleReopen = async () => {
    if (!mr) return
    try {
      await api.updateMergeRequest(owner, repoName, issueId, { state: 'open' })
      toast({
        title: t('repo.reopenedSuccess'),
        status: 'success',
        duration: 3000,
      })
      await loadData()
      await refreshData()
    } catch (err: any) {
      toast({
        title: t('repo.operationFailed'),
        description: err.message || t('repo.unknownError'),
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleAddComment = async () => {
    if (!newComment.trim()) return
    setSubmittingComment(true)
    try {
      await api.addMergeRequestComment(owner, repoName, issueId, newComment)
      setNewComment('')
      await reloadComments()
      toast({
        title: t('repo.commentSuccess'),
        status: 'success',
        duration: 2000,
      })
    } catch (err: any) {
      toast({
        title: t('repo.commentFailed'),
        description: err.message || t('repo.unknownError'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSubmittingComment(false)
    }
  }

  const handleSubmitReview = async (status: ReviewStatus) => {
    setSubmittingReview(true)
    try {
      await api.createReview(owner, repoName, issueId, status, reviewContent.trim() || undefined)
      setReviewContent('')
      await reloadReviews()
      toast({
        title: t('repo.reviewSubmitted'),
        status: 'success',
        duration: 2000,
      })
    } catch (err: any) {
      toast({
        title: t('repo.reviewSubmitFailed'),
        description: err.message || t('repo.unknownError'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSubmittingReview(false)
    }
  }

  const handleConfirmDeleteReview = async () => {
    if (deletingReviewId === null) return
    try {
      await api.deleteReview(owner, repoName, issueId, deletingReviewId)
      toast({ title: t('repo.reviewDeleted'), status: 'success', duration: 2000 })
      await reloadReviews()
    } catch (err: any) {
      toast({
        title: t('repo.reviewDeleteFailed'),
        description: err.message || t('repo.unknownError'),
        status: 'error',
        duration: 3000,
      })
    } finally {
      setDeletingReviewId(null)
    }
  }

  const handleSaveDesc = async () => {
    if (!mr) return
    setEditSaving(true)
    try {
      const updated = await api.updateMergeRequest(owner, repoName, issueId, { body: editDesc })
      setMr(updated)
      setIsEditingDesc(false)
      toast({ title: t('repo.mrUpdated'), status: 'success', duration: 2000 })
    } catch (err: any) {
      toast({ title: t('repo.mrUpdateFailed'), description: err.message, status: 'error', duration: 3000 })
    } finally {
      setEditSaving(false)
    }
  }

  const handleSaveComment = async () => {
    if (editingCommentId === null) return
    setEditSaving(true)
    try {
      const updated = await api.updateIssueComment(owner, repoName, issueId, editingCommentId, editCommentContent)
      setComments(comments.map(c => c.commentId === editingCommentId ? updated : c))
      setEditingCommentId(null)
      toast({ title: t('repo.commentUpdated'), status: 'success', duration: 2000 })
    } catch (err: any) {
      toast({ title: t('repo.commentUpdateFailed'), description: err.message, status: 'error', duration: 3000 })
    } finally {
      setEditSaving(false)
    }
  }

  const handleDeleteComment = async () => {
    if (deletingCommentId === null) return
    setEditSaving(true)
    try {
      await api.deleteIssueComment(owner, repoName, issueId, deletingCommentId)
      setComments(comments.filter(c => c.commentId !== deletingCommentId))
      setDeletingCommentId(null)
      toast({ title: t('repo.commentDeleted'), status: 'success', duration: 2000 })
    } catch (err: any) {
      toast({ title: t('repo.commentDeleteFailed'), description: err.message, status: 'error', duration: 3000 })
    } finally {
      setEditSaving(false)
    }
  }

  if (loading) {
    return (
      <Box py={12} textAlign="center">
        <Spinner size="lg" />
      </Box>
    )
  }

  if (!mr) {
    return (
      <Box py={12} textAlign="center">
        <Text color="myGray.500">{t('repo.mergeRequestNotExist')}</Text>
      </Box>
    )
  }

  // For merged MRs, use stored data; for open MRs, use live compare
  const displayCommits = mr.mergedCommits
    ? (JSON.parse(mr.mergedCommits) || [])
    : (compareResult?.commits || [])
  const displayFileChanges = mr.mergedFileChanges
    ? (JSON.parse(mr.mergedFileChanges) || [])
    : (compareResult?.fileChanges || [])

  const totalAdditions = displayFileChanges.reduce((sum: number, f: any) => sum + (f.additions || 0), 0)
  const totalDeletions = displayFileChanges.reduce((sum: number, f: any) => sum + (f.deletions || 0), 0)

  return (
    <VStack spacing={4} align="stretch">
      {/* Header */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
        <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200">
          <HStack spacing={3} flexWrap="wrap">
            <Icon as={FiGitPullRequest} color={mr.state === 'open' ? 'green.500' : mr.state === 'merged' ? 'purple.500' : 'myGray.400'} w={5} h={5} />
            <Text fontSize="lg" fontWeight="bold" color="myGray.900">
              {mr.title}
            </Text>
            <Badge colorScheme={mr.state === 'open' ? 'green' : mr.state === 'merged' ? 'purple' : 'gray'}>
              {mr.state === 'open' ? t('repo.open') : mr.state === 'merged' ? t('repo.merged') : t('repo.closed')}
            </Badge>
            {mr.isDraft && <Badge colorScheme="gray">{t('repo.draft')}</Badge>}
            <Icon as={FiGitBranch} color="myGray.500" w={4} h={4} />
            <Text fontSize="sm">
              <Text as="span" fontWeight="medium" color="primary.600">
                {mr.requestUserName && mr.requestRepositoryName &&
                 (mr.requestUserName !== owner || mr.requestRepositoryName !== repoName)
                  ? `${mr.requestUserName}/${mr.requestRepositoryName}:${mr.requestBranch}`
                  : mr.requestBranch}
              </Text>
              <Text as="span" color="myGray.500"> → </Text>
              <Text as="span" fontWeight="medium" color="primary.600">{mr.branch}</Text>
            </Text>
            <Text fontSize="sm" color="myGray.500">
              #{mr.issueId} {mr.state === 'merged' ? t('repo.mergedAt') : mr.state === 'closed' ? t('repo.closedAt') : t('repo.openedAt')} {new Date(mr.createdAt).toLocaleDateString(dateLocale)} {t('repo.by')} {mr.requestUserName}
            </Text>
          </HStack>
        </Box>

        {/* Merge status - only show for open MRs */}
        {mr.state === 'open' && compareResult && (
          <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200">
            <HStack spacing={2} px={3} py={2} borderRadius="md" bg={compareResult.mergeable ? 'green.50' : 'yellow.50'}>
              <Icon
                as={compareResult.mergeable ? FiCheckCircle : FiAlertTriangle}
                color={compareResult.mergeable ? 'green.600' : 'yellow.600'}
                w={4}
                h={4}
              />
              <Text fontSize="sm" color={compareResult.mergeable ? 'green.700' : 'yellow.700'}>
                {compareResult.mergeable
                  ? t('repo.ableToMerge')
                  : t('repo.hasConflicts')}
              </Text>
            </HStack>
          </Box>
        )}

        {/* Merged banner */}
        {mr.state === 'merged' && (
          <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200" bg="purple.50">
            <HStack spacing={3}>
              <Icon as={FiGitMerge} color="purple.600" w={5} h={5} />
              <VStack align="start" spacing={0}>
                <Text fontSize="sm" fontWeight="medium" color="purple.700">
                  {t('repo.mergedBanner')}
                </Text>
                <Text fontSize="xs" color="purple.600">
                  {t('repo.mergedBranch', { source: mr.requestBranch, target: mr.branch })}
                </Text>
              </VStack>
            </HStack>
          </Box>
        )}

        {/* Closed banner */}
        {mr.state === 'closed' && !mr.merged && (
          <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
            <HStack spacing={3}>
              <Icon as={FiXCircle} color="myGray.500" w={5} h={5} />
              <Text fontSize="sm" color="myGray.600">
                {t('repo.closedBanner')}
              </Text>
            </HStack>
          </Box>
        )}

        {/* Action buttons */}
        {mr.state === 'open' && (
          <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
            <HStack spacing={2} justify="space-between" flexWrap="wrap">
              <HStack spacing={2}>
                {compareResult?.mergeable && !mr.isDraft && canMerge && (
                  <>
                    <Button
                      size="sm"
                      colorScheme="green"
                      leftIcon={<Icon as={FiGitMerge} />}
                      onClick={() => handleMerge('merge-commit')}
                      isLoading={merging}
                    >
                      {t('repo.mergeCommit')}
                    </Button>
                    <Button
                      size="sm"
                      colorScheme="blue"
                      leftIcon={<Icon as={FiGitMerge} />}
                      onClick={() => handleMerge('squash')}
                      isLoading={merging}
                    >
                      {t('repo.squashAndMerge')}
                    </Button>
                    <Button
                      size="sm"
                      colorScheme="purple"
                      leftIcon={<Icon as={FiGitMerge} />}
                      onClick={() => handleMerge('rebase')}
                      isLoading={merging}
                    >
                      {t('repo.rebaseAndMerge')}
                    </Button>
                  </>
                )}
              </HStack>
              {canCloseOrReopen && (
              <Button
                size="sm"
                colorScheme="red"
                variant="outline"
                leftIcon={<Icon as={FiXCircle} />}
                onClick={handleClose}
              >
                {t('repo.closeMergeRequest')}
              </Button>
              )}
            </HStack>
          </Box>
        )}

        {/* Reopen button */}
        {mr.state === 'closed' && !mr.merged && canCloseOrReopen && (
          <Box px={4} py={3} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
            <HStack justify="flex-end">
              <Button
                size="sm"
                colorScheme="green"
                leftIcon={<Icon as={FiRotateCcw} />}
                onClick={handleReopen}
              >
                {t('repo.reopenMergeRequest')}
              </Button>
            </HStack>
          </Box>
        )}
      </Box>

      {/* Tabs: Conversation / Commits / Checks / Files changed */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
        <Tabs variant="unstyled" defaultIndex={0}>
          <TabList borderBottom="1px" borderColor="myGray.200" px={4}>
            <Tab
              fontSize="sm"
              color="myGray.600"
              py={3}
              _selected={{ color: 'myGray.900', borderBottom: '2px', borderColor: 'primary.500', fontWeight: 'medium' }}
            >
              <HStack spacing={1}>
                <Icon as={FiMessageSquare} w={3.5} h={3.5} />
                <Text>{t('repo.conversation')}</Text>
                {comments.length > 0 && <Badge colorScheme="gray" fontSize="xs">{comments.length}</Badge>}
              </HStack>
            </Tab>
            <Tab
              fontSize="sm"
              color="myGray.600"
              py={3}
              _selected={{ color: 'myGray.900', borderBottom: '2px', borderColor: 'primary.500', fontWeight: 'medium' }}
            >
              <HStack spacing={1}>
                <Icon as={FiGitCommit} w={3.5} h={3.5} />
                <Text>{t('repo.commitsTab')}</Text>
                {compareResult?.commits && <Badge colorScheme="gray" fontSize="xs">{compareResult.commits.length}</Badge>}
              </HStack>
            </Tab>
            <Tab
              fontSize="sm"
              color="myGray.600"
              py={3}
              _selected={{ color: 'myGray.900', borderBottom: '2px', borderColor: 'primary.500', fontWeight: 'medium' }}
            >
              <HStack spacing={1}>
                <Icon as={FiCheckSquare} w={3.5} h={3.5} />
                <Text>{t('repo.checksTab')}</Text>
              </HStack>
            </Tab>
            <Tab
              fontSize="sm"
              color="myGray.600"
              py={3}
              _selected={{ color: 'myGray.900', borderBottom: '2px', borderColor: 'primary.500', fontWeight: 'medium' }}
            >
              <HStack spacing={1}>
                <Icon as={FiFileText} w={3.5} h={3.5} />
                <Text>{t('repo.filesChangedTab')}</Text>
                {compareResult?.fileChanges && <Badge colorScheme="gray" fontSize="xs">{compareResult.fileChanges.length}</Badge>}
                {(totalAdditions > 0 || totalDeletions > 0) && (
                  <HStack spacing={0} fontSize="xs">
                    <Text color="green.600">+{totalAdditions}</Text>
                    <Text color="red.600">-{totalDeletions}</Text>
                  </HStack>
                )}
              </HStack>
            </Tab>
          </TabList>

          <TabPanels>
            {/* Conversation Tab */}
            <TabPanel px={0} py={0}>
              <MrConversation
                mr={mr}
                comments={comments}
                reviews={reviews}
                participantMap={participantMap}
                currentUser={currentUser}
                userRole={userRole}
                canMerge={canMerge}
                canReview={canReview}
                canEditMR={canEditMR}
                dateLocale={dateLocale}
                isEditingDesc={isEditingDesc}
                editDesc={editDesc}
                onEditDescChange={setEditDesc}
                onStartEditDesc={() => { setEditDesc(mr.content || ''); setIsEditingDesc(true) }}
                onCancelEditDesc={() => setIsEditingDesc(false)}
                onSaveDesc={handleSaveDesc}
                editSaving={editSaving}
                editingCommentId={editingCommentId}
                editCommentContent={editCommentContent}
                onEditCommentChange={setEditCommentContent}
                onStartEditComment={(commentId, content) => { setEditingCommentId(commentId); setEditCommentContent(content) }}
                onCancelEditComment={() => setEditingCommentId(null)}
                onSaveComment={handleSaveComment}
                onRequestDeleteComment={(commentId) => setDeletingCommentId(commentId)}
                onRequestDeleteReview={(reviewId) => setDeletingReviewId(reviewId)}
                newComment={newComment}
                onNewCommentChange={setNewComment}
                onSubmitComment={handleAddComment}
                submittingComment={submittingComment}
                reviewContent={reviewContent}
                onReviewContentChange={setReviewContent}
                onSubmitReview={handleSubmitReview}
                submittingReview={submittingReview}
              />
            </TabPanel>

            {/* Commits Tab */}
            <TabPanel px={0} py={0}>
              <MrCommits commits={displayCommits} />
            </TabPanel>

            {/* Checks Tab */}
            <TabPanel px={0} py={0}>
              <MrChecks
                owner={owner}
                repoName={repoName}
                sourceOwner={mr.requestUserName && mr.requestRepositoryName &&
                  (mr.requestUserName !== owner || mr.requestRepositoryName !== repoName)
                  ? mr.requestUserName : ''}
                sourceRepo={mr.requestUserName && mr.requestRepositoryName &&
                  (mr.requestUserName !== owner || mr.requestRepositoryName !== repoName)
                  ? mr.requestRepositoryName : ''}
                headCommitSha={
                  mr.mergedCommitIds
                    ? (mr.mergedCommitIds.split(',')[0] || null)
                    : (compareResult?.headCommit || (compareResult?.commits?.[0]?.id ?? null))
                }
              />
            </TabPanel>

            {/* Files changed Tab */}
            <TabPanel px={0} py={0}>
              <MrFilesChanged
                fileChanges={displayFileChanges}
                reviewComments={reviewComments}
                dateLocale={dateLocale}
              />
            </TabPanel>
          </TabPanels>
        </Tabs>
      </Box>

      {/* Delete review confirmation */}
      <Modal isOpen={deletingReviewId !== null} onClose={() => setDeletingReviewId(null)}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('repo.deleteReview')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>{t('repo.deleteReviewConfirm')}</Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => setDeletingReviewId(null)}>
              {t('common.cancel')}
            </Button>
            <Button variant="solid" colorScheme="red" onClick={handleConfirmDeleteReview}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Delete comment confirmation */}
      <Modal isOpen={deletingCommentId !== null} onClose={() => setDeletingCommentId(null)}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('common.delete')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>{t('repo.deleteCommentConfirm')}</Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => setDeletingCommentId(null)}>
              {t('common.cancel')}
            </Button>
            <Button variant="solid" colorScheme="red" onClick={handleDeleteComment} isLoading={editSaving}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </VStack>
  )
}
