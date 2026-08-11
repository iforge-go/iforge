'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  VStack,
  SimpleGrid,
  Icon,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useEffect } from 'react'
import { FiBook, FiRefreshCw, FiCheckSquare, FiSliders, FiPlay, FiCpu } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'

export default function HomePage() {
  const router = useRouter()
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()

  useEffect(() => {
    if (user) {
      router.replace('/pms')
    }
  }, [user, router])

  if (authLoading) return null

  if (user) {
    return null
  }

  return (
    <Box bg="myGray.50">
      <Container maxW="container.lg" py={20}>
        <VStack spacing={8} textAlign="center">
          <VStack spacing={4}>
            <Heading size="2xl" color="myGray.900">
              {t('home.welcome')}
            </Heading>
            <Text fontSize="lg" color="myGray.600" maxW="650px">
              {t('home.welcomeDesc')}
            </Text>
          </VStack>

          {/* Row 1: 核心差异化特性 */}
          <SimpleGrid columns={{ base: 1, md: 3 }} spacing={6} mt={12} w="full">
            <Link href="/pms">
              <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} cursor="pointer" _hover={{ borderColor: 'primary.400', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }} transition="all 0.2s">
                <VStack spacing={3}>
                  <Box p={3} bg="primary.50" borderRadius="lg">
                    <Icon as={FiRefreshCw} w={6} h={6} color="primary.600" />
                  </Box>
                  <Heading size="sm">{t('home.devLoopFeature')}</Heading>
                  <Text fontSize="sm" color="myGray.600" textAlign="center">
                    {t('home.devLoopFeatureDesc')}
                  </Text>
                </VStack>
              </Box>
            </Link>
            <Link href="/pms">
              <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} cursor="pointer" _hover={{ borderColor: 'primary.400', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }} transition="all 0.2s">
                <VStack spacing={3}>
                  <Box p={3} bg="blue.50" borderRadius="lg">
                    <Icon as={FiCheckSquare} w={6} h={6} color="blue.600" />
                  </Box>
                  <Heading size="sm">{t('home.scrumFeature')}</Heading>
                  <Text fontSize="sm" color="myGray.600" textAlign="center">
                    {t('home.scrumFeatureDesc')}
                  </Text>
                </VStack>
              </Box>
            </Link>
            <Link href="/pms">
              <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} cursor="pointer" _hover={{ borderColor: 'primary.400', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }} transition="all 0.2s">
                <VStack spacing={3}>
                  <Box p={3} bg="purple.50" borderRadius="lg">
                    <Icon as={FiSliders} w={6} h={6} color="purple.600" />
                  </Box>
                  <Heading size="sm">{t('home.workflowFeature')}</Heading>
                  <Text fontSize="sm" color="myGray.600" textAlign="center">
                    {t('home.workflowFeatureDesc')}
                  </Text>
                </VStack>
              </Box>
            </Link>
          </SimpleGrid>

          {/* Row 2: 平台能力 */}
          <SimpleGrid columns={{ base: 1, md: 3 }} spacing={6} w="full">
            <Link href="/vcs">
              <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} cursor="pointer" _hover={{ borderColor: 'primary.400', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }} transition="all 0.2s">
                <VStack spacing={3}>
                  <Box p={3} bg="green.50" borderRadius="lg">
                    <Icon as={FiBook} w={6} h={6} color="green.600" />
                  </Box>
                  <Heading size="sm">{t('home.vcsFeature')}</Heading>
                  <Text fontSize="sm" color="myGray.600" textAlign="center">
                    {t('home.vcsFeatureDesc')}
                  </Text>
                </VStack>
              </Box>
            </Link>
            <Link href="/vcs">
              <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} cursor="pointer" _hover={{ borderColor: 'primary.400', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }} transition="all 0.2s">
                <VStack spacing={3}>
                  <Box p={3} bg="yellow.50" borderRadius="lg">
                    <Icon as={FiPlay} w={6} h={6} color="yellow.600" />
                  </Box>
                  <Heading size="sm">{t('home.cicdFeature')}</Heading>
                  <Text fontSize="sm" color="myGray.600" textAlign="center">
                    {t('home.cicdFeatureDesc')}
                  </Text>
                </VStack>
              </Box>
            </Link>
            <Link href="/pms">
              <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} cursor="pointer" _hover={{ borderColor: 'primary.400', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }} transition="all 0.2s">
                <VStack spacing={3}>
                  <Box p={3} bg="red.50" borderRadius="lg">
                    <Icon as={FiCpu} w={6} h={6} color="red.600" />
                  </Box>
                  <Heading size="sm">{t('home.aiFeature')}</Heading>
                  <Text fontSize="sm" color="myGray.600" textAlign="center">
                    {t('home.aiFeatureDesc')}
                  </Text>
                </VStack>
              </Box>
            </Link>
          </SimpleGrid>
        </VStack>
      </Container>
    </Box>
  )
}
