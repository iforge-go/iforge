// 任务类型图标工具：统一看板、列表等所有任务卡片的类型 icon

// 任务类型对应的 emoji 图标与颜色（与看板 TaskCard.tsx 的 getTaskTypeIcon 一致）
// bug 🐛 / feature 🚀 / improvement 🔧 / 默认 📋
export function getTaskTypeIcon(taskType: string): { label: string; color: string } {
  switch (taskType) {
    case 'bug': return { label: '🐛', color: 'red.500' }
    case 'feature': return { label: '🚀', color: 'purple.500' }
    case 'improvement': return { label: '🔧', color: 'yellow.600' }
    default: return { label: '📋', color: 'blue.500' }
  }
}
