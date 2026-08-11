'use client'

import {
  Box,
  Textarea,
  Button,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Text,
  Alert,
  AlertIcon,
  VStack,
} from '@chakra-ui/react'
import { useState } from 'react'
import { api, QueryResult } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'

export default function DatabaseSqlQuery() {
  const { t } = useI18n()
  const [sqlInput, setSqlInput] = useState('')
  const [result, setResult] = useState<QueryResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const handleExecute = async () => {
    if (!sqlInput.trim()) return

    try {
      setLoading(true)
      setError(null)
      setResult(null)
      const data = await api.executeDbQuery(sqlInput.trim())
      setResult(data)
    } catch (err: any) {
      setError(err.message || 'Query failed')
    } finally {
      setLoading(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.ctrlKey && e.key === 'Enter') {
      handleExecute()
    }
  }

  return (
    <VStack spacing={4} align="stretch">
      <Box>
        <Text fontSize="sm" color="myGray.600" mb={2}>
          {t('admin.dbEnterSql')}
        </Text>
        <Textarea
          value={sqlInput}
          onChange={(e) => setSqlInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="SELECT * FROM users LIMIT 10"
          fontFamily="mono"
          fontSize="sm"
          rows={6}
        />
      </Box>

      <Button
        onClick={handleExecute}
        isLoading={loading}
        loadingText={t('admin.dbExecuting')}
        variant="primary"
        alignSelf="flex-start"
      >
        {t('admin.dbExecute')}
      </Button>

      {error && (
        <Alert status="error" borderRadius="md">
          <AlertIcon />
          <Box>
            <Text fontWeight="bold">{t('admin.dbQueryError')}</Text>
            <Text fontSize="sm">{error}</Text>
          </Box>
        </Alert>
      )}

      {result && (
        <Box>
          <Text fontSize="sm" fontWeight="bold" mb={2}>
            {t('admin.dbQueryResult')} ({result.count} {t('admin.dbRows')})
          </Text>
          {result.rows.length === 0 ? (
            <Text color="myGray.600">{t('admin.dbNoData')}</Text>
          ) : (
            <Box overflowX="auto" borderWidth="1px" borderRadius="md">
              <Table variant="simple" size="sm">
                <Thead>
                  <Tr>
                    {result.columns.map((col) => (
                      <Th key={col}>{col}</Th>
                    ))}
                  </Tr>
                </Thead>
                <Tbody>
                  {result.rows.map((row, rowIdx) => (
                    <Tr key={rowIdx}>
                      {row.map((cell, cellIdx) => (
                        <Td key={cellIdx} maxW="300px" isTruncated>
                          {cell === null || cell === undefined ? (
                            <Text color="myGray.500">NULL</Text>
                          ) : (
                            <Text fontSize="sm">{String(cell)}</Text>
                          )}
                        </Td>
                      ))}
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </Box>
          )}
        </Box>
      )}
    </VStack>
  )
}
