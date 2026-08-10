'use client'

import { Box, VStack, HStack, Text, Icon, SimpleGrid, Table, Thead, Tbody, Tr, Th, Td, Badge, Stat, StatLabel, StatNumber, StatHelpText, Tooltip } from '@chakra-ui/react'
import { SystemInfo } from '@/lib/api'
import { FiPackage, FiCalendar, FiClock, FiUsers, FiDatabase, FiCpu, FiMonitor, FiHardDrive, FiGlobe, FiTerminal, FiFolder, FiHome } from 'react-icons/fi'
import { AiOutlineBank } from 'react-icons/ai'
import { useI18n } from '@/contexts/I18nContext'

interface AdminOverviewProps {
  systemInfo: SystemInfo | null
  uptimeSeconds: number
}

export default function AdminOverview({ systemInfo, uptimeSeconds }: AdminOverviewProps) {
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = Math.floor(seconds % 60)
    if (days > 0) return t('admin.uptimeDays', { days, hours, minutes })
    if (hours > 0) return t('admin.uptimeHours', { hours, minutes, seconds: secs })
    if (minutes > 0) return t('admin.uptimeMinutes', { minutes, seconds: secs })
    return t('admin.uptimeSeconds', { seconds: secs })
  }

  return (
    <VStack spacing={6} align="stretch">
      {/* 当前时间与运行时间 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200" px={6} py={3}>
        <HStack spacing={4} align="center" wrap="wrap">
          <Icon as={FiCalendar} w={4} h={4} color="myGray.400" />
          <Text fontSize="sm" color="myGray.600" fontWeight="medium">{t('admin.currentTime')}</Text>
          <Text fontSize="sm" color="myGray.700" fontFamily="mono">
            {systemInfo?.serverTime ?? '-'}
          </Text>
          <Text fontSize="xs" color="myGray.400">
            ({systemInfo?.timezone ?? '-'})
          </Text>

          <Box w="1px" h="14px" bg="myGray.200" />

          <Icon as={FiClock} w={4} h={4} color="green.500" />
          <Text fontSize="sm" color="myGray.600" fontWeight="medium">{t('admin.uptime')}</Text>
          <Text fontSize="sm" fontWeight="bold" color="green.600" fontFamily="mono">
            {uptimeSeconds > 0 ? formatUptime(uptimeSeconds) : '-'}
          </Text>
        </HStack>
      </Box>

      {/* 统计卡片 */}
      <SimpleGrid columns={{ base: 1, sm: 2, md: 3, xl: 4 }} spacing={6}>
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} borderColor="myGray.200">
          <HStack spacing={4} align="flex-start">
            <Box bg="primary.50" borderRadius="lg" p={3}>
              <Icon as={FiUsers} w={6} h={6} color="primary.500" />
            </Box>
            <Stat>
              <StatLabel color="myGray.600">{t('admin.totalUsers')}</StatLabel>
              <StatNumber color="primary.600">{systemInfo?.userCount ?? 0}</StatNumber>
              <StatHelpText>{t('admin.activeUsers')}</StatHelpText>
            </Stat>
          </HStack>
        </Box>

        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} borderColor="myGray.200">
          <HStack spacing={4} align="flex-start">
            <Box bg="purple.50" borderRadius="lg" p={3}>
              <Icon as={AiOutlineBank} w={6} h={6} color="purple.500" />
            </Box>
            <Stat>
              <StatLabel color="myGray.600">{t('admin.totalOrganizations')}</StatLabel>
              <StatNumber color="purple.600">{systemInfo?.orgCount ?? 0}</StatNumber>
              <StatHelpText>{t('admin.orgs')}</StatHelpText>
            </Stat>
          </HStack>
        </Box>

        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} borderColor="myGray.200">
          <HStack spacing={4} align="flex-start">
            <Box bg="green.50" borderRadius="lg" p={3}>
              <Icon as={FiDatabase} w={6} h={6} color="green.500" />
            </Box>
            <Stat>
              <StatLabel color="myGray.600">{t('admin.totalRepos')}</StatLabel>
              <StatNumber color="green.600">{systemInfo?.repoCount ?? 0}</StatNumber>
              <StatHelpText>{t('admin.gitRepos')}</StatHelpText>
            </Stat>
          </HStack>
        </Box>

        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} borderColor="myGray.200">
          <HStack spacing={4} align="flex-start">
            <Box bg="cyan.50" borderRadius="lg" p={3}>
              <Icon as={FiFolder} w={6} h={6} color="cyan.500" />
            </Box>
            <Stat>
              <StatLabel color="myGray.600">{t('admin.totalProjects')}</StatLabel>
              <StatNumber color="cyan.600">{systemInfo?.projectCount ?? 0}</StatNumber>
              <StatHelpText>{t('admin.projects')}</StatHelpText>
            </Stat>
          </HStack>
        </Box>

        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} borderColor="myGray.200">
          <HStack spacing={4} align="flex-start">
            <Box bg="red.50" borderRadius="lg" p={3}>
              <Icon as={FiCpu} w={6} h={6} color="red.500" />
            </Box>
            <Stat>
              <StatLabel color="myGray.600">{t('admin.cpuCores')}</StatLabel>
              <StatNumber color="red.600">{systemInfo?.cpu?.numCPU ?? '-'}</StatNumber>
              <StatHelpText>{t('admin.goroutine')}: {systemInfo?.cpu?.numGoroutine ?? '-'}</StatHelpText>
            </Stat>
          </HStack>
        </Box>

        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} borderColor="myGray.200">
          <HStack spacing={4} align="flex-start">
            <Box bg="teal.50" borderRadius="lg" p={3}>
              <Icon as={FiMonitor} w={6} h={6} color="teal.500" />
            </Box>
            <Stat>
              <StatLabel color="myGray.600">{t('admin.os')}</StatLabel>
              <StatNumber color="teal.600" fontSize="2xl">{systemInfo?.os?.nameHuman ?? '-'}</StatNumber>
              <StatHelpText>{systemInfo?.os?.name ?? '-'} · {systemInfo?.os?.arch ?? '-'}</StatHelpText>
            </Stat>
          </HStack>
        </Box>

        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} borderColor="myGray.200">
          <HStack spacing={4} align="flex-start">
            <Box bg="orange.50" borderRadius="lg" p={3}>
              <Icon as={FiHardDrive} w={6} h={6} color="orange.500" />
            </Box>
            <Stat>
              <StatLabel color="myGray.600">{t('admin.memory')}</StatLabel>
              <StatNumber color="orange.600" fontSize="2xl">
                {systemInfo?.memory?.allocHuman ?? '-'}
              </StatNumber>
              <StatHelpText>{t('admin.systemAlloc')} {systemInfo?.memory?.sysHuman}</StatHelpText>
            </Stat>
          </HStack>
        </Box>

        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6} borderColor="myGray.200">
          <HStack spacing={4} align="flex-start">
            <Box bg="primary.50" borderRadius="lg" p={3}>
              <Icon as={FiPackage} w={6} h={6} color="primary.500" />
            </Box>
            <VStack align="flex-start" spacing={0} flex={1} minW={0}>
              <Text fontSize="sm" color="myGray.600">{t('admin.iforgeVersion')}</Text>
              <Text fontSize="2xl" fontWeight="bold" color="primary.600">v{systemInfo?.version ?? '-'}</Text>
              {systemInfo?.gitCommit && (
                <Tooltip label={systemInfo.buildTime ? new Date(systemInfo.buildTime).toLocaleString(dateLocale) : undefined}>
                  <Badge colorScheme="cyan" variant="subtle" fontFamily="mono" fontSize="xs" mt={0.5} cursor="pointer" textTransform="none">
                    {systemInfo.gitCommit.toLowerCase()}{systemInfo.dirty ? '*' : ''}
                  </Badge>
                </Tooltip>
              )}
            </VStack>
          </HStack>
        </Box>
      </SimpleGrid>

      {/* 服务信息 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200" mt={6}>
        <Box bg="myGray.100" px={6} py={3} borderBottomWidth="1px" borderColor="myGray.200">
          <Text fontWeight="bold" fontSize="sm" color="myGray.700">{t('admin.serviceInfo')}</Text>
        </Box>
        <Table variant="simple" borderColor="myGray.200">
          <Thead>
            <Tr bg="myGray.50">
              <Th width="160px">{t('admin.service')}</Th>
              <Th>{t('admin.address')}</Th>
              <Th width="100px">{t('admin.port')}</Th>
              <Th width="120px">{t('admin.status')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            <Tr>
              <Td>
                <HStack spacing={2}>
                  <Icon as={FiGlobe} color="primary.500" />
                  <Text fontWeight="medium">HTTP API</Text>
                </HStack>
              </Td>
              <Td>
                <Text fontSize="sm" color="myGray.600" fontFamily="mono">
                  {systemInfo?.services?.http?.address ?? '-'}
                </Text>
              </Td>
              <Td>
                <Badge colorScheme="green">{systemInfo?.services?.http?.port ?? '-'}</Badge>
              </Td>
              <Td>
                <Badge colorScheme="green">{t('admin.running')}</Badge>
              </Td>
            </Tr>
            <Tr>
              <Td>
                <HStack spacing={2}>
                  <Icon as={FiTerminal} color="purple.500" />
                  <Text fontWeight="medium">SSH</Text>
                </HStack>
              </Td>
              <Td>
                <Text fontSize="sm" color="myGray.600" fontFamily="mono">
                  {systemInfo?.services?.ssh?.address ?? '-'}
                </Text>
              </Td>
              <Td>
                <Badge colorScheme={systemInfo?.services?.ssh?.enabled ? 'green' : 'red'}>
                  {systemInfo?.services?.ssh?.port ?? '-'}
                </Badge>
              </Td>
              <Td>
                <Badge colorScheme={systemInfo?.services?.ssh?.enabled ? 'green' : 'red'}>
                  {systemInfo?.services?.ssh?.enabled ? t('admin.running') : t('admin.disabled')}
                </Badge>
              </Td>
            </Tr>
          </Tbody>
        </Table>
      </Box>

      {/* 存储信息 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200" mt={6}>
        <Box bg="myGray.100" px={6} py={3} borderBottomWidth="1px" borderColor="myGray.200">
          <Text fontWeight="bold" fontSize="sm" color="myGray.700">{t('admin.storageInfo')}</Text>
        </Box>
        <Table variant="simple" borderColor="myGray.200">
          <Thead>
            <Tr bg="myGray.50">
              <Th width="160px">{t('admin.name')}</Th>
              <Th>{t('admin.path')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            <Tr>
              <Td>
                <HStack spacing={2}>
                  <Icon as={FiDatabase} color="orange.500" />
                  <Text fontWeight="medium">{t('admin.database')}</Text>
                </HStack>
              </Td>
              <Td>
                <HStack spacing={2}>
                  <Badge colorScheme="orange">{systemInfo?.services?.database?.type ?? '-'}</Badge>
                  <Text fontSize="sm" color="myGray.600" fontFamily="mono">
                    {systemInfo?.services?.database?.path ?? '-'}
                  </Text>
                </HStack>
              </Td>
            </Tr>
            <Tr>
              <Td>
                <HStack spacing={2}>
                  <Icon as={FiFolder} color="blue.500" />
                  <Text fontWeight="medium">{t('admin.reposDir')}</Text>
                </HStack>
              </Td>
              <Td>
                <Text fontSize="sm" color="myGray.600" fontFamily="mono">
                  {systemInfo?.services?.reposPath ?? '-'}
                </Text>
              </Td>
            </Tr>
            <Tr>
              <Td>
                <HStack spacing={2}>
                  <Icon as={FiHome} color="myGray.500" />
                  <Text fontWeight="medium">{t('admin.homeDir')}</Text>
                </HStack>
              </Td>
              <Td>
                <Text fontSize="sm" color="myGray.600" fontFamily="mono">
                  {systemInfo?.services?.homeDir ?? '-'}
                </Text>
              </Td>
            </Tr>
          </Tbody>
        </Table>
      </Box>

    </VStack>
  )
}
