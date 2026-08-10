'use client'

import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeRaw from 'rehype-raw'
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize'
import { Box, Text } from '@chakra-ui/react'
import { highlightMentionsInChildren } from './MentionTextarea'
import { preprocessMarkdown } from '@/lib/markdown'

// 扩展 rehype-sanitize 默认 schema：
// - defaultSchema 基于标准 HTML sanitization filter，已禁用 script/事件处理器/javascript: 协议
// - 额外允许 class 属性（语法高亮、@提及高亮等需要）
// - 允许 align 属性（README 中 <div align="center"> 常用）
const sanitizeSchema = {
  ...defaultSchema,
  attributes: {
    ...defaultSchema.attributes,
    '*': [...(defaultSchema.attributes?.['*'] || []), 'className', 'align'],
  },
}

interface MarkdownRendererProps {
  content: string
  /** When set, relative image src in markdown are rewritten to `${imageBaseUrl}/${src}` */
  imageBaseUrl?: string
}

export function MarkdownRendererInternal({ content, imageBaseUrl }: MarkdownRendererProps) {
  if (!content || content.trim() === '') {
    return <Text color="myGray.500" fontSize="sm">暂无内容</Text>
  }

  return (
    <Box
      className="markdown-body"
      sx={{
        '& h1': { fontSize: '2xl', fontWeight: 'bold', mb: 4 },
        '& h2': { fontSize: 'xl', fontWeight: 'bold', mb: 3, mt: 6 },
        '& h3': { fontSize: 'lg', fontWeight: 'bold', mb: 2, mt: 4 },
        '& h4': { fontSize: 'md', fontWeight: 'bold', mb: 2, mt: 4 },
        '& p': { mb: 3, lineHeight: 'tall' },
        '& ul, & ol': { pl: 6, mb: 3 },
        '& li': { mb: 1 },
        '& code': {
          bg: 'gray.100',
          px: 1.5,
          py: 0.5,
          borderRadius: 'md',
          fontSize: 'sm'
        },
        '& pre': {
          bg: 'gray.50',
          p: 4,
          borderRadius: 'md',
          overflowX: 'auto',
          mb: 3
        },
        '& a': { color: 'primary.600', textDecoration: 'underline' },
        '& blockquote': {
          borderLeft: '4px solid',
          borderColor: 'myGray.200',
          pl: 4,
          ml: 0,
          color: 'gray.600',
          fontStyle: 'italic'
        },
        '& table': {
          width: '100%',
          mb: 3,
          borderCollapse: 'collapse'
        },
        '& th, & td': {
          border: '1px solid',
          borderColor: 'myGray.200',
          p: 2
        },
        '& th': { bg: 'gray.50', fontWeight: 'bold' },
        // 处理 align="center" 容器内的内容（如徽章横向排列）
        '& [align="center"]': {
          textAlign: 'center',
          '& p': {
            display: 'block',
            mb: 2,
          },
          '& img': {
            display: 'inline-block',
            verticalAlign: 'middle',
            mx: 0.5,
            my: 0.5,
          },
          '& h1, & h2, & h3, & h4, & h5, & h6': {
            mb: 2,
          },
        },
      }}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        // 顺序关键：rehype-raw 先把用户输入的原始 HTML 解析进 HAST，
        // rehype-sanitize 随后按白名单过滤，剥离 script/事件处理器/javascript: 协议，
        // 防止 Issue/MR/Wiki 评论中的 XSS 攻击
        rehypePlugins={[[rehypeRaw], [rehypeSanitize, sanitizeSchema]]}
        components={{
          // 在段落和列表项中高亮 @username
          p: ({ children, ...props }) => <p {...props}>{highlightMentionsInChildren(children)}</p>,
          li: ({ children, ...props }) => <li {...props}>{highlightMentionsInChildren(children)}</li>,
          img: ({ src, alt }) => {
            if (!src) return null
            // 绝对 URL 直接使用
            if (typeof src === 'string' && (src.startsWith('http://') || src.startsWith('https://'))) {
              return <img src={src} alt={alt || ''} style={{ maxWidth: '100%', height: 'auto' }} />
            }
            // 相对路径：如果有 imageBaseUrl，则拼接
            const srcStr = typeof src === 'string' ? src : ''
            const finalSrc = imageBaseUrl ? `${imageBaseUrl}/${srcStr}` : srcStr
            return <img src={finalSrc} alt={alt || ''} style={{ maxWidth: '100%', height: 'auto' }} />
          }
        }}
      >
        {preprocessMarkdown(content)}
      </ReactMarkdown>
    </Box>
  )
}
