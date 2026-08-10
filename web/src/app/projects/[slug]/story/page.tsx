'use client'

import { useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'

export default function StoryListPage() {
  const params = useParams()
  const router = useRouter()
  const projectSlug = params.slug as string

  useEffect(() => {
    router.replace(`/projects/${projectSlug}/backlog`)
  }, [router, projectSlug])

  return null
}
