import {
  Anchor,
  Box,
  Group,
  Text,
} from '@mantine/core'
import {
  SSHKeyStatus,
  type SSHKey,
} from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import { useTranslation } from 'react-i18next'

import { formatExpiredAt } from '@/features/profile/profile.utils.ts'
import { DataTable, type DataTableProps } from '@/shared/components/DataTable'
import { formatDateTime } from '@/shared/utils/date'

import type {
  MRT_ColumnDef,
  MRT_Row,
} from 'mantine-react-table'

const StatusCell = ({ row }: { row: MRT_Row<SSHKey> }) => {
  const { t } = useTranslation()
  const status = row.original.status ?? SSHKeyStatus.SSH_KEY_STATUS_UNKNOWN

  const statusColor = {
    [SSHKeyStatus.SSH_KEY_STATUS_VALID]: 'teal.6',
    [SSHKeyStatus.SSH_KEY_STATUS_EXPIRED]: 'red.6',
    [SSHKeyStatus.SSH_KEY_STATUS_UNKNOWN]: 'gray.5',
  }

  return (
    <Group gap={6}>
      <Box w={12} h={12} bdrs="50%" bg={statusColor[status]} />
      <Text size="sm">{t(`profile.sshKey.status.${status}`)}</Text>
    </Group>
  )
}

type TableCellProps = Parameters<NonNullable<MRT_ColumnDef<SSHKey>['Cell']>>[0]

function ActionCell({
  row,
  table,
}: TableCellProps) {
  const { t } = useTranslation()
  const onDelete = (
    table.options.meta as { onDelete?: (key: SSHKey) => void } | undefined
  )?.onDelete

  return (
    <Anchor
      component="button"
      size="sm"
      onClick={() => onDelete?.(row.original)}
    >
      {t('profile.sshKey.delete.action')}
    </Anchor>
  )
}

export type SshKeysTableProps = Omit<DataTableProps<SSHKey>, 'columns'> & {
  onDelete?: (sshKey: SSHKey) => void
}

export function SshKeysTable({
  data,
  onDelete,
  loading = false,
  fetching = false,
  onRefresh,
  ...rest
}: SshKeysTableProps) {
  const { t } = useTranslation()

  const columns: MRT_ColumnDef<SSHKey>[] = [
    {
      accessorKey: 'name',
      header: t('profile.sshKey.name'),
    },
    {
      accessorKey: 'status',
      header: t('profile.sshKey.statusLabel'),
      Cell: StatusCell,
    },
    {
      accessorKey: 'expireAt',
      header: t('profile.sshKey.expireAt'),
      Cell: ({ row }) => formatExpiredAt(row.original.expireAt, t),
    },
    {
      accessorKey: 'createdAt',
      header: t('profile.sshKey.importedAt'),
      Cell: ({ row }) => formatDateTime(row.original.createdAt),
    },
    {
      id: 'actions',
      enableSorting: false,
      header: '',
      Cell: ActionCell,
      size: 80,
    },
  ]

  return (
    <DataTable
      {...rest}
      columns={columns}
      data={data}
      loading={loading}
      fetching={fetching}
      onRefresh={onRefresh}
      tableOptions={{
        meta: { onDelete },
      }}
    />
  )
}
