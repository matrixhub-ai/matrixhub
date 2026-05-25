import { SyncPolicy } from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import {
  keepPreviousData,
  queryOptions,
  useQuery,
} from '@tanstack/react-query'

import i18n from '@/i18n'

import {
  DEFAULT_REPLICATIONS_PAGE_SIZE,
  type ReplicationsSearch,
} from './replications.schema'

export const adminReplicationKeys = {
  all: ['admin', 'replications'] as const,
  lists: () => [...adminReplicationKeys.all, 'list'] as const,
  list: (search: ReplicationsSearch) => [...adminReplicationKeys.lists(), search] as const,
  details: () => [...adminReplicationKeys.all, 'detail'] as const,
  detail: (syncPolicyId: number) => [...adminReplicationKeys.details(), syncPolicyId] as const,
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

export function replicationDetailQueryOptions(syncPolicyId: number) {
  return queryOptions({
    queryKey: adminReplicationKeys.detail(syncPolicyId),
    queryFn: async () => {
      const response = await SyncPolicy.GetSyncPolicy({ syncPolicyId })

      if (!response.syncPolicy) {
        throw new Error(i18n.t('routes.admin.replications.executions.errors.syncPolicyNotFound'))
      }

      return response.syncPolicy
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
