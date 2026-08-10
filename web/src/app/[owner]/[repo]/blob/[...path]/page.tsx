'use client'

import {
  Box,
  Text,
  VStack,
  HStack,
  Icon,
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  Spinner,
  Button,
  Code,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useEffect, useState, useMemo } from 'react'
import { useParams } from 'next/navigation'
import hljs from 'highlight.js/lib/core'
import go from 'highlight.js/lib/languages/go'
import javascript from 'highlight.js/lib/languages/javascript'
import typescript from 'highlight.js/lib/languages/typescript'
import xml from 'highlight.js/lib/languages/xml'
import css from 'highlight.js/lib/languages/css'
import python from 'highlight.js/lib/languages/python'
import java from 'highlight.js/lib/languages/java'
import rust from 'highlight.js/lib/languages/rust'
import shell from 'highlight.js/lib/languages/shell'
import sql from 'highlight.js/lib/languages/sql'
import markdown from 'highlight.js/lib/languages/markdown'
import jsonLang from 'highlight.js/lib/languages/json'
import yaml from 'highlight.js/lib/languages/yaml'
import bash from 'highlight.js/lib/languages/bash'
import 'highlight.js/styles/github.css'
import { api, FileEntry, CommitInfo, SERVER_BASE } from '@/lib/api'
import { parseBranchAndPath, shouldWaitForBranches } from '@/lib/branchPath'
import { FiFile, FiChevronRight, FiGitBranch, FiEdit, FiCopy, FiDownload } from 'react-icons/fi'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useI18n } from '@/contexts/I18nContext'
import { useGithubToast } from '@/app/providers'
import { MarkdownRenderer } from '@/components/MarkdownRenderer'

// Register languages once
const registered = new Set<string>()
function registerLanguage(name: string, lang: any) {
  if (!registered.has(name)) {
    hljs.registerLanguage(name, lang)
    registered.add(name)
  }
}
registerLanguage('go', go)
registerLanguage('javascript', javascript)
registerLanguage('typescript', typescript)
registerLanguage('xml', xml)
registerLanguage('html', xml)
registerLanguage('css', css)
registerLanguage('python', python)
registerLanguage('java', java)
registerLanguage('rust', rust)
registerLanguage('shell', shell)
registerLanguage('sql', sql)
registerLanguage('markdown', markdown)
registerLanguage('json', jsonLang)
registerLanguage('yaml', yaml)
registerLanguage('bash', bash)

function getLanguageFromPath(filePath: string): string | null {
  const ext = filePath.split('.').pop()?.toLowerCase()
  const map: Record<string, string> = {
    go: 'go',
    js: 'javascript',
    jsx: 'javascript',
    mjs: 'javascript',
    ts: 'typescript',
    tsx: 'typescript',
    html: 'xml',
    htm: 'xml',
    xml: 'xml',
    svg: 'xml',
    css: 'css',
    scss: 'css',
    py: 'python',
    java: 'java',
    rs: 'rust',
    sh: 'shell',
    bash: 'bash',
    zsh: 'shell',
    sql: 'sql',
    md: 'markdown',
    markdown: 'markdown',
    json: 'json',
    yml: 'yaml',
    yaml: 'yaml',
  }
  return ext ? (map[ext] || null) : null
}

