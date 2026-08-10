// 任务状态颜色统一工具：确保列表模式、看板模式、详情页等所有任务状态 Badge 使用同一颜色来源（taskStatuses 的 hex 颜色）

export interface TaskStatusDef {
  id: number
  name: string
  slug: string
  color: string
  position: number
  isClosed: boolean
}

// 状态别名映射：将历史/非标准状态值规范化到项目配置的 slug
// 防止任务因 status 不在 taskStatuses 配置中而无法匹配颜色
// 新 4 状态模型：todo / in_progress / review / done
// 与看板 KanbanBoard.tsx 的 STATUS_ALIASES 保持一致
export const STATUS_ALIASES: Record<string, string> = {
  open: 'todo',
  to_do: 'todo',
  completed: 'done',
  closed: 'done',
  archived: 'done',
  active: 'in_progress',
  doing: 'in_progress',
}

export function normalizeStatus(status: string): string {
  return STATUS_ALIASES[status] || status
}

// 用 taskStatuses 的 hex 颜色（与看板列头圆点一致），避免列表和看板颜色不一致
// 找不到匹配状态时返回中性灰色
export function getStatusHexColor(status: string, taskStatuses: TaskStatusDef[] | undefined | null): string {
  const normalized = normalizeStatus(status)
  const found = taskStatuses?.find((s) => s.slug === normalized)
  return found?.color || '#94a3b8'
}
