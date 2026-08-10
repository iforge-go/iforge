'use client'

import { Box, Table, Thead, Tbody, Tr, Th, Td, Badge, Spinner, Text } from '@chakra-ui/react'
import { ColumnInfo } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'

interface DatabaseSchemaPanelProps {
  schema: ColumnInfo[]
  loading: boolean
}

export default function DatabaseSchemaPanel({ schema, loading }: DatabaseSchemaPanelProps) {
  const { t } = useI18n()

  if (loading) {
    return (
      <Box p={8} textAlign="center">
        <Spinner size="sm" />
      </Box>
    )
  }

  if (schema.length === 0) {
    return (
      <Box p={8} textAlign="center">
        <Text color="myGray.600">{t('admin.dbNoData')}</Text>
      </Box>
    )
  }

  return (
    <Box overflowX="auto">
      <Table variant="simple" size="sm">
        <Thead>
          <Tr>
            <Th>{t('admin.dbColumnName')}</Th>
            <Th>{t('admin.dbColumnType')}</Th>
            <Th>{t('admin.dbColumnNullable')}</Th>
            <Th>{t('admin.dbColumnKey')}</Th>
            <Th>{t('admin.dbColumnDefault')}</Th>
          </Tr>
        </Thead>
        <Tbody>
          {schema.map((column) => (
            <Tr key={column.name}>
              <Td fontWeight="medium">{column.name}</Td>
              <Td>
                <Badge colorScheme="blue" variant="subtle">
                  {column.type}
                </Badge>
              </Td>
              <Td>
                {column.nullable ? (
                  <Badge colorScheme="gray">{t('admin.dbNullable')}</Badge>
                ) : (
                  <Badge colorScheme="red">{t('admin.dbNotNull')}</Badge>
                )}
              </Td>
              <Td>
                {column.key === '1' ? (
                  <Badge colorScheme="green">{t('admin.dbPrimaryKey')}</Badge>
                ) : (
                  <Text color="myGray.500">-</Text>
                )}
              </Td>
              <Td>
                <Text fontSize="sm" color="myGray.700">
                  {column.defaultValue || '-'}
                </Text>
              </Td>
            </Tr>
          ))}
        </Tbody>
      </Table>
    </Box>
  )
}
