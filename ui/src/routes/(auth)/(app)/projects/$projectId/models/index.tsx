import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { modelsQueryOptions } from '@/features/models/models.query'
import { ProjectModelsPage } from '@/features/projects/pages/ProjectModelsPage'

// -- URL search schema (route concern) --

const modelsSearchSchema = z.object({
  q: z.string().transform(v => v.trim()).catch(''),
  sort: z.literal('updatedAt').catch('updatedAt'),
  order: z.enum(['asc', 'desc']).catch('desc'),
  page: z.coerce.number().int().positive().catch(1),
})

// -- Route definition --

export const Route = createFileRoute(
  '/(auth)/(app)/projects/$projectId/models/',
)({
  validateSearch: modelsSearchSchema,
  loaderDeps: ({ search }) => search,
  loader: async ({
    context,
    params,
    deps,
  }) => {
    await context.queryClient.ensureQueryData(
      modelsQueryOptions(params.projectId, deps),
    )
  },
  component: RouteComponent,
})

// -- Component --

function RouteComponent() {
  return <ProjectModelsPage />
}
