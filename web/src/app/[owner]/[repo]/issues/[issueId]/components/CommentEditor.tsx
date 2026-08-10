'use client'

import {
  Box,
  HStack,
  Icon,
  Text,
  VStack,
  Button,
} from '@chakra-ui/react'
import { FiLock } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { MarkdownEditor } from '@/components/MarkdownEditor'

interface CommentEditorProps {
  locked: boolean
  value: string
  onChange: (value: string) => void
  onSubmit: () => void
  submitting: boolean
}

export function CommentEditor({
  locked,
  value,
  onChange,
  onSubmit,
  submitting,
}: CommentEditorProps) {
  const { t } = useI18n()

  return (
    <Box mt={4} mb={4} position="relative">
      <Box position="absolute" left="27px" top="0" bottom="0" w="2px" bg="myGray.200" zIndex={0} />
      {locked ? (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="myGray.50" borderColor="myGray.200" position="relative" zIndex={1}>
          <Box p={4}>
            <HStack spacing={2} color="myGray.500">
              <Icon as={FiLock} w={4} h={4} />
              <Text fontSize="sm">{t('repo.issueLockedNotice')}</Text>
            </HStack>
          </Box>
        </Box>
      ) : (
        <Box borderWidth="1px" borderRadius="lg" bg="white" position="relative" zIndex={1}>
          <Box p={4}>
            <VStack spacing={4} align="stretch">
              <Text fontWeight="medium">{t('repo.addComment')}</Text>
              <MarkdownEditor
                value={value}
                onChange={onChange}
                placeholder={t('repo.addCommentPlaceholder')}
                height={200}
              />
              <Button
                variant="primary"
                onClick={onSubmit}
                isLoading={submitting}
                alignSelf="flex-end"
              >
                {t('repo.submitComment')}
              </Button>
            </VStack>
          </Box>
        </Box>
      )}
    </Box>
  )
}
