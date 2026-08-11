'use client'

import {
  Box,
  Container,
  Heading,
  Text,
  Button,
  VStack,
  HStack,
  Icon,
  SimpleGrid,
} from '@chakra-ui/react'
import Link from 'next/link'
import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { FiPlus, FiSettings, FiShield } from 'react-icons/fi'
import { AiOutlineBank } from 'react-icons/ai'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'

export default function DashboardPage() {
  const router = useRouter()
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()

  useEffect(() => {
    const token = localStorage.getItem('token')
    if (!token) {
      router.push('/login')
    }
  }, [router])

  if (authLoading) return null

  if (!user) {
    return (
      <Box bg="myGray.50">
        <Container maxW="container.xl" py={8}>
          <Text>{t('common.pleaseLogin')}</Text>
        </Container>
      </Box>
    )
  }

  return (
    <Box bg="myGray.50">
      <Container maxW="container.xl" py={8}>
        <VStack spacing={6} align="stretch">
          <Heading size="lg" color="myGray.900">
            {t('dashboard.title')}
          </Heading>

          <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} spacing={6}>
            <Box bg="white" borderWidth="1px" borderColor="myGray.200" borderRadius="lg" p={6}>
              <VStack spacing={4} align="start">
                <HStack>
                  <Box p={3} bg="primary.50" borderRadius="lg">
                    <Icon as={AiOutlineBank} w={6} h={6} color="primary.600" />
                  </Box>
                  <VStack align="start" spacing={0}>
                    <Heading size="sm">{t('dashboard.myOrganizations')}</Heading>
                    <Text fontSize="xs" color="myGray.500">{t('dashboard.myOrganizationsDesc')}</Text>
                  </VStack>
                </HStack>
                <Link href="/organizations">
                  <Button leftIcon={<Icon as={AiOutlineBank} />} variant="whiteBase" size="sm">
                    {t('dashboard.viewOrganizations')}
                  </Button>
                </Link>
              </VStack>
            </Box>

            <Box bg="white" borderWidth="1px" borderColor="myGray.200" borderRadius="lg" p={6}>
              <VStack spacing={4} align="start">
                <HStack>
                  <Box p={3} bg="green.50" borderRadius="lg">
                    <Icon as={FiPlus} w={6} h={6} color="green.600" />
                  </Box>
                  <VStack align="start" spacing={0}>
                    <Heading size="sm">{t('dashboard.createOrganization')}</Heading>
                    <Text fontSize="xs" color="myGray.500">{t('dashboard.createOrganizationDesc')}</Text>
                  </VStack>
                </HStack>
                <Link href="/organizations/new">
                  <Button leftIcon={<Icon as={FiPlus} />} variant="primary" size="sm">
                    {t('dashboard.createOrganization')}
                  </Button>
                </Link>
              </VStack>
            </Box>

            {user.isAdmin && (
              <Box bg="white" borderWidth="1px" borderColor="myGray.200" borderRadius="lg" p={6}>
                <VStack spacing={4} align="start">
                  <HStack>
                    <Box p={3} bg="yellow.50" borderRadius="lg">
                      <Icon as={FiShield} w={6} h={6} color="yellow.600" />
                    </Box>
                    <VStack align="start" spacing={0}>
                      <Heading size="sm">{t('dashboard.systemAdmin')}</Heading>
                      <Text fontSize="xs" color="myGray.500">{t('dashboard.systemAdminDesc')}</Text>
                    </VStack>
                  </HStack>
                  <Link href="/admin">
                    <Button leftIcon={<Icon as={FiSettings} />} variant="whiteBase" size="sm">
                      {t('dashboard.enterAdminPanel')}
                    </Button>
                  </Link>
                </VStack>
              </Box>
            )}
          </SimpleGrid>
        </VStack>
      </Container>
    </Box>
  )
}
