import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { modelCommitQueryOptions } from '@/features/models/models.query'
import { ModelCommitDetailPage } from '@/features/models/pages/ModelCommitDetailPage'

const commitDetailSearchSchema = z.object({
  branch: z.string().optional(),
})

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId/commit/$commitId/',
)({
  validateSearch: commitDetailSearchSchema,
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
