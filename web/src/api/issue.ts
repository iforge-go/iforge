import type { User, Issue, IssueTemplate, Comment, Participant, MergeRequest, Review, ReviewStatus, ReviewComment, Label, Milestone } from '../lib/types'

type RequestFn = <T>(endpoint: string, options?: { method?: string; body?: any; token?: string | null }) => Promise<T>

export function createIssueApi(request: RequestFn, _baseUrl: string) {
  return {
    async listIssues(owner: string, repo: string) {
      return request<{ issues: Issue[]; participants: Participant[] }>(`/repos/${owner}/${repo}/issues`)
    },

    // Distinct issue/MR authors / assignees for filter dropdowns
    async listIssueAuthors(owner: string, repo: string, type?: 'issue' | 'mr') {
      const query = type === 'mr' ? '?type=mr' : ''
      return request<User[]>(`/repos/${owner}/${repo}/issues/authors${query}`)
    },

    async listIssueAssignees(owner: string, repo: string, type?: 'issue' | 'mr') {
      const query = type === 'mr' ? '?type=mr' : ''
      return request<User[]>(`/repos/${owner}/${repo}/issues/assignees${query}`)
    },

    async getIssue(owner: string, repo: string, issueId: number) {
      return request<Issue>(`/repos/${owner}/${repo}/issues/${issueId}`)
    },

    async updateIssue(owner: string, repo: string, issueId: number, data: { title?: string; body?: string; state?: string; milestoneId?: number | null }) {
      return request<Issue>(`/repos/${owner}/${repo}/issues/${issueId}`, {
        method: 'PATCH',
        body: data,
      })
    },

    async createIssue(owner: string, repo: string, title: string, content: string, assignees?: string[]) {
      return request<Issue>(`/repos/${owner}/${repo}/issues`, {
        method: 'POST',
        body: { title, body: content, assignees },
      })
    },

    // Issue assignees
    async listAssignees(owner: string, repo: string, issueId: number) {
      return request<{ assignees: User[] }>(`/repos/${owner}/${repo}/issues/${issueId}/assignees`)
    },

    async addAssignee(owner: string, repo: string, issueId: number, username: string) {
      return request<{ assignees: User[] }>(`/repos/${owner}/${repo}/issues/${issueId}/assignees/${username}`, {
        method: 'POST',
      })
    },

    async removeAssignee(owner: string, repo: string, issueId: number, username: string) {
      return request<{ assignees: User[] }>(`/repos/${owner}/${repo}/issues/${issueId}/assignees/${username}`, {
        method: 'DELETE',
      })
    },

    // Issue locking
    async lockIssue(owner: string, repo: string, issueId: number) {
      return request<{ message: string }>(`/repos/${owner}/${repo}/issues/${issueId}/lock`, {
        method: 'POST',
      })
    },

    async unlockIssue(owner: string, repo: string, issueId: number) {
      return request<{ message: string }>(`/repos/${owner}/${repo}/issues/${issueId}/lock`, {
        method: 'DELETE',
      })
    },

    // Issue templates
    async getIssueTemplates(owner: string, repo: string) {
      return request<IssueTemplate[]>(`/repos/${owner}/${repo}/issues/templates`)
    },

    // Batch update issues
    async batchUpdateIssues(owner: string, repo: string, data: {
      issueIds: number[]
      closed?: boolean
      milestoneId?: number
      priorityId?: number
      addLabelIds?: number[]
      removeLabelIds?: number[]
    }) {
      return request<{ message: string; updated: number }>(`/repos/${owner}/${repo}/issues/batch`, {
        method: 'POST',
        body: data,
      })
    },

    async listIssueComments(owner: string, repo: string, issueId: number) {
      return request<{ comments: Comment[]; participants: Participant[] }>(`/repos/${owner}/${repo}/issues/${issueId}/comments`)
    },

    async addIssueComment(owner: string, repo: string, issueId: number, content: string) {
      return request<Comment>(`/repos/${owner}/${repo}/issues/${issueId}/comments`, {
        method: 'POST',
        body: { body: content },
      })
    },

    async updateIssueComment(owner: string, repo: string, issueId: number, commentId: number, content: string) {
      return request<Comment>(`/repos/${owner}/${repo}/issues/${issueId}/comments/${commentId}`, {
        method: 'PATCH',
        body: { content },
      })
    },

    async deleteIssueComment(owner: string, repo: string, issueId: number, commentId: number) {
      return request<void>(`/repos/${owner}/${repo}/issues/${issueId}/comments/${commentId}`, {
        method: 'DELETE',
      })
    },

    // Labels
    async listLabels(owner: string, repo: string) {
      return request<Label[]>(`/repos/${owner}/${repo}/labels`)
    },

    async createLabel(owner: string, repo: string, name: string, color: string) {
      return request<Label>(`/repos/${owner}/${repo}/labels`, {
        method: 'POST',
        body: { name, color },
      })
    },

    async updateLabel(owner: string, repo: string, labelId: number, data: Partial<Label>) {
      return request<Label>(`/repos/${owner}/${repo}/labels/${labelId}`, {
        method: 'PATCH',
        body: data,
      })
    },

    async deleteLabel(owner: string, repo: string, labelId: number) {
      return request<void>(`/repos/${owner}/${repo}/labels/${labelId}`, {
        method: 'DELETE',
      })
    },

    async addLabelToIssue(owner: string, repo: string, issueId: number, labelId: number) {
      return request<Label>(`/repos/${owner}/${repo}/issues/${issueId}/labels/${labelId}`, {
        method: 'POST',
      })
    },

    async removeLabelFromIssue(owner: string, repo: string, issueId: number, labelId: number) {
      return request<void>(`/repos/${owner}/${repo}/issues/${issueId}/labels/${labelId}`, {
        method: 'DELETE',
      })
    },

    // Milestones
    async listMilestones(owner: string, repo: string) {
      return request<Milestone[]>(`/repos/${owner}/${repo}/milestones`)
    },

    async getMilestone(owner: string, repo: string, milestoneId: number) {
      return request<Milestone>(`/repos/${owner}/${repo}/milestones/${milestoneId}`)
    },

    async createMilestone(owner: string, repo: string, title: string, description: string, dueDate?: string) {
      return request<Milestone>(`/repos/${owner}/${repo}/milestones`, {
        method: 'POST',
        body: { title, description, dueDate },
      })
    },

    async updateMilestone(owner: string, repo: string, milestoneId: number, title: string, description: string, dueDate?: string) {
      return request<Milestone>(`/repos/${owner}/${repo}/milestones/${milestoneId}`, {
        method: 'PATCH',
        body: { title, description, dueDate },
      })
    },

    async deleteMilestone(owner: string, repo: string, milestoneId: number) {
      return request<void>(`/repos/${owner}/${repo}/milestones/${milestoneId}`, {
        method: 'DELETE',
      })
    },

    async closeMilestone(owner: string, repo: string, milestoneId: number) {
      return request<void>(`/repos/${owner}/${repo}/milestones/${milestoneId}/close`, {
        method: 'POST',
      })
    },

    async reopenMilestone(owner: string, repo: string, milestoneId: number) {
      return request<void>(`/repos/${owner}/${repo}/milestones/${milestoneId}/reopen`, {
        method: 'POST',
      })
    },

    // Merge Requests
    async listMergeRequests(owner: string, repo: string, state?: string) {
      const params = state ? `?state=${state}` : ''
      const resp = await request<{ mergeRequests: MergeRequest[]; participants: Participant[] }>(`/repos/${owner}/${repo}/merge-requests${params}`)
      return resp || { mergeRequests: [], participants: [] }
    },

    async getMergeRequest(owner: string, repo: string, mrId: number) {
      return request<MergeRequest>(`/repos/${owner}/${repo}/merge-requests/${mrId}`)
    },

    async createMergeRequest(
      owner: string,
      repo: string,
      title: string,
      body: string,
      head: string,
      base: string,
      headUser?: string,
      headRepo?: string
    ) {
      return request<MergeRequest>(`/repos/${owner}/${repo}/merge-requests`, {
        method: 'POST',
        body: { title, body, head, base, headUser, headRepo },
      })
    },

    async mergeMergeRequest(owner: string, repo: string, mrId: number, strategy: string, message?: string) {
      return request<{ message: string; mergeCommit: string; strategy: string }>(
        `/repos/${owner}/${repo}/merge-requests/${mrId}/merge`,
        {
          method: 'POST',
          body: { strategy, message },
        }
      )
    },

    async updateMergeRequest(owner: string, repo: string, mrId: number, data: { title?: string; body?: string; state?: string }) {
      return request<MergeRequest>(`/repos/${owner}/${repo}/merge-requests/${mrId}`, {
        method: 'PATCH',
        body: data,
      })
    },

    async listMergeRequestComments(owner: string, repo: string, mrId: number) {
      return request<{ comments: Comment[]; participants: Participant[] }>(`/repos/${owner}/${repo}/issues/${mrId}/comments`)
    },

    async addMergeRequestComment(owner: string, repo: string, mrId: number, content: string) {
      return request<Comment>(`/repos/${owner}/${repo}/issues/${mrId}/comments`, {
        method: 'POST',
        body: { body: content },
      })
    },

    // Reviews
    async listReviews(owner: string, repo: string, mrId: number) {
      return request<Review[]>(`/repos/${owner}/${repo}/merge-requests/${mrId}/reviews`)
    },

    async createReview(
      owner: string,
      repo: string,
      mrId: number,
      status: ReviewStatus,
      content?: string
    ) {
      return request<Review>(`/repos/${owner}/${repo}/merge-requests/${mrId}/reviews`, {
        method: 'POST',
        body: { status, content: content || null },
      })
    },

    async updateReview(
      owner: string,
      repo: string,
      mrId: number,
      reviewId: number,
      status: ReviewStatus,
      content?: string
    ) {
      return request<void>(`/repos/${owner}/${repo}/merge-requests/${mrId}/reviews/${reviewId}`, {
        method: 'PATCH',
        body: { status, content: content || null },
      })
    },

    async deleteReview(owner: string, repo: string, mrId: number, reviewId: number) {
      return request<void>(`/repos/${owner}/${repo}/merge-requests/${mrId}/reviews/${reviewId}`, {
        method: 'DELETE',
      })
    },

    async listReviewComments(owner: string, repo: string, mrId: number, reviewId: number) {
      return request<ReviewComment[]>(
        `/repos/${owner}/${repo}/merge-requests/${mrId}/reviews/${reviewId}/comments`
      )
    },

    async createReviewComment(
      owner: string,
      repo: string,
      mrId: number,
      reviewId: number,
      filePath: string,
      line: number,
      content: string
    ) {
      return request<ReviewComment>(
        `/repos/${owner}/${repo}/merge-requests/${mrId}/reviews/${reviewId}/comments`,
        {
          method: 'POST',
          body: { filePath, line, content },
        }
      )
    },

    async deleteReviewComment(
      owner: string,
      repo: string,
      mrId: number,
      reviewId: number,
      commentId: number
    ) {
      return request<void>(
        `/repos/${owner}/${repo}/merge-requests/${mrId}/reviews/${reviewId}/comments/${commentId}`,
        { method: 'DELETE' }
      )
    },

    async getAheadBehind(owner: string, repo: string, branch: string, base: string) {
      return request<{ ahead: number; behind: number }>(
        `/repos/${owner}/${repo}/branches/${encodeURIComponent(branch)}/ahead-behind?base=${encodeURIComponent(base)}`
      )
    },

    // Search issues with advanced query syntax (is:open, author:xxx, label:xxx, ...)
    async searchIssues(owner: string, repo: string, q: string, page = 1, limit = 25) {
      const params = new URLSearchParams({ q, page: String(page), limit: String(limit) })
      return request<{
        issues: Issue[]
        total: number
        limit: number
        offset: number
        participants: Participant[]
      }>(`/repos/${owner}/${repo}/issues/search?${params.toString()}`)
    },

    async compare(
      owner: string,
      repo: string,
      base: string,
      head: string,
      headUser?: string,
      headRepo?: string
    ) {
      let url = `/repos/${owner}/${repo}/compare/${encodeURIComponent(base)}...${encodeURIComponent(head)}`
      const params = new URLSearchParams()
      if (headUser) params.append('headUser', headUser)
      if (headRepo) params.append('headRepo', headRepo)
      const queryString = params.toString()
      if (queryString) url += `?${queryString}`

      return request<{
        commits: Array<{ id: string; message: string; author: string; timestamp: string }>
        fileChanges: Array<{ filename: string; additions: number; deletions: number; status: string }>
        baseCommit: string
        headCommit: string
      }>(url)
    },
  }
}
