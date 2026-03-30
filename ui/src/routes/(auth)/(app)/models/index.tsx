import { createFileRoute, stripSearchParams } from '@tanstack/react-router'
import { z } from 'zod'

import { catalogModelsQueryOptions } from '@/features/models/models.query.ts'
import { ModelsPage } from '@/features/models/pages/ModelsPage'

const defaults = {
  q: '',
  sort: 'updatedAt' as const,
  order: 'desc' as const,
  page: 1,
}

const modelsSearchSchema = z.object({
  q: z.string().trim().optional().default(defaults.q),
  sort: z.literal('updatedAt').optional().default(defaults.sort),
  order: z.enum(['asc', 'desc']).optional().default(defaults.order),
  page: z.coerce.number().int().positive().optional().default(defaults.page),
  task: z.string().optional(),
  library: z.string().optional(),
  project: z.string().optional(),
})

export type ModelsCatalogSearch = z.infer<typeof modelsSearchSchema>

export const Route = createFileRoute('/(auth)/(app)/models/')({
  validateSearch: modelsSearchSchema,
  search: {
    middlewares: [stripSearchParams(defaults)],
  },
  loaderDeps: ({ search }) => search,
  loader({
    context, deps,
  }) {
    context.queryClient.prefetchQuery(catalogModelsQueryOptions(deps))
  },
  component: ModelsPage,
})
