// 用户故事状态颜色工具：与迭代状态颜色体系严格一致（gray/green/red .500 色阶）
// 迭代状态：open→gray.500（待开始）/ active→green.500（进行中）/ closed→red.500（已关闭）
// 用户故事状态映射：
//   open（待开始）→ gray.500
//   in_progress（进行中）→ green.500（对应迭代 active）
//   done（已完成）→ red.500（结束状态，对应迭代 closed）
//   closed（已关闭）→ red.500（对应迭代 closed）
// 返回 Chakra UI color token，配合 Badge 的 color/borderColor 使用
export function getStoryStatusColor(status: string): string {
  switch (status) {
    case 'open':
    case 'to_do':
    case 'todo':
      return 'gray.500'
    case 'in_progress':
      return 'green.500'
    case 'done':
    case 'completed':
      return 'red.500'
    case 'closed':
      return 'red.500'
    default:
      return 'gray.500'
  }
}
