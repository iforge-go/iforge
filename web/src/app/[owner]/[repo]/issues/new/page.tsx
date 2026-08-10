'use client'

import {
  Heading,
  Text,
  VStack,
  HStack,
  Box,
  Flex,
  FormControl,
  FormLabel,
  Input,
  Button,
  Badge,
  Icon,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
} from '@chakra-ui/react'
import { useState, useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { api, Label } from '@/lib/api'
import { FiPlus, FiX } from 'react-icons/fi'
import { useGithubToast } from '@/app/providers'
import { useI18n } from '@/contexts/I18nContext'
import { MarkdownEditor } from '@/components/MarkdownEditor'

function getTextColor(bgColor: string): string {
  const hex = bgColor.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  const brightness = (r * 299 + g * 587 + b * 114) / 1000
  return brightness > 128 ? 'myGray.800' : 'white'
}

export default function NewIssuePage() {
  const params = useParams()
  const router = useRouter()
  const owner = params.owner as string
  const repoName = params.repo as string
  const toast = useGithubToast()
  const { t } = useI18n()

  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [loading, setLoading] = useState(false)
  const [allLabels, setAllLabels] = useState<Label[]>([])
  const [selectedLabelIds, setSelectedLabelIds] = useState<number[]>([])

  useEffect(() => {
    api.listLabels(owner, repoName)
      .then((labels) => setAllLabels(labels || []))
      .catch(() => setAllLabels([]))
  }, [owner, repoName])

  const handleSubmit = async () => {
    if (!title.trim()) {
      toast({
        title: t('repo.enterTitle'),
        status: 'warning',
        duration: 2000,
      })
      return
    }

    setLoading(true)
    try {
      const issue = await api.createIssue(owner, repoName, title, content)
      for (const labelId of selectedLabelIds) {
        await api.addLabelToIssue(owner, repoName, issue.issueId, labelId)
      }
      toast({
        title: t('repo.issueCreated'),
        status: 'success',
        duration: 2000,
      })
      router.push(`/${owner}/${repoName}/issues/${issue.issueId}`)
    } catch (error: any) {
      toast({
        title: t('repo.createFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }

  const toggleLabel = (labelId: number) => {
    setSelectedLabelIds(prev =>
      prev.includes(labelId)
        ? prev.filter(id => id !== labelId)
        : [...prev, labelId]
    )
  }

  return (
    <Flex gap={6} align="start">
      {/* 主内容区 */}
      <VStack spacing={6} align="stretch" flex={1}>
        <Heading size="lg">{t('repo.newIssue')}</Heading>

        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
          <Box p={6}>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('repo.title')}</FormLabel>
                <Input
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder={t('repo.issueTitlePlaceholder')}
                />
              </FormControl>

              <FormControl>
                <FormLabel>{t('repo.content')}</FormLabel>
                <MarkdownEditor
                  value={content}
                  onChange={setContent}
                  placeholder={t('repo.issueContentPlaceholder')}
                  rows={10}
                />
              </FormControl>

              <HStack spacing={4} w="full" justify="flex-end">
                <Button
                  variant="whiteBase"
                  onClick={() => router.back()}
                >
                  {t('repo.cancel')}
                </Button>
                <Button
                  variant="primary"
                  onClick={handleSubmit}
                  isLoading={loading}
                >
                  {t('repo.createIssue')}
                </Button>
              </HStack>
            </VStack>
          </Box>
        </Box>
      </VStack>

      {/* 侧边栏 - 标签 */}
      <VStack w="250px" align="stretch" spacing={4}>
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
          <Box p={4}>
            <VStack spacing={3} align="stretch">
              <HStack justify="space-between">
                <Text fontWeight="semibold" fontSize="sm">{t('repo.labelsSection')}</Text>
                <Menu>
                  <MenuButton
                    as={Button}
                    size="xs"
                    variant="ghost"
                    leftIcon={<Icon as={FiPlus} />}
                  >
                    {t('repo.add')}
                  </MenuButton>
                  <MenuList>
                    {allLabels.length === 0 ? (
                      <MenuItem isDisabled>{t('repo.noLabels')}</MenuItem>
                    ) : (
                      allLabels
                        .filter(label => !selectedLabelIds.includes(label.labelId))
                        .map(label => (
                          <MenuItem
                            key={label.labelId}
                            onClick={() => toggleLabel(label.labelId)}
                          >
                            <HStack spacing={2}>
                              <Badge
                                bg={`#${label.color}`}
                                color={getTextColor(label.color)}
                                fontSize="xs"
                                px={2}
                                py={0.5}
                                borderRadius="md"
                              >
                                {label.labelName}
                              </Badge>
                            </HStack>
                          </MenuItem>
                        ))
                    )}
                  </MenuList>
                </Menu>
              </HStack>
              {selectedLabelIds.length === 0 ? (
                <Text fontSize="sm" color="myGray.500">{t('repo.noLabels')}</Text>
              ) : (
                <VStack spacing={2} align="stretch">
                  {selectedLabelIds.map((labelId) => {
                    const label = allLabels.find(l => l.labelId === labelId)
                    if (!label) return null
                    return (
                      <HStack key={labelId} justify="space-between">
                        <Badge
                          bg={`#${label.color}`}
                          color={getTextColor(label.color)}
                          fontSize="xs"
                          px={2}
                          py={1}
                          borderRadius="md"
                          flex={1}
                        >
                          {label.labelName}
                        </Badge>
                        <Button
                          size="xs"
                          variant="ghost"
                          onClick={() => toggleLabel(labelId)}
                        >
                          <Icon as={FiX} />
                        </Button>
                      </HStack>
                    )
                  })}
                </VStack>
              )}
            </VStack>
          </Box>
        </Box>
      </VStack>
    </Flex>
  )
}
