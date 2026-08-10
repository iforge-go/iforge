'use client'

import { Box } from '@chakra-ui/react'
import { useEffect, useRef } from 'react'
import { ReadmeToc, HeadingItem } from './ReadmeToc'

interface ReadmeTocOverlayProps {
  headings: HeadingItem[]
  onClose: () => void
}

export function ReadmeTocOverlay({ headings, onClose }: ReadmeTocOverlayProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) {
        onClose()
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [onClose])

  return (
    <Box
      ref={ref}
      position="absolute"
      top="60px"
      bottom="0"
      right="16px"
      w="280px"
      zIndex={20}
      display={{ base: 'none', lg: 'block' }}
    >
      <Box
        position="sticky"
        top="80px"
        maxH="calc(100vh - 100px)"
        overflowY="auto"
      >
        <ReadmeToc headings={headings} />
      </Box>
    </Box>
  )
}
