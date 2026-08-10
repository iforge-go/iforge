'use client'

import { createContext, useContext } from 'react'
import { ProjectRoleOrEmpty } from '@/lib/projectRole'

export type ProjectContextValue = {
  project: any
  loading: boolean
  setProject: (p: any) => void
  // 成员列表（单一数据源，避免各页面重复请求）
  members: any[]
  setMembers: (m: any[]) => void
  // 当前用户在项目中的角色（'' 表示非成员/未登录/未加载）
  myRole: ProjectRoleOrEmpty
  // 派生权限：Scrum 写操作（member+）
  canEditScrum: boolean
  // 派生权限：项目管理（成员管理 + 关联仓库，admin+）
  canManageProject: boolean
  // 派生权限：项目编辑/删除（owner only）
  isProjectOwner: boolean
}

export const ProjectContext = createContext<ProjectContextValue>({
  project: null,
  loading: true,
  setProject: () => {},
  members: [],
  setMembers: () => {},
  myRole: '',
  canEditScrum: false,
  canManageProject: false,
  isProjectOwner: false,
})

export function useProject() {
  return useContext(ProjectContext)
}

// 根据路径判断当前激活的 tab（对齐 Jira Scrum tab 布局）
export function getCurrentTab(
  pathname: string,
  projectSlug: string,
): 'overview' | 'board' | 'tasks' | 'backlog' | 'reports' | 'code' | 'members' {
  const rest = pathname.replace(`/projects/${projectSlug}`, '')
  if (rest === '' || rest === '/') return 'overview'
  if (rest.startsWith('/board')) return 'board'
  if (rest.startsWith('/tasks') || rest.startsWith('/task/') || rest.startsWith('/statuses')) return 'tasks'
  if (rest.startsWith('/sprint')) return 'backlog'   // Sprint 详情页入口在需求池
  if (rest.startsWith('/backlog') || rest.startsWith('/story')) return 'backlog'
  if (rest.startsWith('/epic')) return 'backlog'      // Epic 管理移入需求池
  if (rest.startsWith('/reports')) return 'reports'
  if (rest.startsWith('/code')) return 'code'
  if (rest.startsWith('/members')) return 'members'
  return 'overview'
}
