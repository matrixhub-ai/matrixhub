import { z } from 'zod'

export const DEFAULT_REGISTRIES_PAGE = 1
export const DEFAULT_REGISTRIES_PAGE_SIZE = 10

export const registriesSearchSchema = z.object({
  page: z.coerce.number().int().positive().catch(DEFAULT_REGISTRIES_PAGE).default(DEFAULT_REGISTRIES_PAGE),
  query: z.string().trim().optional().catch(undefined),
})

export type RegistriesSearch = z.infer<typeof registriesSearchSchema>
