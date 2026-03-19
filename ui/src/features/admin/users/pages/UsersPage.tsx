import { Button } from '@mantine/core'
import { useQueryClient } from '@tanstack/react-query'
import {
  getRouteApi,
  useRouterState,
} from '@tanstack/react-router'
import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'

import { useRouteListState } from '@/shared/hooks/useRouteListState'

import { UsersTable } from '../components/UsersTable'
import {
  adminUsersKeys,
  useAdminUsers,
} from '../users.query'

import type { User } from '@matrixhub/api-ts/v1alpha1/user.pb'

const usersRouteApi = getRouteApi('/(auth)/admin/users')

export function UsersPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const navigate = usersRouteApi.useNavigate()
  const search = usersRouteApi.useSearch()
  const {
    data,
    isFetching,
  } = useAdminUsers(search)
  const routerLoading = useRouterState({
    select: state => state.isLoading,
  })
  const loading = routerLoading || isFetching

  const refreshUsers = useCallback(async () => {
    await queryClient.invalidateQueries({
      queryKey: adminUsersKeys.lists(),
    })
  }, [queryClient])

  const {
    rowSelection,
    setRowSelection,
    selectedCount,
    handleSearchChange,
    handleRefresh,
    handlePageChange,
  } = useRouteListState({
    search,
    navigate,
    refresh: refreshUsers,
  })
  const users = data?.users ?? []
  const pagination = data?.pagination

  const handleCreate = () => {
    // TODO: open create user modal
  }

  const handleDelete = (_user: User) => {
    // TODO: Implement delete functionality
  }

  const handleBatchDelete = () => {
    if (selectedCount === 0) {
      return
    }

    // TODO: open batch delete user modal
  }

  return (
    <UsersTable
      records={users}
      pagination={pagination}
      loading={loading}
      page={search.page ?? 1}
      searchValue={search.query ?? ''}
      onSearchChange={handleSearchChange}
      onRefresh={handleRefresh}
      onDelete={handleDelete}
      onBatchDelete={handleBatchDelete}
      rowSelection={rowSelection}
      onRowSelectionChange={setRowSelection}
      onPageChange={handlePageChange}
      selectedCount={selectedCount}
      toolbarExtra={(
        <Button disabled onClick={handleCreate}>
          {t('routes.admin.users.toolbar.create')}
        </Button>
      )}
    />
  )
}
