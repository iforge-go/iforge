'use client'

import {
  Box,
  Container,
  Text,
  VStack,
  HStack,
  Icon,
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Spinner,
  Button,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverBody,
  PopoverArrow,
  InputGroup,
  InputLeftElement,
  Input,
  IconButton,
  Code,
  Heading,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { api, API_BASE, FileEntry, CommitInfo } from '@/lib/api'
import { parseBranchAndPath, shouldWaitForBranches } from '@/lib/branchPath'
import { FiFolder, FiFile, FiChevronRight, FiGitBranch, FiSearch, FiPlus, FiUpload, FiList } from 'react-icons/fi'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeRaw from 'rehype-raw'
import { useRepo } from '@/app/[owner]/[repo]/RepoContext'
import { useGithubToast } from '@/app/providers'
import { FileListTable } from '@/components/FileListTable'
import { CommitInfoBar } from '@/components/CommitInfoBar'
import { extractHeadings, slugify } from '@/components/ReadmeToc'
import { ReadmeTocOverlay } from '@/components/ReadmeTocOverlay'
import { FiEdit } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

function extractText(children: any): string {
  if (typeof children === 'string') return children
  if (Array.isArray(children)) return children.map(extractText).join('')
  if (children?.props?.children) return extractText(children.props.children)
  return ''
}

export default function TreePage() {
  const params = useParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const pathSegments = params.path as string[]
  const { repo, branches, refreshData } = useRepo()
  const toast = useGithubToast()
  const { t } = useI18n()
  
  const defaultBranch = branches.find(b => b.isDefault)?.name || branches[0]?.name || 'main'
  // 分支名可能含 "/"（如 feature/task-15），catch-all 路由会拆成多段，
  // 用已知分支列表反向匹配前缀来正确分离分支名和文件路径。
  const { ref, subPath: dirPath } = parseBranchAndPath(pathSegments, branches, defaultBranch)

  const [files, setFiles] = useState<FileEntry[]>([])
  const [commits, setCommits] = useState<CommitInfo[]>([])
  const [readmeContent, setReadmeContent] = useState<string>('')
  const [readmeRaw, setReadmeRaw] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [branchSearch, setBranchSearch] = useState('')
  const [creatingBranch, setCreatingBranch] = useState(false)
  const [showToc, setShowToc] = useState(false)

  // branches 是否已加载（多段路径时需等待，以便正确反向匹配分支名）
  const branchesLoaded = branches.length > 0

  useEffect(() => {
    const loadData = async () => {
      // 多段路径可能含 "/" 分支名，需等 branches 加载后才能正确解析，避免空目录闪烁
      if (shouldWaitForBranches(pathSegments, branches)) return
      try {
        setLoading(true)
        const [fileList, commitsData] = await Promise.all([
          api.listFiles(owner, repoName, ref, dirPath).catch(() => []),
          api.listCommits(owner, repoName, ref, 1, 10).catch(() => []),
        ])
        setFiles(fileList)
        setCommits(commitsData || [])
        setError(null)

        const readmeFiles = (fileList || []).filter(f =>
          f.type === 'file' && /^readme(\.(md|markdown|txt))?$/i.test(f.name)
        )
        if (readmeFiles.length > 0) {
          try {
            const readmeFile = readmeFiles[0]
            const fileContent = await api.getFileContent(owner, repoName, readmeFile.path, ref)
            setReadmeContent(fileContent.content)
            setReadmeRaw(fileContent.content)
          } catch (error) {
            console.error('Failed to load README:', error)
          }
        } else {
          setReadmeContent('')
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load directory')
      } finally {
        setLoading(false)
      }
    }

    loadData()
  }, [owner, repoName, dirPath, ref, branchesLoaded, pathSegments.length])

  const handleBranchChange = (branchName: string) => {
    const newPath = dirPath 
      ? `/${owner}/${repoName}/tree/${branchName}/${dirPath}`
      : `/${owner}/${repoName}/tree/${branchName}`
    window.location.href = newPath
  }

  const handleCreateBranch = async (branchName: string) => {
    try {
      setCreatingBranch(true)
      await api.createBranch(owner, repoName, branchName, ref)
      toast({
        title: t('repo.branchCreated'),
        status: 'success',
        duration: 2000,
      })
      setBranchSearch('')
      await refreshData()
      handleBranchChange(branchName)
    } catch (error: any) {
      toast({
        title: t('repo.createBranchFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setCreatingBranch(false)
    }
  }

  const filteredBranches = branches.filter(b => 
    b.name.toLowerCase().includes(branchSearch.toLowerCase())
  )

  if (loading) {
    return (
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={8}>
          <VStack spacing={4}>
            <Spinner size="xl" />
            <Text color="myGray.600">Loading directory...</Text>
          </VStack>
        </Container>
      </Box>
    )
  }

  if (error) {
    return (
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={8}>
          <VStack spacing={4}>
            <Text color="red.500">{error}</Text>
            <Link href={`/${owner}/${repoName}`}>
              <Button variant="whiteBase">Back to repository</Button>
            </Link>
          </VStack>
        </Container>
      </Box>
    )
  }

  if (branches.length === 0) {
    return (
      <VStack spacing={6} align="stretch">
        <HStack justify="space-between" flexWrap="wrap" gap={3}>
          <HStack spacing={3}>
            <Button
              leftIcon={<Icon as={FiGitBranch} />}
              rightIcon={<Icon as={FiChevronRight} transform="rotate(90deg)" />}
              variant="outline"
              size="sm"
              bg="white"
              borderWidth="1px"
              borderColor="myGray.200"
              _hover={{ bg: 'myGray.50', borderColor: 'myGray.300' }}
              disabled
            >
              No branches
            </Button>
          </HStack>
          <HStack>
            <Text fontSize="sm" color="myGray.500">
              0 branches
            </Text>
          </HStack>
        </HStack>

        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
          <Table size="sm" borderColor="myGray.200">
            <Thead>
              <Tr bg="myGray.100" borderBottomColor="myGray.200">
                <Th color="myGray.600" w="40%">Name</Th>
                <Th color="myGray.600" w="40%">Last commit message</Th>
                <Th color="myGray.600" isNumeric>Last commit date</Th>
              </Tr>
            </Thead>
            <Tbody>
              <Tr>
                <Td colSpan={3}>
                  <VStack spacing={4} py={8}>
                    <Icon as={FiFolder} w={10} h={10} color="myGray.300" />
                    <Text color="myGray.500" fontSize="sm">
                      This repository has no branches yet.
                    </Text>
                    <Text color="myGray.400" fontSize="xs">
                      Create a branch to start adding files.
                    </Text>
                  </VStack>
                </Td>
              </Tr>
            </Tbody>
          </Table>
        </Box>
      </VStack>
    )
  }

  const pathParts = dirPath ? dirPath.split('/').filter(Boolean) : []
  const breadcrumbs = [
    { label: repoName, href: `/${owner}/${repoName}` },
    ...pathParts.map((part, index) => ({
      label: part,
      href: `/${owner}/${repoName}/tree/${ref}/${pathParts.slice(0, index + 1).join('/')}`,
    })),
  ]

  const sortedFiles = [...files].sort((a, b) => {
    if (a.type === b.type) return a.name.localeCompare(b.name)
    return a.type === 'dir' ? -1 : 1
  })

  return (
    <VStack spacing={4} align="stretch">
      <HStack justify="space-between" flexWrap="wrap" gap={3}>
        <HStack spacing={3}>
          <Popover placement="bottom-start">
            <PopoverTrigger>
              <Button
                leftIcon={<Icon as={FiGitBranch} />}
                rightIcon={<Icon as={FiChevronRight} transform="rotate(90deg)" />}
                variant="outline"
                size="sm"
                bg="white"
                borderWidth="1px"
                borderColor="myGray.200"
              _hover={{ bg: 'myGray.50', borderColor: 'myGray.300' }}
              >
                {ref}
              </Button>
            </PopoverTrigger>
            <PopoverContent w="400px" maxH="600px" overflow="hidden" bg="white" position="relative">
              <PopoverArrow />
              <IconButton
                position="absolute"
                right={3}
                top={3}
                size="sm"
                onClick={() => document.body.click()}
                aria-label="Close"
                icon={<Icon as={FiChevronRight} transform="rotate(45deg)" w={4} h={4} />}
                bg="transparent"
                _hover={{ bg: 'myGray.100' }}
                zIndex={10}
              />
              <PopoverBody p={0}>
                <Box display="flex" flexDirection="column">
                  <Box p={3} borderBottom="1px" borderColor="myGray.200">
                    <InputGroup size="sm">
                      <InputLeftElement>
                        <Icon as={FiSearch} color="myGray.400" w={3} h={3} />
                      </InputLeftElement>
                      <Input
                        placeholder="Search branches..."
                        value={branchSearch}
                        onChange={(e) => setBranchSearch(e.target.value)}
                      />
                    </InputGroup>
                  </Box>

                  <Box display="flex" flexDirection="column" gap={0}>
                    {branchSearch && !branches.some(b => b.name === branchSearch) && (
                      <Box px={3} py={1} w="100%">
                        <Box
                          key="create-new"
                          display="flex"
                          alignItems="center"
                          gap={2}
                          px={3}
                          py={1}
                          borderRadius="md"
                          cursor={creatingBranch ? 'not-allowed' : 'pointer'}
                          _hover={creatingBranch ? {} : { bg: 'myGray.100' }}
                          onClick={() => !creatingBranch && handleCreateBranch(branchSearch)}
                          opacity={creatingBranch ? 0.6 : 1}
                        >
                          {creatingBranch ? (
                            <Spinner size="xs" />
                          ) : (
                            <Icon as={FiGitBranch} w={4} h={4} />
                          )}
                          <Text fontSize="sm">Create branch</Text>
                          <Code fontSize="xs" colorScheme="gray" px={1} ml={1}>
                            {branchSearch}
                          </Code>
                          <Text fontSize="sm" color="myGray.500" ml={1}>from</Text>
                          <Code fontSize="xs" colorScheme="gray" px={1} ml={1}>
                            {ref}
                          </Code>
                        </Box>
                      </Box>
                    )}

                    {filteredBranches.length === 0 && !branchSearch ? (
                      <Box p={4} textAlign="center">
                        <Text fontSize="sm" color="myGray.500">No branches found</Text>
                      </Box>
                    ) : filteredBranches.length > 0 ? (
                      filteredBranches.map((branch) => (
                        <Box key={branch.name} px={3} py={1} w="100%">
                          <Box
                            display="flex"
                            alignItems="center"
                            gap={2}
                            px={3}
                            py={1}
                            borderRadius="md"
                            cursor="pointer"
                            _hover={{ bg: 'myGray.100' }}
                            onClick={() => handleBranchChange(branch.name)}
                            bg={branch.name === ref ? 'myGray.200' : 'transparent'}
                          >
                            <Icon as={FiGitBranch} w={4} h={4} />
                            <Text fontSize="sm" fontWeight={branch.name === ref ? 'medium' : 'normal'}>
                              {branch.name}
                            </Text>
                            {branch.isDefault && (
                              <Text fontSize="xs" color="myGray.500" ml="auto">default</Text>
                            )}
                          </Box>
                        </Box>
                      ))
                    ) : null}
                  </Box>

                  <Box p={3} borderTop="1px" borderColor="myGray.200">
                    <Link href={`/${owner}/${repoName}/branches`}>
                      <Button variant="ghost" size="sm" color="primary.600" w="100%">
                        View all {branches.length} branches
                      </Button>
                    </Link>
                  </Box>
                </Box>
              </PopoverBody>
            </PopoverContent>
          </Popover>
          <Breadcrumb separator={<Icon as={FiChevronRight} color="myGray.400" />} fontSize="sm">
            {breadcrumbs.map((crumb, index) => (
              <BreadcrumbItem key={index} isCurrentPage={index === breadcrumbs.length - 1}>
                {index === breadcrumbs.length - 1 ? (
                  <BreadcrumbLink color="myGray.900" fontWeight="medium">
                    {crumb.label}
                  </BreadcrumbLink>
                ) : (
                  <BreadcrumbLink as={Link} href={crumb.href} color="myGray.600">
                    {crumb.label}
                  </BreadcrumbLink>
                )}
              </BreadcrumbItem>
            ))}
          </Breadcrumb>
        </HStack>
        <Menu>
          <MenuButton
            as={Button}
            rightIcon={<Icon as={FiChevronRight} transform="rotate(90deg)" />}
            variant="whiteBase"
            size="sm"
            borderWidth="1px"
            borderColor="myGray.200"
          >
            <HStack spacing={1}>
              <Icon as={FiPlus} />
              <Text>Add file</Text>
            </HStack>
          </MenuButton>
          <MenuList>
            <MenuItem as={Link} href={`/${owner}/${repoName}/new/${ref}${dirPath ? '/' + dirPath : ''}`} icon={<Icon as={FiFile} />}>
              New file
            </MenuItem>
            <MenuItem icon={<Icon as={FiUpload} />} isDisabled>
              Upload file
            </MenuItem>
          </MenuList>
        </Menu>
      </HStack>

      <Box borderWidth="1px" borderRadius="lg" overflow="hidden">
        <CommitInfoBar
          owner={owner}
          repo={repoName}
          ref={ref}
          dirPath={dirPath}
          commits={commits}
        />
        <FileListTable
          files={files}
          owner={owner}
          repo={repoName}
          ref={ref}
          labels={{
            name: t('repo.fileName'),
            lastCommitMessage: t('repo.lastCommitMessage'),
            lastCommitDate: t('repo.lastCommitDate'),
            empty: t('repo.emptyDir'),
          }}
        />
      </Box>

      {readmeContent && (
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
                  <IconButton
                    as={Link}
                    aria-label={t('repo.edit')}
                    icon={<Icon as={FiEdit} />}
                    href={`/${owner}/${repoName}/edit/${ref}/README.md`}
                    size="sm"
                    variant="ghost"
                    color="myGray.500"
                    _hover={{ bg: 'myGray.100' }}
                  />
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
                  '& th': { bg: 'gray.50', fontWeight: 'bold' }
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
                        return <img src={src} alt={alt || ''} style={{ maxWidth: '100%', height: 'auto' }} />
                      }
                      const srcStr = typeof src === 'string' ? src : ''
                      const fullPath = dirPath ? `${dirPath}/${srcStr}` : srcStr
                      const rawUrl = `/${owner}/${repoName}/raw/${ref}/${fullPath}`
                      return <img src={rawUrl} alt={alt || ''} style={{ maxWidth: '100%', height: 'auto' }} />
                    }
                  }}
                >
                  {readmeContent}
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
      )}
    </VStack>
  )
}
