'use client'

import { Box, Text, HStack, Badge } from '@chakra-ui/react'
import { DiffHunk, DiffHunkLine } from '@/lib/diff'
import { useI18n } from '@/contexts/I18nContext'
import { useVirtualizer } from '@tanstack/react-virtual'
import { useRef, useMemo } from 'react'

interface SplitDiffViewProps {
  filename: string
  status: string
  additions: number
  deletions: number
  hunks: DiffHunk[]
}

interface SplitLine {
  oldLine?: number
  newLine?: number
  oldContent?: string
  newContent?: string
  type: 'add' | 'delete' | 'context' | 'both'
}

function toSplitLines(hunks: DiffHunk[]): SplitLine[] {
  const result: SplitLine[] = []

  for (const hunk of hunks) {
    let i = 0
    while (i < hunk.lines.length) {
      const line = hunk.lines[i]

      if (line.type === 'context') {
        result.push({ oldLine: line.oldLine, newLine: line.newLine, oldContent: line.content, newContent: line.content, type: 'both' })
        i++
      } else if (line.type === 'delete') {
        const deletes: DiffHunkLine[] = []
        const adds: DiffHunkLine[] = []
        while (i < hunk.lines.length && hunk.lines[i].type === 'delete') {
          deletes.push(hunk.lines[i])
          i++
        }
        while (i < hunk.lines.length && hunk.lines[i].type === 'add') {
          adds.push(hunk.lines[i])
          i++
        }
        const maxLen = Math.max(deletes.length, adds.length)
        for (let j = 0; j < maxLen; j++) {
          result.push({
            oldLine: deletes[j]?.oldLine,
            newLine: adds[j]?.newLine,
            oldContent: deletes[j]?.content,
            newContent: adds[j]?.content,
            type: deletes[j] && adds[j] ? 'both' : deletes[j] ? 'delete' : 'add',
          })
        }
      } else if (line.type === 'add') {
        result.push({ newLine: line.newLine, newContent: line.content, type: 'add' })
        i++
      }
    }
  }

  return result
}

function LineNumber({ num }: { num?: number }) {
  return (
    <Box
      as="span"
      w="40px"
      display="inline-block"
      textAlign="right"
      pr={3}
      color="myGray.400"
      fontSize="12px"
      fontFamily="mono"
      userSelect="none"
      flexShrink={0}
    >
      {num ?? ''}
    </Box>
  )
}

function DiffLineContent({ content, type }: { content?: string; type: string }) {
  const bg = type === 'add' ? '#e6ffec' : type === 'delete' ? '#ffebe9' : 'transparent'
  const indicator = type === 'add' ? '+' : type === 'delete' ? '-' : ' '

  return (
    <Box
      as="pre"
      flex={1}
      m={0}
      px={2}
      py={0}
      bg={bg}
      fontSize="12px"
      fontFamily="mono"
      whiteSpace="pre"
      overflow="hidden"
      textOverflow="ellipsis"
    >
      <Box as="span" color={type === 'add' ? '#1a7f37' : type === 'delete' ? '#cf222e' : 'myGray.400'} mr={1}>
        {indicator}
      </Box>
      {content ?? ''}
    </Box>
  )
}

export function SplitDiffView({ filename, status, additions, deletions, hunks }: SplitDiffViewProps) {
  const { t } = useI18n()
  const splitLines = useMemo(() => toSplitLines(hunks), [hunks])
  const parentRef = useRef<HTMLDivElement>(null)

  const virtualizer = useVirtualizer({
    count: splitLines.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 24, // 每行高度约 24px
    overscan: 10, // 预渲染 10 行
  })

  const statusLabel = status === 'added' ? t('diff.added') : status === 'deleted' ? t('diff.deleted') : t('diff.modified')
  const statusColor = status === 'added' ? 'green' : status === 'deleted' ? 'red' : 'blue'

  return (
    <Box borderWidth="1px" borderRadius="md" borderColor="myGray.200" overflow="hidden" bg="white">
      {/* File header */}
      <Box px={4} py={2} borderBottom="1px solid" borderColor="myGray.200" bg="myGray.50">
        <HStack justify="space-between">
          <HStack spacing={2}>
            <Text fontSize="sm" fontFamily="mono" fontWeight="medium" color="myGray.900">
              {filename}
            </Text>
            <Badge colorScheme={statusColor} variant="subtle" fontSize="xs">
              {statusLabel}
            </Badge>
          </HStack>
          <HStack spacing={2}>
            {additions > 0 && (
              <Badge colorScheme="green" fontSize="xs">+{additions}</Badge>
            )}
            {deletions > 0 && (
              <Badge colorScheme="red" fontSize="xs">-{deletions}</Badge>
            )}
          </HStack>
        </HStack>
      </Box>

      {/* Diff content with virtualization */}
      <Box ref={parentRef} overflowX="auto" overflowY="auto" maxH="600px">
        <Box
          style={{
            height: `${virtualizer.getTotalSize()}px`,
            width: '100%',
            position: 'relative',
          }}
        >
          {virtualizer.getVirtualItems().map((virtualRow) => {
            const line = splitLines[virtualRow.index]
            const leftType = line.type === 'delete' ? 'delete' : line.type === 'both' && line.oldContent !== line.newContent ? 'delete' : 'context'
            const rightType = line.type === 'add' ? 'add' : line.type === 'both' && line.oldContent !== line.newContent ? 'add' : 'context'
            const leftBg = leftType === 'delete' ? '#ffebe9' : '#f6f8fa'
            const rightBg = rightType === 'add' ? '#e6ffec' : '#f6f8fa'

            return (
              <Box
                key={virtualRow.index}
                display="flex"
                borderBottom="1px solid"
                borderColor="myGray.100"
                style={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  width: '100%',
                  height: `${virtualRow.size}px`,
                  transform: `translateY(${virtualRow.start}px)`,
                }}
              >
                {/* Left side (old) */}
                <Box flex={1} display="flex" bg={leftBg} minW="0">
                  <LineNumber num={line.oldLine} />
                  <DiffLineContent content={line.oldContent} type={leftType} />
                </Box>
                {/* Divider */}
                <Box w="1px" bg="myGray.200" flexShrink={0} />
                {/* Right side (new) */}
                <Box flex={1} display="flex" bg={rightBg} minW="0">
                  <LineNumber num={line.newLine} />
                  <DiffLineContent content={line.newContent} type={rightType} />
                </Box>
              </Box>
            )
          })}
        </Box>
      </Box>
    </Box>
  )
}
