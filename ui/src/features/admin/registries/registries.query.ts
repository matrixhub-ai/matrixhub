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
  // Only paging/search affect the response, so UI-only params (e.g. `create`)
  // must not become part of the cache key.
  list: (search: RegistriesSearch) => [
    ...adminRegistryKeys.lists(),
    {
      page: search.page,
      query: search.query,
    },
  ] as const,
  allList: () => [...adminRegistryKeys.lists(), 'all'] as const,
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

/**
 * Every registry in one page, for selectors such as project creation.
 * Shares the `lists()` key prefix so registry mutations invalidate it too.
 */
export function allRegistriesQueryOptions() {
  return queryOptions({
    queryKey: adminRegistryKeys.allList(),
    queryFn: async () => {
      const response = await Registries.ListRegistries({ pageSize: -1 })

      return response.registries ?? []
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
