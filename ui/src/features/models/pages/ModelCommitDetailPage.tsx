import { Stack } from '@mantine/core'
import { useSuspenseQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'

import { modelCommitQueryOptions } from '@/features/models/models.query'
import { CommitDetail } from '@/shared/components/commit-detail/CommitDetail.tsx'

const { useParams } = getRouteApi('/(auth)/(app)/projects_/$projectId/models/$modelId/commit/$commitId/')

export function ModelCommitDetailPage() {
  const {
    projectId,
    modelId,
    commitId,
  } = useParams()
  const { data: commit } = useSuspenseQuery(modelCommitQueryOptions(projectId, modelId, commitId))

  return (
    <Stack pt="sm" gap="sm">
      <CommitDetail commit={commit} />
    </Stack>
  )
}
