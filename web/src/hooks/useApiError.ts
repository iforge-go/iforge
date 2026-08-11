import { useCallback } from 'react'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { getLocalizedErrorMessage, isApiError } from '@/lib/errorMessages'
// ApiError type removed - not directly used in this file

/**
 * 统一的 API 错误处理 hook
 * 
 * 提供标准化的错误展示方式，自动处理 i18n 翻译和 toast 显示
 * 
 * @example
 * ```tsx
 * const { handleError, handleApiError } = useApiError()
 * 
 * // 方式 1: 直接处理错误对象
 * try {
 *   await api.someMethod()
 * } catch (err) {
 *   handleError(err)
 * }
 * 
 * // 方式 2: 处理 API 错误（带自定义标题）
 * try {
 *   await api.createUser(data)
 * } catch (err) {
 *   handleApiError(err, '创建用户失败')
 * }
 * ```
 */
export function useApiError() {
  const toast = useGithubToast()
  const { t } = useI18n()

  /**
   * 处理错误并显示 toast
   * 
   * @param error - 错误对象（Error、ApiError 或其他）
   * @param options - 配置选项
   * @param options.title - toast 标题（可选）
   * @param options.duration - 显示时长（毫秒，默认 5000）
   * @param options.silent - 是否静默处理（不显示 toast，默认 false）
   */
  const handleError = useCallback(
    (
      error: unknown,
      options?: {
        title?: string
        duration?: number
        silent?: boolean
      }
    ) => {
      const { title, duration = 5000, silent = false } = options || {}

      // 静默模式：只记录日志，不显示 toast
      if (silent) {
        if (process.env.NODE_ENV === 'development') {
          console.error('[API Error]', error)
        }
        return
      }

      // 获取本地化的错误消息
      const message = getLocalizedErrorMessage(error, t)

      // 显示 toast
      toast({
        title: title || t('common.error'),
        description: message,
        status: 'error',
        duration,
        isClosable: true,
      })

      // 开发环境下同时输出到控制台
      if (process.env.NODE_ENV === 'development') {
        console.error('[API Error]', error)
      }
    },
    [toast, t]
  )

  /**
   * 处理 API 错误的快捷方法
   * 
   * 自动提取错误信息并显示，支持自定义标题
   * 
   * @param error - API 错误对象
   * @param title - 可选的自定义标题
   */
  const handleApiError = useCallback(
    (error: unknown, title?: string) => {
      handleError(error, { title })
    },
    [handleError]
  )

  /**
   * 检查错误是否为特定的 API 错误原因
   * 
   * @example
   * ```tsx
   * if (isErrorReason(err, 'task_blocked')) {
   *   // 处理任务被阻塞的情况
   * }
   * ```
   */
  const isErrorReason = useCallback(
    (error: unknown, reason: string): boolean => {
      return isApiError(error) && error.reason === reason
    },
    []
  )

  /**
   * 获取错误中的 reason 字段
   */
  const getErrorReason = useCallback(
    (error: unknown): string | undefined => {
      return isApiError(error) ? error.reason : undefined
    },
    []
  )

  /**
   * 获取错误中的 blockedTasks 字段
   */
  const getBlockedTasks = useCallback(
    (error: unknown): any[] | undefined => {
      return isApiError(error) ? error.blockedTasks : undefined
    },
    []
  )

  return {
    handleError,
    handleApiError,
    isErrorReason,
    getErrorReason,
    getBlockedTasks,
  }
}
