import { createFileRoute, stripSearchParams } from '@tanstack/react-router'
import { z } from 'zod'

import { projectModelsQueryOptions } from '@/features/models/models.query'
import { ProjectModelsPage } from '@/features/projects/pages/ProjectModelsPage'

// -- URL search schema (route concern) --

const defaults = {
  q: '',
  sort: 'updatedAt' as const,
  order: 'desc' as const,
  page: 1,
}

const modelsSearchSchema = z.object({
  q: z.string().trim().optional().default(defaults.q).catch(defaults.q),
  sort: z.literal('updatedAt').optional().default(defaults.sort).catch(defaults.sort),
  order: z.enum(['asc', 'desc']).optional().default(defaults.order).catch(defaults.order),
  page: z.coerce.number().int().positive().optional().default(defaults.page).catch(defaults.page),
})

export type ProjectModelsSearch = z.infer<typeof modelsSearchSchema>

// -- Route definition --

export const Route = createFileRoute(
  '/(auth)/(app)/projects/$projectId/models/',
)({
  validateSearch: modelsSearchSchema,
  search: {
    middlewares: [stripSearchParams(defaults)],
  },
  loaderDeps: ({ search }) => search,
  loader({
    context,
    params,
    deps,
  }) {
    context.queryClient.prefetchQuery(projectModelsQueryOptions(params.projectId, deps))
  },
  component: RouteComponent,
})

// -- Component --

function RouteComponent() {
  return <ProjectModelsPage />
}
