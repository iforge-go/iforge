'use client'

import { Box, Text } from '@chakra-ui/react'

export function DemoBanner() {
  if (process.env.NEXT_PUBLIC_DEMO_BANNER !== 'true') return null

  return (
    <Box
      position="fixed"
      bottom={0}
      left={0}
      right={0}
      bg="#1a365d"
      color="white"
      py={2}
      textAlign="center"
      zIndex={9999}
      fontSize="13px"
    >
      <Text>
        演示环境 | 账户: demo / demo12345 (管理员) · alice / bob / charlie / demo12345 | 每日凌晨 3:00 重置数据
      </Text>
    </Box>
  )
}
