import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { ModelsPage } from '@/features/models/pages/ModelsPage'

const modelsSearchSchema = z.object({
  q: z.string().trim().optional(),
  sort: z.literal('updatedAt').optional(),
  order: z.enum(['asc', 'desc']).optional(),
  page: z.coerce.number().int().positive().optional(),
  task: z.string().optional(),
  library: z.string().optional(),
  project: z.string().optional(),
})

export type ModelsCatalogSearch = z.infer<typeof modelsSearchSchema>

export const Route = createFileRoute('/(auth)/(app)/models/')({
  validateSearch: modelsSearchSchema,
  component: ModelsPage,
})
