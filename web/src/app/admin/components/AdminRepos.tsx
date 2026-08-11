'use client'

import { Box, Heading, Text, Button, VStack, HStack, Table, Thead, Tbody, Tr, Th, Td, Badge, Icon, Modal, ModalOverlay, ModalContent, ModalHeader, ModalBody, ModalFooter, ModalCloseButton, FormControl, FormLabel, Input } from '@chakra-ui/react'
import { useEffect, useState, useCallback } from 'react'
import { api, Repository } from '@/lib/api'
import { FiFolder, FiChevronLeft, FiChevronRight } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'

export default function AdminRepos() {
  const toast = useGithubToast()
  const { t, locale } = useI18n()
  const dateLocale = locale === 'zh' ? 'zh-CN' : 'en-US'
  const [repos, setRepos] = useState<Repository[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [limit] = useState(20)
  const [deletingRepo, setDeletingRepo] = useState<Repository | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState('')

  const loadRepos = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.adminListRepos(page, limit)
      setRepos(data.repositories)
      setTotal(data.total)
    } catch (error) {
      console.error('Failed to load repos:', error)
    } finally {
      setLoading(false)
    }
  }, [page, limit])

  useEffect(() => {
    loadRepos()
  }, [loadRepos])

  const handleConfirmDelete = async () => {
    if (!deletingRepo) return
    const expected = `${deletingRepo.userName}/${deletingRepo.repositoryName}`
    if (deleteConfirm !== expected) {
      toast({ title: '确认失败', description: `请输入 "${expected}" 以确认删除`, status: 'error', duration: 3000 })
      return
    }
    try {
      await api.adminDeleteRepo(deletingRepo.userName, deletingRepo.repositoryName)
      toast({ title: '仓库已删除', status: 'success', duration: 2000 })
      await loadRepos()
      setDeletingRepo(null)
      setDeleteConfirm('')
    } catch (error: any) {
      toast({ title: '删除失败', description: error.message, status: 'error', duration: 3000 })
    }
  }

  const totalPages = Math.ceil(total / limit)

  if (loading) {
    return <Text>{t('common.loading')}</Text>
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" p={6}>
      <HStack justify="space-between" mb={4}>
        <Heading size="md">{t('admin.repos')}</Heading>
        <Text fontSize="sm" color="myGray.600">
          共 {total} 个仓库
        </Text>
      </HStack>
      <Table variant="simple" borderColor="myGray.200">
        <Thead>
          <Tr bg="myGray.100">
            <Th>仓库</Th>
            <Th>所有者</Th>
            <Th>可见性</Th>
            <Th>默认分支</Th>
            <Th>创建时间</Th>
            <Th>{t('admin.actions')}</Th>
          </Tr>
        </Thead>
        <Tbody>
          {repos.map((repo) => (
            <Tr key={`${repo.userName}/${repo.repositoryName}`}>
              <Td>
                <HStack spacing={2}>
                  <Icon as={FiFolder} color="myGray.500" />
                  <Text fontWeight="medium">{repo.repositoryName}</Text>
                  {repo.isArchived && <Badge colorScheme="red">已归档</Badge>}
                  {repo.isTemplate && <Badge colorScheme="purple">模板</Badge>}
                </HStack>
              </Td>
              <Td>
                <Text fontSize="sm" color="myGray.600">
                  {repo.userName}
                </Text>
              </Td>
              <Td>
                <Badge colorScheme={repo.isPrivate ? 'yellow' : 'green'}>
                  {repo.isPrivate ? t('repo.private') : t('repo.public')}
                </Badge>
              </Td>
              <Td>
                <Text fontSize="sm" fontFamily="mono">
                  {repo.defaultBranch}
                </Text>
              </Td>
              <Td>
                <Text fontSize="sm">{new Date(repo.registeredDate).toLocaleDateString(dateLocale)}</Text>
              </Td>
              <Td>
                <Button size="xs" variant="danger" onClick={() => setDeletingRepo(repo)}>
                  {t('common.delete')}
                </Button>
              </Td>
            </Tr>
          ))}
          {repos.length === 0 && (
            <Tr>
              <Td colSpan={6}>
                <Text textAlign="center" color="myGray.500" py={4}>
                  {t('admin.noRepos')}
                </Text>
              </Td>
            </Tr>
          )}
        </Tbody>
      </Table>

      {/* 分页 */}
      {totalPages > 1 && (
        <HStack justify="center" spacing={4} mt={6}>
          <Button
            size="sm"
            variant="whiteBase"
            leftIcon={<Icon as={FiChevronLeft} />}
            onClick={() => setPage(Math.max(1, page - 1))}
            isDisabled={page === 1}
          >
            上一页
          </Button>
          <Text fontSize="sm" color="myGray.600">
            第 {page} / {totalPages} 页
          </Text>
          <Button
            size="sm"
            variant="whiteBase"
            rightIcon={<Icon as={FiChevronRight} />}
            onClick={() => setPage(Math.min(totalPages, page + 1))}
            isDisabled={page === totalPages}
          >
            下一页
          </Button>
        </HStack>
      )}

      {/* 删除仓库确认 Modal */}
      <Modal isOpen={deletingRepo !== null} onClose={() => { setDeletingRepo(null); setDeleteConfirm('') }}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>删除仓库</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4} align="stretch">
              <Text>
                确定要删除仓库 <strong>{deletingRepo?.userName}/{deletingRepo?.repositoryName}</strong> 吗？
              </Text>
              <Text fontSize="sm" color="red.600">
                此操作不可恢复，仓库的所有数据将被永久删除。
              </Text>
              <FormControl>
                <FormLabel>
                  请输入 <strong>{deletingRepo?.userName}/{deletingRepo?.repositoryName}</strong> 以确认删除
                </FormLabel>
                <Input
                  value={deleteConfirm}
                  onChange={(e) => setDeleteConfirm(e.target.value)}
                  placeholder={`${deletingRepo?.userName}/${deletingRepo?.repositoryName}`}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => { setDeletingRepo(null); setDeleteConfirm('') }}>
              {t('common.cancel')}
            </Button>
            <Button
              variant="danger"
              onClick={handleConfirmDelete}
              isDisabled={deleteConfirm !== `${deletingRepo?.userName}/${deletingRepo?.repositoryName}`}
            >
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  )
}
