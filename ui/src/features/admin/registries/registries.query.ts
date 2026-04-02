import { Registries } from '@matrixhub/api-ts/v1alpha1/registry.pb'
import {
  keepPreviousData,
  queryOptions,
  useQuery,
} from '@tanstack/react-query'

import {
  DEFAULT_REGISTRIES_PAGE_SIZE,
  type RegistriesSearch,
} from './registries.schema'

export const adminRegistryKeys = {
  all: ['admin', 'registries'] as const,
  lists: () => [...adminRegistryKeys.all, 'list'] as const,
  list: (search: RegistriesSearch) => [...adminRegistryKeys.lists(), search] as const,
}

export function registriesQueryOptions(search: RegistriesSearch) {
  return queryOptions({
    queryKey: adminRegistryKeys.list(search),
    queryFn: async () => {
      const response = await Registries.ListRegistries({
        page: search.page,
        pageSize: DEFAULT_REGISTRIES_PAGE_SIZE,
        search: search.query,
      })

      return {
        registries: response.registries ?? [],
        pagination: response.pagination,
      }
    },
  })
}

// -- Custom hook --

export function useRegistries(search: RegistriesSearch) {
  return useQuery({
    ...registriesQueryOptions(search),
    placeholderData: keepPreviousData,
  })
}
