'use client'

import { useParams } from 'next/navigation'
import {
  Box,
  Text,
  Spinner,
  Flex,
  Stack,
} from '@chakra-ui/react'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import MembersTab from '../components/MembersTab'
import { useProject } from '../ProjectContext'

export default function MembersPage() {
  const params = useParams()
  const projectSlug = params.slug as string
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()
  // 复用 ProjectContext 的 members/setMembers/canManageProject（单一数据源）
  const { project, loading: projectLoading, members, setMembers, canManageProject } = useProject()

  if (authLoading) return null

  if (!user) {
    return (
      <Box py={20}>
        <Text textAlign="center">{t('common.pleaseLogin')}</Text>
      </Box>
    )
  }

  if (projectLoading) {
    return (
      <Flex justify="center" align="center" minH="60vh">
        <Spinner size="xl" color="blue.500" />
      </Flex>
    )
  }

  if (!project) {
    return null
  }

  return (
    <Stack spacing={6}>
      <MembersTab
        projectSlug={projectSlug}
        members={members}
        onMembersChange={setMembers}
        canManage={canManageProject}
      />
    </Stack>
  )
}
