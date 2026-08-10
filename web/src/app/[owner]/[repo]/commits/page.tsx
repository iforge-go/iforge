'use client'

import { Text } from '@chakra-ui/react'
import { useSearchParams, useParams } from 'next/navigation'
import { Suspense } from 'react'
import { CommitsList } from './CommitsList'
import { useI18n } from '@/contexts/I18nContext'

function CommitsContent() {
  const params = useParams()
  const searchParams = useSearchParams()
  const owner = params.owner as string
  const repoName = params.repo as string
  const branch = searchParams.get('branch') || 'main'

  return <CommitsList owner={owner} repoName={repoName} branch={branch} />
}

export default function CommitsPage() {
  const { t } = useI18n()
  return (
    <Suspense fallback={<Text>{t('common.loading')}</Text>}>
      <CommitsContent />
    </Suspense>
  )
}
