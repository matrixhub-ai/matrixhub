import { z } from 'zod'

export const DEFAULT_REGISTRIES_PAGE = 1
export const DEFAULT_REGISTRIES_PAGE_SIZE = 10

export const registriesSearchDefaults = {
  page: DEFAULT_REGISTRIES_PAGE,
  query: undefined as string | undefined,
}

export const registriesSearchSchema = z.object({
  page: z.coerce.number().int().positive().default(registriesSearchDefaults.page).catch(registriesSearchDefaults.page),

  query: z.string().trim().optional().catch(undefined),
})

export type RegistriesSearch = z.infer<typeof registriesSearchSchema>
