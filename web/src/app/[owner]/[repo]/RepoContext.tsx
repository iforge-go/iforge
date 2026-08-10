'use client'

import { createContext, useContext } from 'react'
import { Repository, Branch, Issue, MergeRequest } from '@/lib/api'

// 仓库上下文类型
export interface RepoContextType {
  repo: Repository | null
  branches: Branch[]
  issues: Issue[]
  mergeRequests: MergeRequest[]
  starred: boolean
  watching: boolean
  starCount: number
  forkCount: number
  isArchived: boolean
  userRole: 'owner' | 'member' | 'viewer' | '' | null
  canCreateIssue: boolean
  refreshData: () => Promise<void>
  handleStar: () => Promise<void>
  handleWatch: () => Promise<void>
  handleFork: () => Promise<void>
}

export const RepoContext = createContext<RepoContextType>({
  repo: null,
  branches: [],
  issues: [],
  mergeRequests: [],
  starred: false,
  watching: false,
  starCount: 0,
  forkCount: 0,
  isArchived: false,
  userRole: null,
  canCreateIssue: false,
  refreshData: async () => {},
  handleStar: async () => {},
  handleWatch: async () => {},
  handleFork: async () => {},
})

export const useRepo = () => useContext(RepoContext)
