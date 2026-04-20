import { SyncPolicy } from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import {
  keepPreviousData,
  queryOptions,
  useQuery,
} from '@tanstack/react-query'

import { DEFAULT_PAGE_SIZE } from '@/utils/constants'

import { type ReplicationExecutionsSearch } from './executions.schema'

export const adminReplicationExecutionKeys = {
  all: ['admin', 'replications', 'executions'] as const,
  lists: (syncPolicyId: number) => [...adminReplicationExecutionKeys.all, syncPolicyId, 'list'] as const,
  list: (syncPolicyId: number, search: ReplicationExecutionsSearch) => [
    ...adminReplicationExecutionKeys.lists(syncPolicyId),
    search,
  ] as const,
}

export function replicationExecutionsQueryOptions(
  syncPolicyId: number,
  search: ReplicationExecutionsSearch,
) {
  return queryOptions({
    queryKey: adminReplicationExecutionKeys.list(syncPolicyId, search),
    queryFn: async () => {
      const response = await SyncPolicy.ListSyncTasks({
        syncPolicyId,
        page: search.page,
        pageSize: DEFAULT_PAGE_SIZE,
        status: search.status,
      })

      return {
        executions: response.syncTasks ?? [],
        pagination: response.pagination,
      }
    },
  })
}

export function useReplicationExecutions(
  syncPolicyId: number,
  search: ReplicationExecutionsSearch,
) {
  return useQuery({
    ...replicationExecutionsQueryOptions(syncPolicyId, search),
    placeholderData: keepPreviousData,
  })
}
