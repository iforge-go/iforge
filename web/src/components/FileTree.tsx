'use client'

import { Box, Text, Icon, Input, InputGroup, InputLeftElement, VStack, HStack } from '@chakra-ui/react'
import { FiFile, FiFolder, FiSearch, FiChevronDown, FiChevronRight } from 'react-icons/fi'
import { useState, useMemo } from 'react'

interface FileTreeNode {
  name: string
  path: string
  type: 'file' | 'dir'
  children?: FileTreeNode[]
}

interface FileTreeProps {
  files: string[]
  selectedFile?: string
  onSelectFile: (path: string) => void
}

function buildTree(files: string[]): FileTreeNode[] {
  const root: FileTreeNode[] = []

  for (const filePath of files) {
    // Normalize path separators (handle both / and \)
    const normalizedPath = filePath.replace(/\\/g, '/')
    const parts = normalizedPath.split('/')
    let current = root

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i]
      const isFile = i === parts.length - 1
      const path = parts.slice(0, i + 1).join('/')

      let existing = current.find((node) => node.name === part)

      if (!existing) {
        existing = {
          name: part,
          path,
          type: isFile ? 'file' : 'dir',
          children: isFile ? undefined : [],
        }
        current.push(existing)
      }

      if (!isFile && existing.children) {
        current = existing.children
      }
    }
  }

  return root
}

function TreeNode({
  node,
  depth,
  selectedFile,
  onSelectFile,
  searchTerm,
}: {
  node: FileTreeNode
  depth: number
  selectedFile?: string
  onSelectFile: (path: string) => void
  searchTerm: string
}) {
  const [expanded, setExpanded] = useState(depth < 2)

  const matchesSearch = searchTerm === '' || node.name.toLowerCase().includes(searchTerm.toLowerCase())
  const hasMatchingChild = node.children?.some(
    (child) => child.name.toLowerCase().includes(searchTerm.toLowerCase())
  )

  if (!matchesSearch && !hasMatchingChild && searchTerm !== '') return null

  if (node.type === 'file') {
    const isSelected = selectedFile === node.path
    return (
      <Box
        pl={`${depth * 16 + 8}px`}
        py={1}
        cursor="pointer"
        bg={isSelected ? 'blue.50' : 'transparent'}
        _hover={{ bg: 'gray.50' }}
        onClick={() => onSelectFile(node.path)}
      >
        <HStack spacing={2} align="center">
          <Box w={3} h={3} flexShrink={0} />
          <Icon as={FiFile} w={3} h={3} color="myGray.400" flexShrink={0} />
          <Text fontSize="sm" color={isSelected ? 'blue.600' : 'myGray.700'} noOfLines={1}>
            {node.name}
          </Text>
        </HStack>
      </Box>
    )
  }

  return (
    <Box>
      <Box
        pl={`${depth * 16 + 8}px`}
        py={1}
        cursor="pointer"
        _hover={{ bg: 'gray.50' }}
        onClick={() => setExpanded(!expanded)}
      >
        <HStack spacing={2} align="center">
          <Icon as={expanded ? FiChevronDown : FiChevronRight} w={3} h={3} color="myGray.500" flexShrink={0} />
          <Icon as={FiFolder} w={3} h={3} color="yellow.600" flexShrink={0} />
          <Text fontSize="sm" fontWeight="medium" color="myGray.700">
            {node.name}
          </Text>
        </HStack>
      </Box>
      {expanded && node.children && (
        <VStack spacing={0} align="stretch">
          {node.children.map((child) => (
            <TreeNode
              key={child.path}
              node={child}
              depth={depth + 1}
              selectedFile={selectedFile}
              onSelectFile={onSelectFile}
              searchTerm={searchTerm}
            />
          ))}
        </VStack>
      )}
    </Box>
  )
}

export function FileTree({ files, selectedFile, onSelectFile }: FileTreeProps) {
  const [searchTerm, setSearchTerm] = useState('')

  const tree = useMemo(() => buildTree(files), [files])

  return (
    <Box borderWidth="1px" borderRadius="md" borderColor="myGray.200" bg="white" overflow="hidden">
      <Box p={3} borderBottom="1px solid" borderColor="myGray.200" bg="myGray.50">
        <InputGroup size="sm">
          <InputLeftElement pointerEvents="none">
            <Icon as={FiSearch} color="myGray.400" />
          </InputLeftElement>
          <Input
            placeholder="Filter files..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            bg="white"
          />
        </InputGroup>
      </Box>
      <Box maxH="600px" overflowY="auto" py={2}>
        {tree.map((node) => (
          <TreeNode
            key={node.path}
            node={node}
            depth={0}
            selectedFile={selectedFile}
            onSelectFile={onSelectFile}
            searchTerm={searchTerm}
          />
        ))}
      </Box>
    </Box>
  )
}
