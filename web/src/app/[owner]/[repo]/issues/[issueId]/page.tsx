'use client'

import {
  Text,
  Flex,
  Button,
  Spinner,
  useDisclosure,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
} from '@chakra-ui/react'
import { useEffect, useState, useMemo } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { api, Issue, Comment, Label, Milestone, User, UserSearchResult, Participant } from '@/lib/api'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { IssueTimeline } from './components/IssueTimeline'
import { IssueSidebar } from './components/IssueSidebar'

export default function IssueDetailPage() {
  const params = useParams()
  const router = useRouter()
  const owner = params.owner as string
  const repoName = params.repo as string
  const issueId = parseInt(params.issueId as string)
  const toast = useGithubToast()
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'
  const { userRole } = useRepo()

  const { user: currentUser } = useCurrentUser()
  const [issue, setIssue] = useState<Issue | null>(null)
  const [comments, setComments] = useState<Comment[]>([])
  const [participants, setParticipants] = useState<Participant[]>([])
  const [newComment, setNewComment] = useState('')
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [allLabels, setAllLabels] = useState<Label[]>([])
  const [issueLabels, setIssueLabels] = useState<Label[]>([])
  const [assignees, setAssignees] = useState<User[]>([])
  const [userSearchKeyword, setUserSearchKeyword] = useState('')
  const [userSearchResults, setUserSearchResults] = useState<UserSearchResult[]>([])
  const [userSearchLoading, setUserSearchLoading] = useState(false)
  const [assigneeLoading, setAssigneeLoading] = useState(false)
  const [isAssigneePopoverOpen, setIsAssigneePopoverOpen] = useState(false)
  const [milestones, setMilestones] = useState<Milestone[]>([])
  const [currentMilestone, setCurrentMilestone] = useState<Milestone | null>(null)
  const [lockLoading, setLockLoading] = useState(false)
  const { isOpen: isLockOpen, onOpen: onLockOpen, onClose: onLockClose } = useDisclosure()
  const [pendingLockAction, setPendingLockAction] = useState<'lock' | 'unlock' | null>(null)

  // Inline editing state
  const [isEditingTitle, setIsEditingTitle] = useState(false)
  const [editTitle, setEditTitle] = useState('')
  const [isEditingContent, setIsEditingContent] = useState(false)
  const [editContent, setEditContent] = useState('')
  const [editingCommentId, setEditingCommentId] = useState<number | null>(null)
  const [editCommentContent, setEditCommentContent] = useState('')
  const [editSaving, setEditSaving] = useState(false)
  const { isOpen: isDeleteCommentOpen, onOpen: onDeleteCommentOpen, onClose: onDeleteCommentClose } = useDisclosure()
  const [deletingCommentId, setDeletingCommentId] = useState<number | null>(null)

  // 判断是否有 Developer 权限（可以管理标签、里程碑、指派人员）
  const canManageIssue = userRole === 'owner' || userRole === 'member'

  // 判断是否是 issue 作者
  const isIssueAuthor = currentUser && issue && currentUser.userName === issue.openedUserName

  // 判断是否可以编辑 issue（作者或 Developer/Owner）
  const canEditIssue = isIssueAuthor || canManageIssue

  // 判断是否可以锁定/解锁 issue（Owner 或作者）
  const canLockIssue = userRole === 'owner' || isIssueAuthor

  // 构建 participant map：issue opener 排第一，然后是评论者
  const participantMap = useMemo(() => {
    const map = new Map<string, Participant>()
    // 创建者排第一
    if (issue) {
      map.set(issue.openedUserName, {
        userName: issue.openedUserName,
        fullName: issue.openedUserFullName || issue.openedUserName,
        image: issue.openedUserImage ?? null,
      })
    }
    // 追加评论者（已存在的跳过）
    for (const p of participants) {
      if (!map.has(p.userName)) {
        map.set(p.userName, p)
      }
    }
    return map
  }, [participants, issue])

  // 所有参与者列表（去重），用于 Participants 栏
  const allParticipants = useMemo(() => {
    return Array.from(participantMap.values())
  }, [participantMap])

  useEffect(() => {
    Promise.all([
      api.getIssue(owner, repoName, issueId),
      api.listIssueComments(owner, repoName, issueId).catch(() => ({ comments: [], participants: [] })),
      api.listLabels(owner, repoName).catch(() => []),
      api.listMilestones(owner, repoName).catch(() => []),
    ])
      .then(([issueData, commentsData, labelsData, milestonesData]) => {
        if (issueData.isMergeRequest) {
          router.replace(`/${owner}/${repoName}/merge-requests/${issueId}`)
          return
        }
        setIssue(issueData)
        setComments(commentsData.comments || [])
        setParticipants(commentsData.participants || [])
        setAllLabels(labelsData || [])
        setIssueLabels(issueData.labels || [])
        setAssignees(issueData.assignees || [])
        setMilestones(milestonesData || [])
        if (issueData.milestoneId) {
          const ms = (milestonesData || []).find(m => m.milestoneId === issueData.milestoneId)
          setCurrentMilestone(ms || null)
        }
      })
      .finally(() => setLoading(false))
  }, [owner, repoName, issueId])

  // Search users with debounce (for assignee picker)
  useEffect(() => {
    if (!userSearchKeyword.trim()) {
      setUserSearchResults([])
      return
    }
    setUserSearchLoading(true)
    const timer = setTimeout(async () => {
      try {
        const result = await api.searchUsers(userSearchKeyword, 20)
        setUserSearchResults(result.users || [])
      } catch {
        setUserSearchResults([])
      } finally {
        setUserSearchLoading(false)
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [userSearchKeyword])

  const handleAddAssignee = async (username: string) => {
    setAssigneeLoading(true)
    try {
      const { assignees: updated } = await api.addAssignee(owner, repoName, issueId, username)
      setAssignees(updated || [])
      setUserSearchKeyword('')
      setIsAssigneePopoverOpen(false)
      refreshComments()
      toast({
        title: t('repo.assigneeAdded'),
        status: 'success',
        duration: 2000,
      })
    } catch (error: any) {
      toast({
        title: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setAssigneeLoading(false)
    }
  }

  const handleRemoveAssignee = async (username: string) => {
    try {
      const { assignees: updated } = await api.removeAssignee(owner, repoName, issueId, username)
      setAssignees(updated || [])
      refreshComments()
      toast({
        title: t('repo.assigneeRemoved'),
        status: 'success',
        duration: 2000,
      })
    } catch (error: any) {
      toast({
        title: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleAddComment = async () => {
    if (!newComment.trim()) return

    setSubmitting(true)
    try {
      const comment = await api.addIssueComment(owner, repoName, issueId, newComment)
      setComments([...comments, comment])
      setNewComment('')
      toast({
        title: t('repo.commentAdded'),
        status: 'success',
        duration: 2000,
      })
    } catch (error: any) {
      toast({
        title: t('repo.addCommentFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSubmitting(false)
    }
  }

  const handleSaveTitle = async () => {
    if (!issue || !editTitle.trim()) return
    setEditSaving(true)
    try {
      await api.updateIssue(owner, repoName, issueId, { title: editTitle })
      setIssue({ ...issue, title: editTitle })
      setIsEditingTitle(false)
      toast({ title: t('repo.issueUpdated'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('repo.issueUpdateFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setEditSaving(false)
    }
  }

  const handleSaveContent = async () => {
    if (!issue) return
    setEditSaving(true)
    try {
      await api.updateIssue(owner, repoName, issueId, { body: editContent })
      setIssue({ ...issue, content: editContent })
      setIsEditingContent(false)
      toast({ title: t('repo.issueUpdated'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('repo.issueUpdateFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setEditSaving(false)
    }
  }

  const handleSaveComment = async () => {
    if (!issue || editingCommentId === null) return
    setEditSaving(true)
    try {
      const updated = await api.updateIssueComment(owner, repoName, issueId, editingCommentId, editCommentContent)
      setComments(comments.map(c => c.commentId === editingCommentId ? updated : c))
      setEditingCommentId(null)
      toast({ title: t('repo.commentUpdated'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('repo.commentUpdateFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setEditSaving(false)
    }
  }

  const handleDeleteComment = async () => {
    if (!issue || deletingCommentId === null) return
    setEditSaving(true)
    try {
      await api.deleteIssueComment(owner, repoName, issueId, deletingCommentId)
      setComments(comments.filter(c => c.commentId !== deletingCommentId))
      onDeleteCommentClose()
      toast({ title: t('repo.commentDeleted'), status: 'success', duration: 2000 })
    } catch (error: any) {
      toast({ title: t('repo.commentDeleteFailed'), description: error.message, status: 'error', duration: 3000 })
    } finally {
      setEditSaving(false)
    }
  }

  const handleToggleLock = async () => {
    if (!issue || !pendingLockAction) return
    setLockLoading(true)
    try {
      if (pendingLockAction === 'lock') {
        await api.lockIssue(owner, repoName, issueId)
        setIssue({ ...issue, locked: true })
        toast({ title: t('repo.issueLockedSuccess'), status: 'success', duration: 2000 })
      } else {
        await api.unlockIssue(owner, repoName, issueId)
        setIssue({ ...issue, locked: false })
        toast({ title: t('repo.issueUnlockedSuccess'), status: 'success', duration: 2000 })
      }
      onLockClose()
      setPendingLockAction(null)
    } catch (error: any) {
      toast({
        title: t('repo.lockIssueFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLockLoading(false)
    }
  }

  const openLockConfirm = (action: 'lock' | 'unlock') => {
    setPendingLockAction(action)
    onLockOpen()
  }

  // 重新获取评论和参与者（用于标签/里程碑/指派等操作后刷新时间轴）
  const refreshComments = () => {
    api.listIssueComments(owner, repoName, issueId)
      .then(data => {
        setComments(data.comments || [])
        setParticipants(data.participants || [])
      })
      .catch(() => {})
  }

  const handleAddLabel = async (labelId: number) => {
    try {
      await api.addLabelToIssue(owner, repoName, issueId, labelId)
      const label = allLabels.find(l => l.labelId === labelId)
      if (label && !issueLabels.find(l => l.labelId === labelId)) {
        setIssueLabels([...issueLabels, label])
      }
      refreshComments()
      toast({
        title: t('repo.labelAdded'),
        status: 'success',
        duration: 2000,
      })
    } catch (error: any) {
      toast({
        title: t('repo.addLabelFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleRemoveLabel = async (labelId: number) => {
    try {
      await api.removeLabelFromIssue(owner, repoName, issueId, labelId)
      setIssueLabels(issueLabels.filter(l => l.labelId !== labelId))
      refreshComments()
      toast({
        title: t('repo.labelRemoved'),
        status: 'success',
        duration: 2000,
      })
    } catch (error: any) {
      toast({
        title: t('repo.removeLabelFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleSetMilestone = async (milestoneId: number | null) => {
    try {
      await api.updateIssue(owner, repoName, issueId, { milestoneId })
      setCurrentMilestone(milestoneId ? milestones.find(m => m.milestoneId === milestoneId) || null : null)
      refreshComments()
      toast({
        title: milestoneId ? t('repo.milestoneSet') : t('repo.milestoneCleared'),
        status: 'success',
        duration: 2000,
      })
    } catch (error: any) {
      toast({
        title: t('repo.operationFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  if (loading) {
    return <Text>{t('common.loading')}</Text>
  }

  if (!issue) {
    return <Text>{t('repo.issueNotExist')}</Text>
  }

  return (
    <Flex gap={6} align="start">
      {/* 主内容区 */}
      <IssueTimeline
        issue={issue}
        comments={comments}
        participantMap={participantMap}
        canEditIssue={canEditIssue}
        canManageIssue={canManageIssue}
        currentUser={currentUser}
        userRole={userRole}
        isEditingTitle={isEditingTitle}
        editTitle={editTitle}
        onEditTitleChange={setEditTitle}
        onStartEditTitle={() => { setEditTitle(issue.title); setIsEditingTitle(true) }}
        onCancelEditTitle={() => setIsEditingTitle(false)}
        onSaveTitle={handleSaveTitle}
        editSaving={editSaving}
        isEditingContent={isEditingContent}
        editContent={editContent}
        onEditContentChange={setEditContent}
        onStartEditContent={() => { setEditContent(issue.content || ''); setIsEditingContent(true) }}
        onCancelEditContent={() => setIsEditingContent(false)}
        onSaveContent={handleSaveContent}
        editingCommentId={editingCommentId}
        editCommentContent={editCommentContent}
        onEditCommentChange={setEditCommentContent}
        onStartEditComment={(commentId, content) => { setEditingCommentId(commentId); setEditCommentContent(content) }}
        onCancelEditComment={() => setEditingCommentId(null)}
        onSaveComment={handleSaveComment}
        onRequestDeleteComment={(commentId) => { setDeletingCommentId(commentId); onDeleteCommentOpen() }}
        newComment={newComment}
        onNewCommentChange={setNewComment}
        onSubmitComment={handleAddComment}
        submitting={submitting}
      />

      {/* 侧边栏 */}
      <IssueSidebar
        issue={issue}
        owner={owner}
        repoName={repoName}
        assignees={assignees}
        allLabels={allLabels}
        issueLabels={issueLabels}
        milestones={milestones}
        currentMilestone={currentMilestone}
        allParticipants={allParticipants}
        canManageIssue={canManageIssue}
        canLockIssue={!!canLockIssue}
        dateLocale={dateLocale}
        lockLoading={lockLoading}
        onLockAction={openLockConfirm}
        userSearchKeyword={userSearchKeyword}
        onUserSearchKeywordChange={setUserSearchKeyword}
        userSearchResults={userSearchResults}
        userSearchLoading={userSearchLoading}
        isAssigneePopoverOpen={isAssigneePopoverOpen}
        onAssigneePopoverOpenChange={setIsAssigneePopoverOpen}
        assigneeLoading={assigneeLoading}
        onAddAssignee={handleAddAssignee}
        onRemoveAssignee={handleRemoveAssignee}
        onAddLabel={handleAddLabel}
        onRemoveLabel={handleRemoveLabel}
        onSetMilestone={handleSetMilestone}
      />

      <Modal isOpen={isLockOpen} onClose={onLockClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>
            {pendingLockAction === 'lock' ? t('repo.lockIssue') : t('repo.unlockIssue')}
          </ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text fontSize="sm" color="myGray.600">
              {pendingLockAction === 'lock'
                ? t('repo.issueLockedNotice')
                : t('repo.issueUnlockedSuccess')}
            </Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onLockClose}>{t('common.cancel')}</Button>
            <Button
              colorScheme={pendingLockAction === 'lock' ? 'red' : 'green'}
              variant="solid"
              onClick={handleToggleLock}
              isLoading={lockLoading}
            >
              {lockLoading ? <Spinner size="sm" /> : t('common.confirm')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Delete comment confirmation */}
      <Modal isOpen={isDeleteCommentOpen} onClose={onDeleteCommentClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('common.delete')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text fontSize="sm" color="myGray.600">{t('repo.deleteCommentConfirm')}</Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="whiteBase" mr={3} onClick={onDeleteCommentClose}>{t('common.cancel')}</Button>
            <Button colorScheme="red" variant="solid" onClick={handleDeleteComment} isLoading={editSaving}>
              {editSaving ? <Spinner size="sm" /> : t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Flex>
  )
}
