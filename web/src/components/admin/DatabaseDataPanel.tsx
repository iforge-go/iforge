'use client'

import { Box, Spinner, Text, HStack, Button, Select } from '@chakra-ui/react'
import { useState, useEffect } from 'react'
import { api, QueryResult } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'

interface DatabaseDataPanelProps {
  tableName: string
  queryResult: QueryResult | null
  loading: boolean
  onRefresh: () => void
}

export default function DatabaseDataPanel({
  tableName,
  queryResult,
  loading,
  onRefresh,
}: DatabaseDataPanelProps) {
  const { t } = useI18n()
  const [page, setPage] = useState(0)
  const [pageSize, setPageSize] = useState(20)
  const [data, setData] = useState<QueryResult | null>(queryResult)
  const [loadingData, setLoadingData] = useState(false)

  useEffect(() => {
    setData(queryResult)
    setPage(0)
  }, [queryResult, tableName])

  useEffect(() => {
    if (page === 0) return
    loadData()
  }, [page, pageSize])

  const loadData = async () => {
    try {
      setLoadingData(true)
      const offset = page * pageSize
      const result = await api.queryDbTable(tableName, pageSize, offset)
      setData(result)
    } catch (error) {
      console.error('Failed to load data:', error)
    } finally {
      setLoadingData(false)
    }
  }

  const handlePageSizeChange = (newSize: number) => {
    setPageSize(newSize)
    setPage(0)
  }

  if (loading || loadingData) {
    return (
      <Box p={8} textAlign="center">
        <Spinner size="sm" />
      </Box>
    )
  }

  if (!data || !data.rows || data.rows.length === 0) {
    return (
      <Box p={8} textAlign="center">
        <Text color="myGray.600">{t('admin.dbNoData')}</Text>
      </Box>
    )
  }

  const hasMore = data.count === pageSize

  return (
    <div>
      <div style={{ overflowX: 'auto', border: '1px solid #e2e8f0', borderRadius: '8px' }}>
        <table style={{ borderCollapse: 'collapse', width: `${data.columns.length * 200}px` }}>
          <thead>
            <tr>
              {data.columns.map((col) => (
                <th key={col} style={{ padding: '8px 12px', textAlign: 'left', borderBottom: '2px solid #e2e8f0', whiteSpace: 'nowrap', backgroundColor: '#f7fafc', fontWeight: 600, fontSize: '14px' }}>
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.rows.map((row, rowIdx) => (
              <tr key={rowIdx}>
                {row.map((cell, cellIdx) => (
                  <td key={cellIdx} style={{ padding: '8px 12px', borderBottom: '1px solid #e2e8f0', whiteSpace: 'nowrap', fontSize: '14px' }}>
                    {cell === null || cell === undefined ? (
                      <span style={{ color: '#a0aec0' }}>NULL</span>
                    ) : (
                      <span>{String(cell)}</span>
                    )}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <HStack spacing={4} mt={4} px={4} justify="space-between">
        <HStack spacing={2}>
          <Text fontSize="sm" color="myGray.600">
            {t('admin.dbPageSize')}:
          </Text>
          <Select
            size="sm"
            w="80px"
            value={pageSize}
            onChange={(e) => handlePageSizeChange(Number(e.target.value))}
          >
            <option value={10}>10</option>
            <option value={20}>20</option>
            <option value={50}>50</option>
            <option value={100}>100</option>
          </Select>
        </HStack>

        <HStack spacing={2}>
          <Button
            size="sm"
            onClick={() => setPage(page - 1)}
            isDisabled={page === 0}
          >
            {t('admin.dbPrevPage')}
          </Button>
          <Text fontSize="sm" color="myGray.600">
            {t('admin.dbPageInfo', { page: page + 1, total: data.count })}
          </Text>
          <Button
            size="sm"
            onClick={() => setPage(page + 1)}
            isDisabled={!hasMore}
          >
            {t('admin.dbNextPage')}
          </Button>
        </HStack>
      </HStack>
    </div>
  )
}
