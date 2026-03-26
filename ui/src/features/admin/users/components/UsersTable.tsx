import {
  Badge,
  Group,
  Text,
} from '@mantine/core'
import { UserSource } from '@matrixhub/api-ts/v1alpha1/user.pb'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTable, type DataTableProps } from '@/shared/components/DataTable'
import { formatDateTime } from '@/shared/utils/date'

import { DeleteUserAction } from './DeleteUserAction'
import { ResetUserPasswordAction } from './ResetUserPasswordAction'
import { ToggleUserAdminAction } from './ToggleUserAdminAction'
import { getUserRowId } from '../users.utils'

import type { User } from '@matrixhub/api-ts/v1alpha1/user.pb'
import type { MRT_ColumnDef } from 'mantine-react-table'

type UserCellProps = Parameters<NonNullable<MRT_ColumnDef<User>['Cell']>>[0]

type UserRowActionsProps = Parameters<NonNullable<DataTableProps<User>['renderRowActions']>>[0]

type UsersTableProps = Omit<DataTableProps<User>, 'columns'>

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

function UserActionsCell({
  row,
}: UserRowActionsProps) {
  const actionDisabled = row.original.id == null

  return (
    <Group gap={4} wrap="nowrap">
      <ToggleUserAdminAction
        user={row.original}
        disabled={actionDisabled}
      />
      <ResetUserPasswordAction
        user={row.original}
        disabled={actionDisabled}
      />
      <DeleteUserAction
        user={row.original}
        disabled={actionDisabled}
      />
    </Group>
  )
}

export function UsersTable({
  tableOptions,
  ...props
}: UsersTableProps) {
  const { t } = useTranslation()

  const columns = useMemo<MRT_ColumnDef<User>[]>(() => [
    {
      accessorKey: 'username',
      header: t('routes.admin.users.table.username'),
      size: 180,
      accessorFn: row => row.username ?? '-',
    },
    {
      id: 'isAdmin',
      header: t('routes.admin.users.table.admin'),
      size: 180,
      Cell: UserAdminCell,
    },
    {
      id: 'source',
      header: t('routes.admin.users.table.source'),
      size: 180,
      Cell: UserSourceCell,
    },
    {
      id: 'createdAt',
      header: t('routes.admin.users.table.createdAt'),
      size: 180,
      accessorFn: row => formatDateTime(row.createdAt),
    },

  ], [t])

  return (
    <DataTable
      {...props}
      columns={columns}
      emptyTitle={t('routes.admin.users.table.empty')}
      searchPlaceholder={t('routes.admin.users.searchPlaceholder')}
      enableRowSelection
      getRowId={getUserRowId}
      enableRowActions
      renderRowActions={UserActionsCell}
      positionActionsColumn="last"
      tableOptions={{
        ...tableOptions,
        enableBatchRowSelection: true,
        enableMultiRowSelection: true,
      }}
    />
  )
}
