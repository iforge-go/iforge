'use client'

import { Box, Tabs, TabList, TabPanels, Tab, TabPanel, Text, VStack, Icon } from '@chakra-ui/react'
import { useState, useEffect, useCallback } from 'react'
import { api, ColumnInfo, QueryResult } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { FiDatabase } from 'react-icons/fi'
import DatabaseSchemaPanel from './DatabaseSchemaPanel'
import DatabaseDataPanel from './DatabaseDataPanel'

interface DatabaseTableDetailProps {
  tableName: string | null
  onRefreshTables?: () => void
}

export default function DatabaseTableDetail({
  tableName,
  onRefreshTables: _onRefreshTables,
}: DatabaseTableDetailProps) {
  const { t } = useI18n()
  const [schema, setSchema] = useState<ColumnInfo[]>([])
  const [queryResult, setQueryResult] = useState<QueryResult | null>(null)
  const [loading, setLoading] = useState(false)

  const loadTableData = useCallback(async () => {
    if (!tableName) return

    try {
      setLoading(true)
      const [schemaData, queryData] = await Promise.all([
        api.getDbTableSchema(tableName),
        api.queryDbTable(tableName, 20, 0),
      ])
      setSchema(schemaData || [])
      setQueryResult(queryData || null)
    } catch (error) {
      console.error('Failed to load table data:', error)
    } finally {
      setLoading(false)
    }
  }, [tableName])

  useEffect(() => {
    if (tableName) {
      loadTableData()
    }
  }, [tableName, loadTableData])

  if (!tableName) {
    return (
      <Box p={8} textAlign="center">
        <VStack spacing={4}>
          <Icon as={FiDatabase} w={12} h={12} color="myGray.400" />
          <Text color="myGray.600">{t('admin.dbSelectTable')}</Text>
        </VStack>
      </Box>
    )
  }

  return (
    <Tabs variant="soft-rounded" size="sm" px={4} pt={3}>
      <TabList>
        <Tab>{t('admin.dbData')}</Tab>
        <Tab>{t('admin.dbSchema')}</Tab>
      </TabList>
      <TabPanels>
        <TabPanel px={0} py={4}>
          <DatabaseDataPanel
            tableName={tableName}
            queryResult={queryResult}
            loading={loading}
            onRefresh={loadTableData}
          />
        </TabPanel>
        <TabPanel px={0} py={4}>
          <DatabaseSchemaPanel schema={schema} loading={loading} />
        </TabPanel>
      </TabPanels>
    </Tabs>
  )
}
