import { z } from 'zod'

export const DEFAULT_REGISTRIES_PAGE = 1
export const DEFAULT_REGISTRIES_PAGE_SIZE = 10

export const registriesSearchDefaults = {
  page: DEFAULT_REGISTRIES_PAGE,
  query: undefined as string | undefined,
  create: undefined as boolean | undefined,
}

export const registriesSearchSchema = z.object({
  page: z.coerce.number().int().positive().default(registriesSearchDefaults.page).catch(registriesSearchDefaults.page),
  query: z.string().trim().optional().catch(registriesSearchDefaults.query),
  // Opens the create dialog on arrival, e.g. when linked from project creation.
  // Accepts both the parsed boolean and the raw "true"/"false" string, since
  // the router may hand over either. `z.coerce.boolean()` is unusable here:
  // it turns the string "false" into true.
  create: z.union([z.boolean(), z.enum(['true', 'false']).transform(v => v === 'true')])
    .optional()
    .catch(registriesSearchDefaults.create),
})

export type RegistriesSearch = z.infer<typeof registriesSearchSchema>
