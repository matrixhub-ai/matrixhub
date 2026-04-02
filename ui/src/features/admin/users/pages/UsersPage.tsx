import { useQueryClient } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'

import { usePayloadModal } from '@/shared/hooks/usePayloadModal'
import { useRouteListState } from '@/shared/hooks/useRouteListState'

import { BatchDeleteUsersModal } from '../components/BatchDeleteUsersModal'
import { CreateUserAction } from '../components/CreateUserAction'
import { UsersTable } from '../components/UsersTable'
import {
  adminUserKeys,
  useUsers,
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
    isLoading,
    isFetching,
  } = useUsers(search)
  const users = data?.users ?? []
  const pagination = data?.pagination
  const batchDeleteModal = usePayloadModal<User[]>()

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
        loading={isLoading}
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
