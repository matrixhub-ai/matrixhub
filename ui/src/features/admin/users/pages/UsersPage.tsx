import {
  useQueryClient,
  useSuspenseQuery,
} from '@tanstack/react-query'
import {
  getRouteApi,
  useRouterState,
} from '@tanstack/react-router'

import { usePayloadModal } from '@/shared/hooks/usePayloadModal'
import { useRouteListState } from '@/shared/hooks/useRouteListState'

import { BatchDeleteUsersModal } from '../components/BatchDeleteUsersModal'
import { CreateUserAction } from '../components/CreateUserAction'
import { UsersTable } from '../components/UsersTable'
import {
  adminUserKeys,
  usersQueryOptions,
} from '../users.query'
import { DEFAULT_USERS_PAGE } from '../users.schema'
import { getUserRowId } from '../users.utils'

import type { User } from '@matrixhub/api-ts/v1alpha1/user.pb'

const usersRouteApi = getRouteApi('/(auth)/admin/users')

export function UsersPage() {
  const queryClient = useQueryClient()
  const navigate = usersRouteApi.useNavigate()
  const search = usersRouteApi.useSearch()
  const {
    data,
    isFetching,
  } = useSuspenseQuery(usersQueryOptions(search))
  const {
    users,
    pagination,
  } = data
  const batchDeleteModal = usePayloadModal<User[]>()
  const routeLoading = useRouterState({
    select: state => state.isLoading,
  })

  const refreshUsers = () => queryClient.invalidateQueries({
    queryKey: adminUserKeys.lists(),
  })

  const {
    rowSelection,
    setRowSelection,
    clearRowSelection,
    selectedCount,
    selectedRecords,
    onSearchChange,
    onRefresh,
    onPageChange,
  } = useRouteListState({
    search,
    navigate,
    records: users,
    getRecordId: getUserRowId,
    refresh: refreshUsers,
  })

  const handleBatchDelete = () => {
    if (selectedRecords.length === 0) {
      return
    }

    batchDeleteModal.open(selectedRecords)
  }

  return (
    <>
      <UsersTable
        data={users}
        pagination={pagination}
        loading={routeLoading}
        fetching={isFetching}
        page={search.page ?? DEFAULT_USERS_PAGE}
        searchValue={search.query ?? ''}
        onSearchChange={onSearchChange}
        onRefresh={onRefresh}
        onBatchDelete={handleBatchDelete}
        rowSelection={rowSelection}
        onRowSelectionChange={setRowSelection}
        onPageChange={onPageChange}
        selectedCount={selectedCount}
        toolbarExtra={<CreateUserAction />}
      />

      {batchDeleteModal.opened && batchDeleteModal.payload && (
        <BatchDeleteUsersModal
          opened
          users={batchDeleteModal.payload}
          onClose={batchDeleteModal.close}
          onSuccess={() => {
            clearRowSelection()
          }}
        />
      )}
    </>
  )
}
