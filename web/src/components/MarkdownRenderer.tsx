'use client'

import dynamic from 'next/dynamic'
import { Skeleton } from '@chakra-ui/react'

// 动态导入 MarkdownRenderer，减少首屏加载体积
// react-markdown + remark-gfm + rehype-raw + rehype-sanitize 合计约 100KB+
const MarkdownRendererInternal = dynamic(
  () => import('./MarkdownRendererInternal').then(mod => ({ default: mod.MarkdownRendererInternal })),
  {
    loading: () => <Skeleton height="200px" borderRadius="md" />,
    ssr: false,
  }
)

interface MarkdownRendererProps {
  content: string
  /** When set, relative image src in markdown are rewritten to `${imageBaseUrl}/${src}` */
  imageBaseUrl?: string
}

export function MarkdownRenderer({ content, imageBaseUrl }: MarkdownRendererProps) {
  return <MarkdownRendererInternal content={content} imageBaseUrl={imageBaseUrl} />
}
