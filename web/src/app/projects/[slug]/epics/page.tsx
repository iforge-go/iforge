'use client'

import { Suspense, useState, useEffect } from 'react'
import { useParams, useSearchParams, useRouter } from 'next/navigation'
import {
  Box,
  Text,
  Spinner,
  Flex,
  Stack,
  HStack,
  Button,
  Heading,
} from '@chakra-ui/react'
import { FiList, FiCalendar, FiPlus } from 'react-icons/fi'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import EpicsTab from './components/EpicsTab'
import RoadmapView from './components/RoadmapView'
import { useProject } from '../ProjectContext'
import { Epic } from '@/lib/types'

function EpicsListContent() {
  const params = useParams()
  const projectSlug = params.slug as string
  const { t } = useI18n()
  const { user, authLoading } = useCurrentUser()
  const { project, loading: projectLoading, canEditScrum } = useProject()
  const [epics, setEpics] = useState<Epic[]>([])
  const [loading, setLoading] = useState(true)
  const [newEpicTrigger, setNewEpicTrigger] = useState(0)
  const router = useRouter()
  const searchParams = useSearchParams()
  const view = searchParams.get('view') === 'roadmap' ? 'roadmap' : 'list'

  const switchView = (next: 'list' | 'roadmap') => {
    if (next === 'list') {
      router.push(`/projects/${projectSlug}/epics`)
    } else {
      router.push(`/projects/${projectSlug}/epics?view=roadmap`)
    }
  }

  useEffect(() => {
    if (!user || !projectSlug) return
    api.getEpics(projectSlug)
      .then((data) => setEpics(data || []))
      .catch(() => setEpics([]))
      .finally(() => setLoading(false))
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
    <Stack spacing={4}>
      {/* 标题 + 视图切换 + 新建按钮（一行） */}
      <Flex justify="space-between" align="center">
        <Heading size="lg">{t('pms.epics')}</Heading>
        <HStack spacing={2}>
          <Button
            size="sm"
            variant={view === 'list' ? 'solid' : 'outline'}
            colorScheme="blue"
            leftIcon={<FiList />}
            onClick={() => switchView('list')}
          >
            {t('pms.epicListView')}
          </Button>
          <Button
            size="sm"
            variant={view === 'roadmap' ? 'solid' : 'outline'}
            colorScheme="purple"
            leftIcon={<FiCalendar />}
            onClick={() => switchView('roadmap')}
          >
            {t('pms.epicRoadmapView')}
          </Button>
          {canEditScrum && view === 'list' && (
            <Button
              size="sm"
              colorScheme="blue"
              leftIcon={<FiPlus />}
              onClick={() => setNewEpicTrigger(prev => prev + 1)}
            >
              {t('pms.newEpic')}
            </Button>
          )}
        </HStack>
      </Flex>

      {view === 'roadmap' ? (
        <RoadmapView projectSlug={projectSlug} epics={epics} />
      ) : (
        <EpicsTab
          projectSlug={projectSlug}
          epics={epics}
          onEpicsChange={setEpics}
          newEpicTrigger={newEpicTrigger}
        />
      )}
    </Stack>
  )
}

export default function EpicsListPage() {
  return (
    <Suspense fallback={null}>
      <EpicsListContent />
    </Suspense>
  )
}
