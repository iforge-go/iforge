'use client'

import React, { useState, useRef, useEffect, useCallback } from 'react'
import { Box, Textarea, HStack, Avatar, Text, Spinner } from '@chakra-ui/react'
import NextLink from 'next/link'
import { api } from '@/lib/api'
import { UserSearchResult } from '@/lib/types'

// ============================================================
// useMention — 可复用的 @提及联想 hook
// ============================================================

interface UseMentionOptions {
  value: string
  onChange: (value: string) => void
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
  /** Ctrl/Cmd+Enter 时触发提交 */
  onSubmit?: () => void
}

/**
 * useMention — 在任意 Textarea 上添加 @用户联想功能。
 *
 * 使用方式：
 * ```tsx
 * const mention = useMention({ value, onChange, textareaRef })
 * <Textarea
 *   ref={textareaRef}
 *   onChange={(e) => { onChange(e.target.value); mention.detectMention(e.target.value, e.target.selectionStart) }}
 *   onBlur={mention.handleBlur}
 *   onKeyDown={mention.handleKeyDown}
 * />
 * {mention.showDropdown && <MentionDropdown {...mention} />}
 * ```
 */
export function useMention({ value, onChange, textareaRef, onSubmit }: UseMentionOptions) {
  const [mention, setMention] = useState<{ active: boolean; query: string; start: number }>({
    active: false,
    query: '',
    start: -1,
  })
  const [users, setUsers] = useState<UserSearchResult[]>([])
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [loading, setLoading] = useState(false)

  // 从光标位置检测 @提及输入
  const detectMention = useCallback((text: string, cursorPos: number) => {
    const textBeforeCursor = text.substring(0, cursorPos)
    // @ 后跟字母数字_-，位于行首或非字字符之后
    const match = textBeforeCursor.match(/(?:^|[^a-zA-Z0-9_])@([a-zA-Z0-9_-]*)$/)
    if (match) {
      const atPos = cursorPos - match[0].length + (match[0][0] === '@' ? 0 : 1)
      setMention({ active: true, query: match[1], start: atPos })
    } else {
      setMention((prev) => (prev.active ? { active: false, query: '', start: -1 } : prev))
    }
  }, [])

  const insertMention = useCallback((username: string) => {
    // 直接从闭包读取 mention.start，不放在 setMention updater 内部，
    // 避免 updater（render 阶段）中调用 onChange 触发父组件 setState
    const before = value.substring(0, mention.start)
    const cursorPos = textareaRef.current?.selectionStart ?? value.length
    const after = value.substring(cursorPos)
    const insertText = `@${username} `
    onChange(before + insertText + after)
    setMention({ active: false, query: '', start: -1 })
    setUsers([])
    // 恢复焦点，光标移到插入内容之后
    setTimeout(() => {
      const newPos = before.length + insertText.length
      textareaRef.current?.focus()
      textareaRef.current?.setSelectionRange(newPos, newPos)
    }, 0)
  }, [value, onChange, textareaRef, mention.start])

  const handleBlur = useCallback(() => {
    // 延迟关闭，让 dropdown 的 mousedown 先触发
    setTimeout(() => {
      setMention({ active: false, query: '', start: -1 })
    }, 150)
  }, [])

  // 防抖搜索用户
  useEffect(() => {
    if (!mention.active || !mention.query) {
      setUsers([])
      return
    }
    setLoading(true)
    const timer = setTimeout(async () => {
      try {
        const result = await api.searchUsers(mention.query, 5)
        setUsers(result.users || [])
        setSelectedIndex(0)
      } catch {
        setUsers([])
      } finally {
        setLoading(false)
      }
    }, 200)
    return () => clearTimeout(timer)
  }, [mention.active, mention.query])

  const handleKeyDown = useCallback((e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (mention.active && users.length > 0) {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setSelectedIndex((prev) => (prev + 1) % users.length)
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        setSelectedIndex((prev) => (prev - 1 + users.length) % users.length)
        return
      }
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        insertMention(users[selectedIndex].userName)
        return
      }
      if (e.key === 'Escape') {
        e.preventDefault()
        setMention({ active: false, query: '', start: -1 })
        return
      }
    }
    // Ctrl/Cmd+Enter 提交
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault()
      onSubmit?.()
    }
  }, [mention.active, users, selectedIndex, insertMention, onSubmit])

  return {
    showDropdown: mention.active && (users.length > 0 || loading),
    users,
    selectedIndex,
    loading,
    detectMention,
    handleKeyDown,
    handleBlur,
    insertMention,
    setSelectedIndex,
  }
}

// ============================================================
// MentionDropdown — @提及联想下拉列表
// ============================================================

interface MentionDropdownProps {
  users: UserSearchResult[]
  selectedIndex: number
  loading: boolean
  onSelect: (username: string) => void
  onHover: (index: number) => void
}

