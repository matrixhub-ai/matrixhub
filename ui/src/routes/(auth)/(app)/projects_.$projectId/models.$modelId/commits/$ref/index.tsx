import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { modelCommitsQueryOptions } from '@/features/models/models.query'
import { ModelCommitsPage } from '@/features/models/pages/ModelCommitsPage'

const commitsSearchSchema = z.object({
  page: z.coerce.number().int().positive().optional().transform(v => v ?? 1).catch(1),
})

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId/commits/$ref/',
)({
  validateSearch: commitsSearchSchema,
  loaderDeps: ({ search }) => ({
    page: search.page,
  }),
  loader: async ({
    context,
    params,
    deps,
  }) => {
    await context.queryClient.ensureQueryData(modelCommitsQueryOptions(params.projectId, params.modelId, {
      revision: params.ref,
      page: deps.page,
    }))
  },
  component: ModelCommitsPage,
})
