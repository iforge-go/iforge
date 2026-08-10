// CI/CD 页面共享辅助函数
import {
  FiClock,
  FiCheckCircle,
  FiXCircle,
  FiLoader,
  FiMinusCircle,
  FiAlertCircle,
  FiGitPullRequest,
  FiUpload,
  FiUser,
  FiCalendar,
  FiExternalLink,
} from 'react-icons/fi'
import type { IconType } from 'react-icons'

// 状态 -> 颜色 (Chakra colorScheme)
export function pipelineStatusColor(status: string): string {
  switch (status) {
    case 'success': return 'green.500'
    case 'failed': return 'red.500'
    case 'running': return 'blue.500'
    case 'pending': return 'myGray.400'
    case 'canceled': return 'myGray.400'
    case 'skipped': return 'myGray.300'
    default: return 'myGray.400'
  }
}

// 状态 -> 图标
export function pipelineStatusIcon(status: string): IconType {
  switch (status) {
    case 'success': return FiCheckCircle
    case 'failed': return FiXCircle
    case 'running': return FiLoader
    case 'pending': return FiClock
    case 'canceled': return FiMinusCircle
    case 'skipped': return FiMinusCircle
    default: return FiAlertCircle
  }
}

// 状态 -> 中文/英文标签
export function statusLabel(status: string, t: (k: string, p?: any) => string): string {
  switch (status) {
    case 'pending': return t('cicd.statusPending')
    case 'running': return t('cicd.statusRunning')
    case 'success': return t('cicd.statusSuccess')
    case 'failed': return t('cicd.statusFailed')
    case 'canceled': return t('cicd.statusCanceled')
    case 'skipped': return t('cicd.statusSkipped')
    default: return status
  }
}

// 触发事件 -> 图标
export function triggerEventIcon(event: string): IconType {
  switch (event) {
    case 'push': return FiUpload
    case 'merge_request': return FiGitPullRequest
    case 'manual': return FiUser
    case 'cron': return FiCalendar
    case 'external': return FiExternalLink
    default: return FiAlertCircle
  }
}

// 触发事件 -> 标签
export function triggerEventLabel(event: string, t: (k: string, p?: any) => string): string {
  switch (event) {
    case 'push': return t('cicd.triggerPush')
    case 'merge_request': return t('cicd.triggerMergeRequest')
    case 'manual': return t('cicd.triggerManual')
    case 'cron': return t('cicd.triggerCron')
    case 'external': return t('cicd.triggerExternal')
    default: return event
  }
}

// 部署状态 -> 颜色
export function deploymentStatusColor(status: string): string {
  switch (status) {
    case 'success': return 'green.500'
    case 'failed': return 'red.500'
    case 'in_progress': return 'blue.500'
    case 'canceled': return 'myGray.400'
    default: return 'myGray.400'
  }
}

// 部署状态 -> 标签
export function deploymentStatusLabel(status: string, t: (k: string, p?: any) => string): string {
  switch (status) {
    case 'in_progress': return t('cicd.deployInProgress')
    case 'success': return t('cicd.deploySuccess')
    case 'failed': return t('cicd.deployFailed')
    case 'canceled': return t('cicd.deployCanceled')
    default: return status
  }
}

// 持续时间格式化(毫秒 -> "1m 30s" / "30s")
export function formatDuration(ms: number, t: (k: string, p?: any) => string): string {
  const totalSec = Math.floor(ms / 1000)
  const min = Math.floor(totalSec / 60)
  const sec = totalSec % 60
  if (min > 0) return t('cicd.durationMinutes', { min, sec })
  return t('cicd.durationSeconds', { sec })
}

// 计算正在运行的 job 的已运行时长(返回毫秒)
export function elapsedMs(startedAt?: string | null): number | null {
  if (!startedAt) return null
  const start = new Date(startedAt).getTime()
  return Date.now() - start
}

// 字节数格式化(1024 -> "1.0 KB", 1048576 -> "1.0 MB")
export function formatBytes(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const idx = Math.min(i, units.length - 1)
  const val = bytes / Math.pow(1024, idx)
  return `${val.toFixed(idx === 0 ? 0 : 1)} ${units[idx]}`
}