export function MentionDropdown({ users, selectedIndex, loading, onSelect, onHover }: MentionDropdownProps) {
  return (
    <Box
      position="absolute"
      bottom="100%"
      left={0}
      mb={1}
      bg="white"
      border="1px solid"
      borderColor="myGray.200"
      borderRadius="md"
      boxShadow="lg"
      zIndex={20}
      maxH="240px"
      overflowY="auto"
      w="100%"
    >
      {loading ? (
        <HStack justify="center" py={3}>
          <Spinner size="sm" />
        </HStack>
      ) : (
        users.map((user, index) => (
          <Box
            key={user.userName}
            px={3}
            py={2}
            cursor="pointer"
            bg={index === selectedIndex ? 'blue.50' : 'white'}
            _hover={{ bg: 'blue.50' }}
            onMouseDown={(e) => {
              e.preventDefault() // 阻止 textarea blur
              onSelect(user.userName)
            }}
            onMouseEnter={() => onHover(index)}
          >
            <HStack spacing={2}>
              <Avatar size="xs" name={user.fullName || user.userName} src={user.image || undefined} />
              <Box minW={0}>
                <Text fontSize="sm" fontWeight="medium" noOfLines={1}>
                  {user.fullName || user.userName}
                </Text>
                <Text fontSize="xs" color="myGray.500">
                  @{user.userName}
                </Text>
              </Box>
            </HStack>
          </Box>
        ))
      )}
    </Box>
  )
}

// ============================================================
// MentionTextarea — 带有 @用户联想的 Textarea（封装 useMention + MentionDropdown）
// ============================================================

interface MentionTextareaProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  rows?: number
  size?: 'sm' | 'md' | 'lg'
  /** Ctrl/Cmd+Enter 时触发提交 */
  onSubmit?: () => void
}

export default function MentionTextarea({
  value,
  onChange,
  placeholder,
  rows = 3,
  size = 'sm',
  onSubmit,
}: MentionTextareaProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const mention = useMention({ value, onChange, textareaRef, onSubmit })

  return (
    <Box position="relative">
      <Textarea
        ref={textareaRef}
        value={value}
        onChange={(e) => {
          onChange(e.target.value)
          mention.detectMention(e.target.value, e.target.selectionStart)
        }}
        onBlur={mention.handleBlur}
        onKeyDown={mention.handleKeyDown}
        placeholder={placeholder}
        rows={rows}
        size={size}
      />
      {mention.showDropdown && (
        <MentionDropdown
          users={mention.users}
          selectedIndex={mention.selectedIndex}
          loading={mention.loading}
          onSelect={mention.insertMention}
          onHover={mention.setSelectedIndex}
        />
      )}
    </Box>
  )
}

// ============================================================
// renderWithMentions — @提及高亮渲染（用于评论显示）
// ============================================================

const mentionHighlightRegex = /(^|[^a-zA-Z0-9_])@([a-zA-Z][a-zA-Z0-9_-]*)/g

/**
 * renderWithMentions — 将文本中的 @username 渲染为高亮 span + 链接。
 * 与后端 mentionRegex 保持一致的正则口径。
 */
export function renderWithMentions(content: string): React.ReactNode[] {
  const parts: React.ReactNode[] = []
  let lastIndex = 0
  let key = 0

  mentionHighlightRegex.lastIndex = 0
  let match: RegExpExecArray | null
  while ((match = mentionHighlightRegex.exec(content)) !== null) {
    if (match.index > lastIndex) {
      parts.push(content.substring(lastIndex, match.index))
    }
    if (match[1]) {
      parts.push(match[1])
    }
    parts.push(
      <NextLink key={`mention-${key++}`} href={`/${match[2]}`}>
        <Text
          as="span"
          color="blue.600"
          fontWeight="medium"
          bg="blue.50"
          borderRadius="sm"
          px={0.5}
          cursor="pointer"
          _hover={{ bg: 'blue.100', textDecoration: 'underline' }}
        >
          @{match[2]}
        </Text>
      </NextLink>
    )
    lastIndex = match.index + match[0].length
  }
  if (lastIndex < content.length) {
    parts.push(content.substring(lastIndex))
  }
  return parts
}

/**
 * highlightMentionsInChildren — 处理 React children 中的文本节点，
 * 将 @username 高亮。用于 MarkdownRenderer 自定义组件渲染。
 *
 * 仅处理字符串类型的 children，其他 React 元素（如 <strong>、<code>）保持不变。
 */
export function highlightMentionsInChildren(children: React.ReactNode): React.ReactNode {
  if (typeof children === 'string') {
    return renderWithMentions(children)
  }
  if (Array.isArray(children)) {
    return children.map((child, i) =>
      typeof child === 'string'
        ? <React.Fragment key={i}>{renderWithMentions(child)}</React.Fragment>
        : child
    )
  }
  return children
}
