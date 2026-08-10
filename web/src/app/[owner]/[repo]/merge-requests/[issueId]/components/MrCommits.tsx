'use client'

import {
  Box,
  Text,
  HStack,
} from '@chakra-ui/react'
import { useI18n } from '@/contexts/I18nContext'

interface MrCommitsProps {
  commits: Array<{ id: string; message: string; author: string; timestamp: string }>
}

export function MrCommits({ commits }: MrCommitsProps) {
  const { t } = useI18n()

  return (
    <>
      {commits.length > 0 ? (
        <Box>
          {commits.map((commit) => (
            <Box key={commit.id} px={4} py={3} borderBottom="1px" borderColor="myGray.100">
              <HStack spacing={3}>
                <Text fontSize="xs" fontFamily="mono" color="primary.600" fontWeight="medium">
                  {commit.id.substring(0, 7)}
                </Text>
                <Text fontSize="sm" color="myGray.900" flex={1}>
                  {commit.message.split('\n')[0]}
                </Text>
                <Text fontSize="xs" color="myGray.500">
                  {commit.author}
                </Text>
              </HStack>
            </Box>
          ))}
        </Box>
      ) : (
        <Box px={4} py={8} textAlign="center">
          <Text fontSize="sm" color="myGray.500">{t('repo.noCommits')}</Text>
        </Box>
      )}
    </>
  )
}
