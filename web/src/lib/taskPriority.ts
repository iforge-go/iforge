// 任务优先级颜色工具：统一看板、列表、详情页等所有任务卡片的左侧优先级色条颜色

// 优先级左侧色条颜色（与看板 TaskCard.tsx 的 getPriorityBorderColor 一致）
// urgent 红 / high 橙 / medium 黄 / low 绿
export function getPriorityBorderColor(priority: string): string {
  switch (priority) {
    case 'urgent': return '#ef4444'
    case 'high': return '#f97316'
    case 'medium': return '#eab308'
    case 'low': return '#22c55e'
    default: return '#e2e8f0'
  }
}
