import {
  Badge,
  Button,
  Group,
  Text,
} from '@mantine/core'
import { UserSource } from '@matrixhub/api-ts/v1alpha1/user.pb'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTable, type TableProps } from '@/shared/components/DataTable'
import { formatDateTime } from '@/shared/utils/date'

import type { User } from '@matrixhub/api-ts/v1alpha1/user.pb'
import type { MRT_ColumnDef } from 'mantine-react-table'

type UserCellProps = Parameters<NonNullable<MRT_ColumnDef<User>['Cell']>>[0]

interface UsersTableMeta {
  onDelete?: (user: User) => void
}

type UsersTableProps = TableProps<User>

function getUserRowId(user: User) {
  return String(user.id ?? user.username ?? user.email ?? '')
}

function UserAdminCell({ row }: UserCellProps) {
  const { t } = useTranslation()
  const isAdmin = !!row.original.isAdmin

  return (
    <Badge
      color={isAdmin ? 'green' : 'red'}
      variant="light"
    >
      {isAdmin
        ? t('routes.admin.users.boolean.yes')
        : t('routes.admin.users.boolean.no')}
    </Badge>
  )
}

function UserSourceCell({ row }: UserCellProps) {
  const { t } = useTranslation()

  if (row.original.source === UserSource.USER_SOURCE_LOCAL) {
    return <Text size="sm">{t('routes.admin.users.source.local')}</Text>
  }

  return <Text size="sm">-</Text>
}

function UserNameCell({ row }: UserCellProps) {
  return (
    <Text fw={500}>
      {row.original.username ?? '-'}
    </Text>
  )
}

function UserActionsCell({
  row,
  table,
}: UserCellProps) {
  const { t } = useTranslation()
  const adminAction = row.original.isAdmin
    ? t('routes.admin.users.actions.revokeAdmin')
    : t('routes.admin.users.actions.setAdmin')

  const onDelete = (table.options.meta as UsersTableMeta | undefined)?.onDelete

  return (
    <Group gap={4} wrap="nowrap">
      <Button
        variant="subtle"
        size="compact-sm"
        disabled
      >
        {adminAction}
      </Button>
      <Button
        variant="subtle"
        size="compact-sm"
        disabled
      >
        {t('routes.admin.users.actions.resetPassword')}
      </Button>
      <Button
        variant="subtle"
        size="compact-sm"
        disabled
        onClick={() => onDelete?.(row.original)}
      >
        {t('routes.admin.users.actions.delete')}
      </Button>
    </Group>
  )
}

export function UsersTable({
  records,
  pagination,
  page,
  loading,
  searchValue,
  onSearchChange,
  onRefresh,
  onDelete,
  onBatchDelete,
  rowSelection,
  onRowSelectionChange,
  onPageChange,
  selectedCount,
  toolbarExtra,
}: UsersTableProps) {
  const { t } = useTranslation()

  const columns = useMemo<MRT_ColumnDef<User>[]>(() => [
    {
      accessorKey: 'username',
      header: t('routes.admin.users.table.username'),
      size: 180,
      grow: false,
      Cell: UserNameCell,
    },
    {
      id: 'isAdmin',
      header: t('routes.admin.users.table.admin'),
      size: 180,
      grow: false,
      Cell: UserAdminCell,
    },
    {
      id: 'source',
      header: t('routes.admin.users.table.source'),
      size: 180,
      grow: false,
      Cell: UserSourceCell,
    },
    {
      id: 'createdAt',
      header: t('routes.admin.users.table.createdAt'),
      size: 180,
      grow: false,
      accessorFn: row => formatDateTime(row.createdAt),
    },
    {
      id: 'actions',
      header: t('routes.admin.users.table.actions'),
      size: 260,
      Cell: UserActionsCell,
    },
  ], [t])

  return (
    <DataTable
      data={records}
      columns={columns}
      pagination={pagination}
      page={page}
      loading={loading}
      emptyTitle={t('routes.admin.users.table.empty')}
      searchPlaceholder={t('routes.admin.users.searchPlaceholder')}
      searchValue={searchValue}
      onSearchChange={onSearchChange}
      onRefresh={onRefresh}
      onBatchDelete={onBatchDelete}
      selectedCount={selectedCount}
      onPageChange={onPageChange}
      toolbarExtra={toolbarExtra}
      enableRowSelection
      rowSelection={rowSelection}
      onRowSelectionChange={onRowSelectionChange}
      getRowId={getUserRowId}
      tableOptions={{
        enableBatchRowSelection: true,
        enableMultiRowSelection: true,
        meta: { onDelete },
      }}
    />
  )
}
