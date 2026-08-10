'use client'

import { Box, Tabs, TabList, TabPanels, Tab, TabPanel } from '@chakra-ui/react'
import { useState, useEffect } from 'react'
import { api, TableInfo } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import DatabaseTableList from './DatabaseTableList'
import DatabaseTableDetail from './DatabaseTableDetail'
import DatabaseSqlQuery from './DatabaseSqlQuery'

export default function DatabaseViewer() {
  const { t } = useI18n()
  const [tables, setTables] = useState<TableInfo[]>([])
  const [activeTable, setActiveTable] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadTables()
  }, [])

  const loadTables = async () => {
    try {
      setLoading(true)
      const data = await api.listDbTables()
      setTables(data || [])
    } catch (error) {
      console.error('Failed to load tables:', error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" bg="white">
      <Tabs variant="enclosed">
        <TabList px={4} pt={2}>
          <Tab>{t('admin.dbTables')}</Tab>
          <Tab>{t('admin.dbSqlQuery')}</Tab>
        </TabList>
        <TabPanels>
          <TabPanel px={0} py={0}>
            <div style={{ display: 'grid', gridTemplateColumns: '280px 1fr', height: '600px' }}>
              <div style={{ borderRight: '1px solid #e2e8f0', overflowY: 'auto' }}>
                <DatabaseTableList
                  tables={tables}
                  activeTable={activeTable}
                  onSelectTable={setActiveTable}
                  loading={loading}
                />
              </div>
              <div style={{ overflow: 'auto' }}>
                <DatabaseTableDetail
                  tableName={activeTable}
                  onRefreshTables={loadTables}
                />
              </div>
            </div>
          </TabPanel>
          <TabPanel>
            <DatabaseSqlQuery />
          </TabPanel>
        </TabPanels>
      </Tabs>
    </Box>
  )
}
