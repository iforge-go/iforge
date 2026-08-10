// 前端项目成员角色工具：与后端 model/project_role.go 对齐。
//
// 角色等级（越高权限越大）：
//   owner  = 4  （项目创建者，唯一）
//   admin  = 3  （成员管理 + 关联仓库 + Scrum 写）
//   member = 2  （Scrum 写）
//   viewer = 1  （只读）
//   无     = 0  （非成员）

export type ProjectRole = 'owner' | 'admin' | 'member' | 'viewer'
export type ProjectRoleOrEmpty = ProjectRole | ''

const ROLE_LEVEL: Record<string, number> = {
  owner: 4,
  admin: 3,
  member: 2,
  viewer: 1,
}

// roleLevel 返回角色等级；未知角色或空字符串返回 0。
export function roleLevel(role: string | undefined | null): number {
  if (!role) return 0
  return ROLE_LEVEL[role.toLowerCase()] ?? 0
}

// hasRoleAtLeast 判断角色等级是否 >= required。
// 用于前端权限点门控，与后端 RequireRole 语义一致。
export function hasRoleAtLeast(
  role: string | undefined | null,
  required: ProjectRole,
): boolean {
  return roleLevel(role) >= ROLE_LEVEL[required]
}

// deriveMyRole 从当前用户、项目和成员列表推导用户在项目中的角色。
// - 项目 owner（创建者）返回 'owner'
// - 成员行匹配返回其 role（小写归一化）
// - 非成员或数据异常返回 ''
//
// 这是前端统一的角色来源，所有页面应通过 ProjectContext.myRole 消费，
// 而非各自本地计算。
export function deriveMyRole(
  user: { userName: string } | null | undefined,
  project: { ownerName?: string } | null | undefined,
  members: Array<{ userName: string; role: string }> | undefined,
): ProjectRoleOrEmpty {
  if (!user || !project) return ''
  if (project.ownerName === user.userName) return 'owner'
  const m = members?.find((x) => x.userName === user.userName)
  if (!m) return ''
  const r = m.role?.toLowerCase()
  if (r === 'owner' || r === 'admin' || r === 'member' || r === 'viewer') return r
  return ''
}

// ROLE_LABELS 供 UI 显示角色名称时使用（i18n key 由调用方决定，
// 这里仅提供稳定的 role 字符串）。
export const PROJECT_ROLES: ProjectRole[] = ['admin', 'member', 'viewer']
