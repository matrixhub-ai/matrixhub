import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { projectModelsQueryOptions } from '@/features/models/models.query'
import { ProjectModelsPage } from '@/features/projects/pages/ProjectModelsPage'

// -- URL search schema (route concern) --

const modelsSearchSchema = z.object({
  q: z.string().optional().transform(v => v?.trim() ?? '').catch(''),
  sort: z.literal('updatedAt').optional().transform(v => v ?? 'updatedAt').catch('updatedAt'),
  order: z.enum(['asc', 'desc']).optional().transform(v => v ?? 'desc').catch('desc'),
  page: z.coerce.number().int().positive().optional().transform(v => v ?? 1).catch(1),
})

export type ProjectModelsSearch = z.infer<typeof modelsSearchSchema>

// -- Route definition --

export const Route = createFileRoute(
  '/(auth)/(app)/projects/$projectId/models/',
)({
  validateSearch: modelsSearchSchema,
  loaderDeps: ({ search }) => search,
  loader({
    context,
    params,
    deps,
  }) {
    context.queryClient.fetchQuery(projectModelsQueryOptions(params.projectId, deps))
  },
  component: RouteComponent,
})

// -- Component --

function RouteComponent() {
  return <ProjectModelsPage />
}
