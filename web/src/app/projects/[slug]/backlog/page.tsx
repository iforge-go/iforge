'use client'

import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import {
  Box,
  Text,
  Spinner,
  Flex,
  Stack,
} from '@chakra-ui/react'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import BacklogTab from '../components/BacklogTab'
import { useProject } from '../ProjectContext'

export default function BacklogPage() {
  const params = useParams()
  const projectSlug = params.slug as string
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()
  const { project, loading: projectLoading } = useProject()
  // 单一数据源:allStories 包含所有 Story(含 sprintSlug 字段区分归属)
  const [allStories, setAllStories] = useState<any[]>([])
  const [sprints, setSprints] = useState<any[]>([])
  const [epics, setEpics] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!user || !projectSlug) return
    Promise.all([
      api.getUserStories(projectSlug).then((data) => setAllStories(data || [])).catch(() => {}),
      api.getSprints(projectSlug).then((data) => setSprints(data || [])).catch(() => {}),
      api.getEpics(projectSlug).then((data) => setEpics(data || [])).catch(() => {}),
    ]).finally(() => setLoading(false))
  }, [user, projectSlug])

  if (authLoading) return null

  if (!user) {
    return (
      <Box py={20}>
        <Text textAlign="center">{t('common.pleaseLogin')}</Text>
      </Box>
    )
  }

  if (projectLoading || loading) {
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
      <BacklogTab
        projectSlug={projectSlug}
        allStories={allStories}
        sprints={sprints}
        epics={epics}
        onAllStoriesChange={setAllStories}
        onSprintsChange={setSprints}
        onEpicsChange={setEpics}
      />
    </Stack>
  )
}
