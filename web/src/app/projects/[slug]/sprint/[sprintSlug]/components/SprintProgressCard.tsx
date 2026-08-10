'use client'

import TaskProgressCard from '@/components/TaskProgressCard'

interface SprintProgressCardProps {
  tasks: any[]
}

/**
 * Sprint 进度卡：复用通用 TaskProgressCard，标题默认为 "Sprint 进度"。
 */
export default function SprintProgressCard({ tasks }: SprintProgressCardProps) {
  return <TaskProgressCard tasks={tasks} />
}
