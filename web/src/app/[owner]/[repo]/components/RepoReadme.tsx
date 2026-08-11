'use client'

import {
  Box,
  HStack,
  Icon,
  IconButton,
  Heading,
} from '@chakra-ui/react'
import Link from 'next/link'
import Image from 'next/image'
import { useState } from 'react'
import {
  FiFile,
  FiEdit,
  FiList,
} from 'react-icons/fi'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeRaw from 'rehype-raw'
import { useI18n } from '@/contexts/I18nContext'
import { extractHeadings, slugify } from '@/components/ReadmeToc'
import { ReadmeTocOverlay } from '@/components/ReadmeTocOverlay'
import { preprocessMarkdown } from '@/lib/markdown'

function extractText(children: any): string {
  if (typeof children === 'string') return children
  if (Array.isArray(children)) return children.map(extractText).join('')
  if (children?.props?.children) return extractText(children.props.children)
  return ''
}

interface RepoReadmeProps {
  owner: string
  repoName: string
  selectedBranch: string
  readmeContent: string
  readmeRaw: string
  canEditFile: boolean
}

export function RepoReadme({
  owner,
  repoName,
  selectedBranch,
  readmeContent,
  readmeRaw,
  canEditFile,
}: RepoReadmeProps) {
  const { t } = useI18n()
  const [showToc, setShowToc] = useState(false)

  if (!readmeContent) return null

  return (
    <Box position="relative">
      <Box
        borderWidth="1px"
        borderRadius="lg"
        overflow="hidden"
        bg="white"
      >
        <Box py={2} px={4} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
          <HStack justify="space-between">
            <HStack>
              <Icon as={FiFile} color="primary.500" />
              <Heading size="sm">README</Heading>
            </HStack>
            <HStack spacing={2}>
              {canEditFile && (
                <IconButton
                  as={Link}
                  aria-label={t('repo.edit')}
                  icon={<Icon as={FiEdit} />}
                  href={`/${owner}/${repoName}/edit/${selectedBranch}/README.md`}
                  size="sm"
                  variant="ghost"
                  color="myGray.500"
                  _hover={{ bg: 'myGray.100' }}
                />
              )}
              {extractHeadings(readmeRaw).length > 0 && (
                <IconButton
                  aria-label={t('repo.toc')}
                  icon={<Icon as={FiList} />}
                  size="sm"
                  variant="ghost"
                  onClick={() => setShowToc(!showToc)}
                  color={showToc ? 'primary.600' : 'myGray.500'}
                  _hover={{ bg: 'myGray.100' }}
                  display={{ base: 'none', lg: 'inline-flex' }}
                />
              )}
            </HStack>
          </HStack>
        </Box>
        <Box p={6}>
          <Box
            className="markdown-body"
            sx={{
              '& h1': { fontSize: '2xl', fontWeight: 'bold', mb: 4, scrollMarginTop: '80px' },
              '& h2': { fontSize: 'xl', fontWeight: 'bold', mb: 3, mt: 6, scrollMarginTop: '80px' },
              '& h3': { fontSize: 'lg', fontWeight: 'bold', mb: 2, mt: 4, scrollMarginTop: '80px' },
              '& h4': { fontSize: 'md', fontWeight: 'bold', mb: 2, mt: 4, scrollMarginTop: '80px' },
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
              // 通过 preprocessMarkdown 将连续 badges 合并到同一行
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
              rehypePlugins={[rehypeRaw]}
              components={{
                h1: ({ children }) => <h1 id={`readme-heading-${slugify(extractText(children))}`}>{children}</h1>,
                h2: ({ children }) => <h2 id={`readme-heading-${slugify(extractText(children))}`}>{children}</h2>,
                h3: ({ children }) => <h3 id={`readme-heading-${slugify(extractText(children))}`}>{children}</h3>,
                h4: ({ children }) => <h4 id={`readme-heading-${slugify(extractText(children))}`}>{children}</h4>,
                img: ({ src, alt }) => {
                  if (!src) return null
                  if (typeof src === 'string' && (src.startsWith('http://') || src.startsWith('https://'))) {
                    return <Image src={src} alt={alt || ''} style={{ maxWidth: '100%', height: 'auto' }} fill={false} />
                  }
                  const srcStr = typeof src === 'string' ? src : ''
                  // 使用 GitHub-style raw URL: /:owner/:repo/raw/:ref/*
                  const rawUrl = `/${owner}/${repoName}/raw/${selectedBranch}/${srcStr}`
                  return <Image src={rawUrl} alt={alt || ''} style={{ maxWidth: '100%', height: 'auto' }} fill={false} />
                }
              }}
            >
              {preprocessMarkdown(readmeContent)}
            </ReactMarkdown>
          </Box>
        </Box>
      </Box>
      {showToc && extractHeadings(readmeRaw).length > 0 && (
        <ReadmeTocOverlay
          headings={extractHeadings(readmeRaw)}
          onClose={() => setShowToc(false)}
        />
      )}
    </Box>
  )
}