export default function BlobPage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const pathSegments = params.path as string[]

  const { branches, userRole } = useRepo()
  const { t } = useI18n()
  const toast = useGithubToast()

  const defaultBranch = branches.find(b => b.isDefault)?.name || branches[0]?.name || 'main'
  // 分支名可能含 "/"（如 feature/task-15），用已知分支列表反向匹配前缀来正确分离分支名和文件路径
  const { ref, subPath: filePath } = parseBranchAndPath(pathSegments, branches, defaultBranch)

  const [fileContent, setFileContent] = useState<string>('')
  const [fileEntry, setFileEntry] = useState<FileEntry | null>(null)
  const [commits, setCommits] = useState<CommitInfo[]>([])
  const [loading, setLoading] = useState(true)

  const canEdit = userRole === 'owner' || userRole === 'member'
  const branchesLoaded = branches.length > 0

  useEffect(() => {
    const loadData = async () => {
      // 多段路径可能含 "/" 分支名，需等 branches 加载后才能正确解析
      if (shouldWaitForBranches(pathSegments, branches)) return
      try {
        const [contentData, commitsData] = await Promise.all([
          api.getFileContent(owner, repoName, filePath, ref),
          api.listCommits(owner, repoName, ref, 1, 10).catch(() => []),
        ])

        setFileContent(contentData.content)
        setFileEntry({
          name: filePath.split('/').pop() || filePath,
          path: filePath,
          type: 'file',
          size: contentData.content.length,
          mode: '',
        })
        setCommits(commitsData || [])
      } catch (error) {
        console.error('Failed to load file:', error)
      } finally {
        setLoading(false)
      }
    }

    loadData()
  }, [owner, repoName, filePath, ref, branchesLoaded])

  const pathParts = filePath ? filePath.split('/').filter(Boolean) : []
  const breadcrumbs = [
    { label: repoName, href: `/${owner}/${repoName}` },
    ...pathParts.map((part, index) => ({
      label: part,
      href: index === pathParts.length - 1
        ? undefined
        : `/${owner}/${repoName}/tree/${ref}/${pathParts.slice(0, index + 1).join('/')}`,
    })),
  ]

  if (loading) {
    return (
      <VStack spacing={4} py={8}>
        <Spinner size="xl" />
        <Text>{t('common.loading')}</Text>
      </VStack>
    )
  }

  if (!fileEntry) {
    return (
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
        <Text>{t('repo.fileNotFound')}</Text>
      </Box>
    )
  }

  const isMarkdown = /\.(md|markdown)$/i.test(filePath)
  const isImage = /\.(png|jpg|jpeg|gif|svg|webp)$/i.test(filePath)

  // 复制文件原始内容到剪贴板（图片无文本内容，不显示复制按钮）
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(fileContent)
      toast({ title: t('repo.copied'), status: 'success', duration: 2000 })
    } catch {
      // 静默失败：clipboard API 仅在安全上下文（HTTPS/localhost）可用
    }
  }

  return (
    <VStack spacing={4} align="stretch">
      {/* Breadcrumb */}
      <Breadcrumb separator={<Icon as={FiChevronRight} color="myGray.400" />} fontSize="sm">
        {breadcrumbs.map((crumb, index) => (
          <BreadcrumbItem key={index} isCurrentPage={index === breadcrumbs.length - 1}>
            {crumb.href ? (
              <BreadcrumbLink as={Link} href={crumb.href} color="myGray.600">
                {crumb.label}
              </BreadcrumbLink>
            ) : (
              <BreadcrumbLink color="myGray.900" fontWeight="medium">
                {crumb.label}
              </BreadcrumbLink>
            )}
          </BreadcrumbItem>
        ))}
      </Breadcrumb>

      {/* File header */}
      <Box borderWidth="1px" borderColor="myGray.200" borderRadius="lg" overflow="hidden" bg="white">
        <Box p={4} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
          <HStack justify="space-between">
            <HStack spacing={2}>
              <Icon as={FiFile} color="myGray.600" />
              <Text fontWeight="medium">{fileEntry.name}</Text>
              <Text fontSize="sm" color="myGray.500">
                {fileEntry.size} bytes
              </Text>
            </HStack>
            <HStack spacing={2}>
              {!isImage && (
                <Button
                  size="sm"
                  variant="whiteBase"
                  leftIcon={<Icon as={FiCopy} />}
                  onClick={handleCopy}
                >
                  {t('repo.copy')}
                </Button>
              )}
              {canEdit && (
                <Link href={`/${owner}/${repoName}/edit/${ref}/${filePath}`}>
                  <Button size="sm" variant="whiteBase" leftIcon={<Icon as={FiEdit} />}>
                    {t('common.edit')}
                  </Button>
                </Link>
              )}
              <a href={`/${owner}/${repoName}/raw/${ref}/${filePath}`} target="_blank" rel="noopener noreferrer">
                <Button size="sm" variant="whiteBase" leftIcon={<Icon as={FiDownload} />}>
                  {t('repo.raw')}
                </Button>
              </a>
            </HStack>
          </HStack>
        </Box>

        {/* File content */}
        <Box p={4}>
          {isImage ? (
            <Box textAlign="center">
              <Box
                as="img"
                src={`/${owner}/${repoName}/raw/${ref}/${filePath}`}
                maxW="100%"
                h="auto"
                borderRadius="md"
              />
            </Box>
          ) : isMarkdown ? (
            <MarkdownRenderer content={fileContent} imageBaseUrl={`/${owner}/${repoName}/raw/${ref}`} />
          ) : (
            <HighlightedCode content={fileContent} filePath={filePath} />
          )}
        </Box>
      </Box>

      {/* Recent commits */}
      {commits.length > 0 && (
        <Box borderWidth="1px" borderColor="myGray.200" borderRadius="lg" overflow="hidden" bg="white">
          <Box p={4} borderBottom="1px" borderColor="myGray.200" bg="myGray.50">
            <Text fontWeight="medium">{t('repo.recentCommits')}</Text>
          </Box>
          <VStack spacing={0} align="stretch">
            {commits.slice(0, 5).map((commit, index) => (
              <HStack
                key={commit.id}
                p={3}
                borderBottom={index < 4 ? '1px' : 'none'}
                borderColor="myGray.200"
                _hover={{ bg: 'myGray.50' }}
              >
                <Text fontSize="sm" flex={1} noOfLines={1}>
                  {commit.message}
                </Text>
                <Text fontSize="xs" color="myGray.500">
                  {commit.author}
                </Text>
                <Code fontSize="xs" colorScheme="gray" px={2} py={0.5} borderRadius="md">
                  {commit.id.substring(0, 7)}
                </Code>
              </HStack>
            ))}
          </VStack>
        </Box>
      )}
    </VStack>
  )
}

function HighlightedCode({ content, filePath }: { content: string; filePath: string }) {
  const html = useMemo(() => {
    const lang = getLanguageFromPath(filePath)
    try {
      if (lang && hljs.getLanguage(lang)) {
        return hljs.highlight(content, { language: lang, ignoreIllegals: true }).value
      }
    } catch (e) {
      // fall through
    }
    return hljs.highlightAuto(content).value
  }, [content, filePath])

  return (
    <Box
      as="pre"
      p={4}
      borderRadius="md"
      bg="myGray.50"
      fontSize="sm"
      overflowX="auto"
      fontFamily="monospace"
      lineHeight="1.5"
      sx={{ 'code.hljs': { bg: 'transparent', p: 0 } }}
    >
      <code
        className="hljs"
        dangerouslySetInnerHTML={{ __html: html }}
      />
    </Box>
  )
}
