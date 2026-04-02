import { SyncPolicy } from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import {
  keepPreviousData,
  queryOptions,
  useQuery,
} from '@tanstack/react-query'

import {
  DEFAULT_REPLICATIONS_PAGE_SIZE,
  type ReplicationsSearch,
} from './replications.schema'

export const adminReplicationKeys = {
  all: ['admin', 'replications'] as const,
  lists: () => [...adminReplicationKeys.all, 'list'] as const,
  list: (search: ReplicationsSearch) => [...adminReplicationKeys.lists(), search] as const,
}

export function replicationsQueryOptions(search: ReplicationsSearch) {
  return queryOptions({
    queryKey: adminReplicationKeys.list(search),
    queryFn: async () => {
      const response = await SyncPolicy.ListSyncPolicies({
        page: search.page,
        pageSize: DEFAULT_REPLICATIONS_PAGE_SIZE,
        search: search.query,
      })

      return {
        replications: response.syncPolicies ?? [],
        pagination: response.pagination,
      }
    },
  })
}

// -- Custom hook --

export function useReplications(search: ReplicationsSearch) {
  return useQuery({
    ...replicationsQueryOptions(search),
    placeholderData: keepPreviousData,
  })
}
