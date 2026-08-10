'use client'

import { createContext, useContext, useEffect, useRef, useState, useCallback } from 'react'
import { useCurrentUser } from './UserContext'
import { API_BASE } from '@/lib/api'

type MessageHandler = (payload: any) => void

type WebSocketContextValue = {
  isConnected: boolean
  subscribe: (type: string, handler: MessageHandler) => () => void
}

const WebSocketContext = createContext<WebSocketContextValue | null>(null)

// http(s):// → ws(s)://
function toWsUrl(base: string): string {
  // 如果 base 为空（使用相对路径），则使用当前页面的 origin
  if (!base) {
    if (typeof window !== 'undefined') {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      return `${protocol}//${window.location.host}/api/v1/ws`
    }
    return 'ws://localhost:3001/api/v1/ws'
  }
  return base.replace(/^http/, 'ws') + '/ws'
}

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const { user } = useCurrentUser()
  const [isConnected, setIsConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const pingIntervalRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined)
  const retryCountRef = useRef(0)
  // handlers 用 ref 维护，避免重连时丢失订阅
  const handlersRef = useRef<Map<string, Set<MessageHandler>>>(new Map())

  const subscribe = useCallback((type: string, handler: MessageHandler) => {
    if (!handlersRef.current.has(type)) {
      handlersRef.current.set(type, new Set())
    }
    handlersRef.current.get(type)!.add(handler)
    return () => {
      handlersRef.current.get(type)?.delete(handler)
    }
  }, [])

  // connect as function declaration to avoid TDZ issues in onclose callback
  function connect() {
    if (!user) return
    // 避免重复连接
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) return

    const ws = new WebSocket(toWsUrl(API_BASE))
    wsRef.current = ws

    ws.onopen = () => {
      retryCountRef.current = 0
      setIsConnected(true)
      // 30s ping 心跳，保活
      pingIntervalRef.current = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ action: 'ping' }))
        }
      }, 30000)
    }

    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data)
        const handlers = handlersRef.current.get(msg.type)
        if (handlers) {
          handlers.forEach((h) => h(msg.payload))
        }
      } catch {
        // 忽略非 JSON / 格式错误的消息
      }
    }

    ws.onclose = () => {
      setIsConnected(false)
      if (pingIntervalRef.current) clearInterval(pingIntervalRef.current)
      // 指数退避重连：1s → 2s → 5s → 10s，封顶 30s
      const delays = [1000, 2000, 5000, 10000, 30000]
      const delay = delays[Math.min(retryCountRef.current, delays.length - 1)]
      retryCountRef.current++
      reconnectTimeoutRef.current = setTimeout(connect, delay)
    }

    ws.onerror = () => {
      // 让 onclose 处理重连
      ws.close()
    }
  }

  useEffect(() => {
    if (!user) {
      // 未登录：断开并清理
      if (wsRef.current) {
        wsRef.current.onclose = null // 避免触发重连
        wsRef.current.close()
        wsRef.current = null
      }
      setIsConnected(false)
      return
    }
    connect()
    return () => {
      if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current)
      if (pingIntervalRef.current) clearInterval(pingIntervalRef.current)
      if (wsRef.current) {
        wsRef.current.onclose = null
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [user])

  return (
    <WebSocketContext.Provider value={{ isConnected, subscribe }}>
      {children}
    </WebSocketContext.Provider>
  )
}

export function useWebSocket(type: string, handler: MessageHandler) {
  const ctx = useContext(WebSocketContext)
  if (!ctx) {
    throw new Error('useWebSocket must be used within WebSocketProvider')
  }
  // handler 用 ref 避免订阅反复注册/注销
  const handlerRef = useRef(handler)
  handlerRef.current = handler
  useEffect(() => {
    const wrapped = (payload: any) => handlerRef.current(payload)
    return ctx.subscribe(type, wrapped)
  }, [type, ctx])
}

export function useWebSocketStatus() {
  const ctx = useContext(WebSocketContext)
  if (!ctx) {
    throw new Error('useWebSocketStatus must be used within WebSocketProvider')
  }
  return ctx.isConnected
}
