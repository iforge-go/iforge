'use client'

import dynamic from 'next/dynamic'
import { Skeleton } from '@chakra-ui/react'

// 动态导入 MarkdownRenderer，减少首屏加载体积
// react-markdown + remark-gfm + rehype-raw + rehype-sanitize 合计约 100KB+
const MarkdownRendererInternal = dynamic(() => import('./MarkdownRenderer').then(mod => ({ default: mod.MarkdownRenderer })), {
  loading: () => <Skeleton height="200px" borderRadius="md" />,
  ssr: false,
})

interface DynamicMarkdownRendererProps {
  content: string
}

export function DynamicMarkdownRenderer({ content }: DynamicMarkdownRendererProps) {
  return <MarkdownRendererInternal content={content} />
}
