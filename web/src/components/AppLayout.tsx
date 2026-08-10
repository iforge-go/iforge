'use client'

import { Box } from '@chakra-ui/react'
import { Footer } from './Footer'

export function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <Box display="flex" flexDirection="column" minH="100vh">
      <Box flex="1">
        {children}
      </Box>
      <Footer />
    </Box>
  )
}
