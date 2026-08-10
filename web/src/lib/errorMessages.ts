// 错误消息本地化工具
//
// 后端对"面向用户展示的本地化错误"返回 { error: "English msg", messageKey: "errors.xxx" }，
// messageKey 直接就是前端 i18n 的翻译键。这里直接 t(messageKey) 即可，无需中间映射表。
//
// 另有一类"程序逻辑判断标识符"走 reason 字段（如 reason: "task_blocked"），
// 那是给前端走分支逻辑用的，不经过这里。

/**
 * 阻塞任务信息
 */
export interface BlockedTask {
  taskId: number
  title: string
  status?: string
}

/**
 * API 错误对象接口
 */
export interface ApiError extends Error {
  status?: number
  messageKey?: string
  reason?: string
  blockedTasks?: BlockedTask[]
}

/**
 * 类型守卫：判断是否为 ApiError
 */
export function isApiError(err: unknown): err is ApiError {
  return err instanceof Error && ('status' in err || 'messageKey' in err || 'reason' in err)
}

/**
 * 从错误对象中获取本地化消息。
 * 优先用 messageKey 对应的 i18n 翻译，没有则降级为后端原始英文消息。
 *
 * @param err - Error 对象（可能含 messageKey 字段）
 * @param t   - i18n 翻译函数
 * @returns 本地化错误消息
 */
export function getLocalizedErrorMessage(
  err: unknown,
  t: (key: string) => string,
): string {
  if (isApiError(err) && err.messageKey) {
    return t(err.messageKey)
  }
  if (err instanceof Error) {
    return err.message
  }
  return String(err)
}
