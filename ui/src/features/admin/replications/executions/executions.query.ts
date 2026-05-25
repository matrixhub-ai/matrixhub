import {
  type ListSyncTasksRequest,
  SyncPolicy,
} from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import {
  keepPreviousData,
  queryOptions,
  useQuery,
} from '@tanstack/react-query'

import { DEFAULT_PAGE_SIZE } from '@/utils/constants'

type ReplicationExecutionsListParams = Pick<ListSyncTasksRequest, 'page' | 'status'>

export const adminReplicationExecutionKeys = {
  all: ['admin', 'replications', 'executions'] as const,
  lists: (syncPolicyId: number) => [...adminReplicationExecutionKeys.all, syncPolicyId, 'list'] as const,
  list: (syncPolicyId: number, params: ReplicationExecutionsListParams) => [
    ...adminReplicationExecutionKeys.lists(syncPolicyId),
    params,
  ] as const,
}

export function replicationExecutionsQueryOptions(
  syncPolicyId: number,
  params: ReplicationExecutionsListParams,
) {
  return queryOptions({
    queryKey: adminReplicationExecutionKeys.list(syncPolicyId, params),
    queryFn: async () => {
      const response = await SyncPolicy.ListSyncTasks({
        syncPolicyId,
        page: params.page,
        pageSize: DEFAULT_PAGE_SIZE,
        status: params.status,
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
  params: ReplicationExecutionsListParams,
) {
  return useQuery({
    ...replicationExecutionsQueryOptions(syncPolicyId, params),
    placeholderData: keepPreviousData,
  })
}
