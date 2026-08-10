'use client'

import { Box, VStack, Text, Spinner, Badge } from '@chakra-ui/react'
import { TableInfo } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'

interface DatabaseTableListProps {
  tables: TableInfo[]
  activeTable: string | null
  onSelectTable: (tableName: string) => void
  loading: boolean
}

export default function DatabaseTableList({
  tables,
  activeTable,
  onSelectTable,
  loading,
}: DatabaseTableListProps) {
  const { t } = useI18n()

  if (loading) {
    return (
      <Box p={4} textAlign="center">
        <Spinner size="sm" />
        <Text mt={2} fontSize="sm" color="myGray.600">
          {t('admin.dbLoadingTables')}
        </Text>
      </Box>
    )
  }

  if (tables.length === 0) {
    return (
      <Box p={4} textAlign="center">
        <Text fontSize="sm" color="myGray.600">
          {t('admin.dbNoData')}
        </Text>
      </Box>
    )
  }

  return (
    <VStack spacing={0} align="stretch">
      {tables.map((table) => (
        <Box
          key={table.name}
          px={4}
          py={3}
          cursor="pointer"
          bg={activeTable === table.name ? 'primary.50' : 'transparent'}
          borderLeftWidth={activeTable === table.name ? '3px' : '0px'}
          borderLeftColor="primary.500"
          _hover={{
            bg: activeTable === table.name ? 'primary.50' : 'myGray.50',
          }}
          onClick={() => onSelectTable(table.name)}
          transition="all 0.2s"
        >
          <Box display="flex" justifyContent="space-between" alignItems="center">
            <Text fontSize="sm" fontWeight="medium" color="gray.800">
              {table.name}
            </Text>
            <Badge colorScheme="gray" fontSize="xs">
              {t('admin.dbRowsCount', { count: table.rowCount })}
            </Badge>
          </Box>
        </Box>
      ))}
    </VStack>
  )
}
