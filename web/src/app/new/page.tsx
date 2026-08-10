import { Suspense } from 'react'
import NewRepoContent from './NewRepoContent'

export default function NewRepoPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <NewRepoContent />
    </Suspense>
  )
}
