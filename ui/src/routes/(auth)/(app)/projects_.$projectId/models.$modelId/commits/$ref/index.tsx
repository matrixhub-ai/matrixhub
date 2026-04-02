import { createFileRoute, stripSearchParams } from '@tanstack/react-router'
import { z } from 'zod'

import { modelCommitsQueryOptions } from '@/features/models/models.query'
import { ModelCommitsPage } from '@/features/models/pages/ModelCommitsPage'

const defaults = { page: 1 }

const commitsSearchSchema = z.object({
  page: z.coerce.number().int().positive().optional().default(defaults.page).catch(defaults.page),
})

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId/commits/$ref/',
)({
  validateSearch: commitsSearchSchema,
  search: {
    middlewares: [stripSearchParams(defaults)],
  },
  loaderDeps: ({ search }) => ({
    page: search.page,
  }),
  loader({
    context,
    params,
    deps,
  }) {
    context.queryClient.prefetchQuery(modelCommitsQueryOptions(params.projectId, params.modelId, {
      revision: params.ref,
      page: deps.page,
    }))
  },
  component: ModelCommitsPage,
})
