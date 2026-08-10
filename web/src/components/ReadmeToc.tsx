'use client'

import { Box, Input, InputGroup, InputLeftElement, Icon, Text, VStack } from '@chakra-ui/react'
import { FiSearch } from 'react-icons/fi'
import { useState, useMemo, useEffect, useRef } from 'react'

export interface HeadingItem {
  id: string
  text: string
  level: number
}

interface ReadmeTocProps {
  headings: HeadingItem[]
}

export function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\u4e00-\u9fa5\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .trim()
}

export function extractHeadings(markdown: string): HeadingItem[] {
  const lines = markdown.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n')
  const headings: HeadingItem[] = []
  const seen = new Map<string, number>()
  let inCodeBlock = false

  for (const line of lines) {
    if (/^(`{3,}|~{3,})/.test(line.trim())) {
      inCodeBlock = !inCodeBlock
      continue
    }
    if (inCodeBlock) continue

    const match = line.match(/^(#{1,6})\s+(.+)$/)
    if (match) {
      const level = match[1].length
      const text = match[2].trim()
      let id = slugify(text)
      const count = seen.get(id) || 0
      if (count > 0) {
        id = `${id}-${count}`
      }
      seen.set(slugify(text), count + 1)
      headings.push({ id, text, level })
    }
  }

  return headings
}

export function ReadmeToc({ headings }: ReadmeTocProps) {
  const [filter, setFilter] = useState('')
  const [activeId, setActiveId] = useState('')
  const containerRef = useRef<HTMLDivElement>(null)

  const filteredHeadings = useMemo(() => {
    if (!filter.trim()) return headings
    const lower = filter.toLowerCase()
    return headings.filter(h => h.text.toLowerCase().includes(lower))
  }, [headings, filter])

  useEffect(() => {
    const handleScroll = () => {
      let currentId = ''
      for (const heading of headings) {
        const el = document.getElementById(`readme-heading-${heading.id}`)
        if (el) {
          const rect = el.getBoundingClientRect()
          if (rect.top <= 150) {
            currentId = heading.id
          }
        }
      }
      setActiveId(currentId)
    }

    handleScroll()
    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [headings])

  if (headings.length === 0) return null

  const minLevel = Math.min(...headings.map(h => h.level))

  const handleClick = (id: string) => {
    const el = document.getElementById(`readme-heading-${id}`)
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  }

  return (
    <Box
      ref={containerRef}
      borderWidth="1px"
      borderColor="myGray.200"
      borderRadius="lg"
      overflow="hidden"
      bg="white"
      boxShadow="md"
    >
      <Box p={3} borderBottomWidth="1px" borderColor="myGray.200">
        <InputGroup size="sm">
          <InputLeftElement>
            <Icon as={FiSearch} w={3} h={3} color="myGray.400" />
          </InputLeftElement>
          <Input
            placeholder="Filter headings"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            bg="myGray.50"
            _hover={{ bg: 'myGray.100' }}
            _focus={{ bg: 'white' }}
          />
        </InputGroup>
      </Box>

      <Box maxH="calc(100vh - 200px)" overflowY="auto" py={2}>
        <VStack spacing={0} align="stretch">
          {filteredHeadings.map((heading) => (
            <Box
              key={heading.id}
              px={3}
              py={1.5}
              pl={`${3 + (heading.level - minLevel) * 16}px`}
              cursor="pointer"
              borderLeftWidth={activeId === heading.id ? '3px' : '0px'}
              borderLeftColor="primary.500"
              _hover={{ bg: 'myGray.50' }}
              onClick={() => handleClick(heading.id)}
              bg={activeId === heading.id ? 'myGray.50' : 'transparent'}
              transition="all 0.15s ease"
            >
              <Text
                fontSize="sm"
                color={activeId === heading.id ? 'primary.600' : 'myGray.700'}
                noOfLines={1}
                fontWeight={activeId === heading.id ? 'medium' : 'normal'}
              >
                {heading.text}
              </Text>
            </Box>
          ))}
        </VStack>
      </Box>
    </Box>
  )
}
