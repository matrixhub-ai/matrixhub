import { createFileRoute } from '@tanstack/react-router'

import { modelCommitQueryOptions } from '@/features/models/models.query'
import { ModelCommitDetailPage } from '@/features/models/pages/ModelCommitDetailPage'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId/commit/$commitId/',
)({
  loader: async ({
    context,
    params,
  }) => {
    await context.queryClient.ensureQueryData(
      modelCommitQueryOptions(params.projectId, params.modelId, params.commitId),
    )
  },
  component: ModelCommitDetailPage,
})
