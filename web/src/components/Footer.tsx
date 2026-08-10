'use client'

import { Box, Container, Text, HStack, Link } from '@chakra-ui/react'
import { useI18n } from '@/contexts/I18nContext'
import { FiGithub } from 'react-icons/fi'

export function Footer() {
  const { t } = useI18n()
  const version = process.env.NEXT_PUBLIC_APP_VERSION || ''

  return (
    <Box as="footer" bg="myGray.50" borderTop="1px" borderColor="myGray.200" py={6}>
      <Container maxW="container.xl">
        <HStack justify="center" spacing={4} flexWrap="wrap">
          <Text fontSize="sm" color="myGray.600">
            {t('footer.copyrightPrefix', { year: new Date().getFullYear() })}
            <Link
              href="https://iforge-go.com"
              isExternal
              color="myGray.600"
              _hover={{ color: 'primary.600' }}
            >
              iForge
            </Link>
            {version && (
              <Text as="span" fontSize="sm" color="myGray.500" fontFamily="mono" mx={1}>
                v{version}
              </Text>
            )}
            {t('footer.copyrightSuffix')}
          </Text>
          <Text fontSize="sm" color="myGray.400">•</Text>
          <Link
            href="https://github.com/iforge-go/iforge"
            isExternal
            fontSize="sm"
            color="myGray.600"
            _hover={{ color: 'primary.600' }}
          >
            <FiGithub style={{ display: 'inline', verticalAlign: 'middle' }} /> GitHub
          </Link>
        </HStack>
      </Container>
    </Box>
  )
}
