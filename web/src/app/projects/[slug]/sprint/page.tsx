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
import SprintsTab from '../components/SprintsTab'
import { useProject } from '../ProjectContext'

export default function SprintListPage() {
  const params = useParams()
  const projectSlug = params.slug as string
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()
  const { project, loading: projectLoading } = useProject()
  const [sprints, setSprints] = useState<any[]>([])
  const [tasks, setTasks] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!user || !projectSlug) return
    Promise.all([
      api.getSprints(projectSlug).then((data) => setSprints(data || [])).catch(() => {}),
      api.getTasks(projectSlug).then((data) => setTasks(data || [])).catch(() => {}),
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
      <SprintsTab
        projectSlug={projectSlug}
        sprints={sprints}
        tasks={tasks}
        onSprintsChange={setSprints}
      />
    </Stack>
  )
}
