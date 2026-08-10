'use client'

import { useState, useRef, useCallback } from 'react'
import {
  Box, Textarea, HStack, Button, Text, IconButton,
  Divider, useColorModeValue
} from '@chakra-ui/react'
import {
  FiBold, FiItalic, FiHash, FiList, FiLink,
  FiCode, FiCheckSquare, FiMessageSquare
} from 'react-icons/fi'
import { MarkdownRenderer } from './MarkdownRenderer'
import { useI18n } from '@/contexts/I18nContext'
import { useMention, MentionDropdown } from './MentionTextarea'

interface MarkdownEditorProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  rows?: number
  height?: number
}

export function MarkdownEditor({
  value,
  onChange,
  placeholder,
  rows = 10,
  height
}: MarkdownEditorProps) {
  const { t } = useI18n()
  const [activeTab, setActiveTab] = useState<'write' | 'preview'>('write')
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const mention = useMention({ value, onChange, textareaRef })

  const insertText = useCallback((before: string, after: string = '') => {
    const textarea = textareaRef.current
    if (!textarea) return

    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const selected = value.substring(start, end)
    const newText = value.substring(0, start) + before + selected + after + value.substring(end)
    onChange(newText)

    setTimeout(() => {
      textarea.focus()
      const cursorPos = start + before.length + selected.length
      textarea.setSelectionRange(cursorPos, cursorPos)
    }, 0)
  }, [value, onChange])

  const insertLinePrefix = useCallback((prefix: string) => {
    const textarea = textareaRef.current
    if (!textarea) return

    const start = textarea.selectionStart
    const lineStart = value.lastIndexOf('\n', start - 1) + 1
    const newText = value.substring(0, lineStart) + prefix + value.substring(lineStart)
    onChange(newText)

    setTimeout(() => {
      textarea.focus()
      textarea.setSelectionRange(start + prefix.length, start + prefix.length)
    }, 0)
  }, [value, onChange])

  const toolbarBg = useColorModeValue('gray.50', 'gray.700')
  const borderColor = useColorModeValue('gray.200', 'gray.600')

  const toolbarBtn = (
    icon: React.ReactElement,
    title: string,
    action: () => void
  ) => (
    <IconButton
      aria-label={title}
      icon={icon}
      size="sm"
      variant="ghost"
      title={title}
      onClick={action}
      _hover={{ bg: 'gray.200' }}
    />
  )

  return (
    <Box borderWidth="1px" borderRadius="md" borderColor={borderColor}>
      {/* Toolbar + Tabs in one row */}
      <HStack
        spacing={1}
        px={2}
        py={1}
        bg={toolbarBg}
        borderBottom="1px solid"
        borderColor={borderColor}
        borderTopRadius="md"
        flexWrap="wrap"
      >
        <Button
          size="sm"
          variant={activeTab === 'write' ? 'solid' : 'ghost'}
          colorScheme={activeTab === 'write' ? 'primary' : undefined}
          borderRadius="md"
          onClick={() => setActiveTab('write')}
        >
          {t('editor.write')}
        </Button>
        <Button
          size="sm"
          variant={activeTab === 'preview' ? 'solid' : 'ghost'}
          colorScheme={activeTab === 'preview' ? 'primary' : undefined}
          borderRadius="md"
          onClick={() => setActiveTab('preview')}
        >
          {t('editor.preview')}
        </Button>

        <Box flex={1} />

        {toolbarBtn(<FiHash />, '标题', () => insertLinePrefix('### '))}
        {toolbarBtn(<FiBold />, '粗体', () => insertText('**', '**'))}
        {toolbarBtn(<FiItalic />, '斜体', () => insertText('*', '*'))}
        {toolbarBtn(<FiMessageSquare />, '引用', () => insertLinePrefix('> '))}
        {toolbarBtn(<FiCode />, '代码', () => insertText('`', '`'))}
        {toolbarBtn(<FiLink />, '链接', () => insertText('[', '](url)'))}
        <Divider orientation="vertical" h={4} mx={1} />
        {toolbarBtn(<FiList />, '无序列表', () => insertLinePrefix('- '))}
        {toolbarBtn(<FiCheckSquare />, '任务列表', () => insertLinePrefix('- [ ] '))}
      </HStack>

      {/* Content area */}
      {activeTab === 'write' ? (
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
            height={height ? `${height}px` : undefined}
            fontFamily="mono"
            fontSize="sm"
            border="none"
            borderRadius="none"
            resize="vertical"
            _focus={{ boxShadow: 'none' }}
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
      ) : (
        <Box p={4} minH={height ? `${height}px` : '200px'}>
          {value.trim() ? (
            <MarkdownRenderer content={value} />
          ) : (
            <Text color="myGray.500" fontSize="sm">{t('editor.nothingToPreview')}</Text>
          )}
        </Box>
      )}
    </Box>
  )
}
