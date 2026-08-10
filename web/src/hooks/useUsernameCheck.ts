'use client'

import { useState, useEffect } from 'react'
import { api } from '@/lib/api'

// 用户名校验状态机：
// idle     — 空/未触发
// checking — 防抖中或请求中
// available— 可用
// invalid  — 格式不合规（3-30 位 [a-zA-Z0-9_-]）
// reserved — 命中系统保留词
// taken    — 已被占用
export type UsernameStatus = 'idle' | 'checking' | 'available' | 'invalid' | 'reserved' | 'taken'

// 与后端 isValidUsername 一致：3-30 位，字母/数字/连字符/下划线
const USERNAME_RE = /^[a-zA-Z0-9_-]+$/
const isFormatValid = (u: string) => USERNAME_RE.test(u) && u.length >= 3 && u.length <= 30

/**
 * 用户名实时可用性校验。
 * 格式不合规本地即时判断；格式 OK 则 400ms 防抖后调后端 check-username。
 * 用于注册页 / 初始化向导 / 管理员新建用户三处共享。
 *
 * @param username 受控用户名值
 * @returns 当前校验状态
 */
export function useUsernameCheck(username: string): UsernameStatus {
  const [status, setStatus] = useState<UsernameStatus>('idle')

  useEffect(() => {
    const u = username.trim()
    if (!u) {
      setStatus('idle')
      return
    }
    // 格式不合规：本地立即反馈，不发请求
    if (!isFormatValid(u)) {
      setStatus('invalid')
      return
    }
    setStatus('checking')
    let cancelled = false
    const timer = setTimeout(async () => {
      try {
        const res = await api.checkUsername(u)
        if (cancelled) return
        if (res.available) {
          setStatus('available')
        } else {
          // reason ∈ invalid | reserved | taken，未识别统一降级 invalid
          setStatus((res.reason as UsernameStatus) || 'invalid')
        }
      } catch {
        // 网络异常不阻塞输入，交由提交时再校验
        if (!cancelled) setStatus('idle')
      }
    }, 400)
    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [username])

  return status
}
