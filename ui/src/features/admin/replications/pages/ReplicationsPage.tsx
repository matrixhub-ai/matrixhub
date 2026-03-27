import {
  useQueryClient,
  useSuspenseQuery,
} from '@tanstack/react-query'
import {
  getRouteApi,
  useRouterState,
} from '@tanstack/react-router'

import { useRouteListState } from '@/shared/hooks/useRouteListState'

import { CreateReplicationAction } from '../components/CreateReplicationAction'
import { ReplicationsTable } from '../components/ReplicationsTable'
import {
  adminReplicationKeys,
  replicationsQueryOptions,
} from '../replications.query'
import { DEFAULT_REPLICATIONS_PAGE } from '../replications.schema'
import { getReplicationRowId } from '../replications.utils'

const replicationsRouteApi = getRouteApi('/(auth)/admin/replications')

export function ReplicationsPage() {
  const queryClient = useQueryClient()
  const navigate = replicationsRouteApi.useNavigate()
  const search = replicationsRouteApi.useSearch()
  const {
    data,
    isFetching,
  } = useSuspenseQuery(replicationsQueryOptions(search))
  const {
    replications,
    pagination,
  } = data
  const routeLoading = useRouterState({
    select: state => state.isLoading,
  })

  const refreshReplications = () => queryClient.invalidateQueries({
    queryKey: adminReplicationKeys.lists(),
  })

  const {
    onSearchChange,
    onRefresh,
    onPageChange,
  } = useRouteListState({
    search,
    navigate,
    records: replications,
    getRecordId: getReplicationRowId,
    refresh: refreshReplications,
  })

  return (
    <ReplicationsTable
      data={replications}
      pagination={pagination}
      loading={routeLoading}
      fetching={isFetching}
      page={search.page ?? DEFAULT_REPLICATIONS_PAGE}
      searchValue={search.query ?? ''}
      onSearchChange={onSearchChange}
      onRefresh={onRefresh}
      onPageChange={onPageChange}
      toolbarExtra={<CreateReplicationAction />}
    />
  )
}
