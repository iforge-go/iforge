'use client'

import { Box } from '@chakra-ui/react'
import { Footer } from './Footer'

export function FooterWrapper() {
  return <Footer />
}

export function MainContent({ children }: { children: React.ReactNode }) {
  return <Box flex="1">{children}</Box>
}

export function LayoutContainer({ children }: { children: React.ReactNode }) {
  return (
    <Box display="flex" flexDirection="column" minH="100vh">
      {children}
    </Box>
  )
}
