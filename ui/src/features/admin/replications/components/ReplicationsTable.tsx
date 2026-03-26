import {
  Badge,
  Button,
  Group,
  Text,
} from '@mantine/core'
import { SyncPolicyType } from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTable, type DataTableProps } from '@/shared/components/DataTable'

import {
  formatReplicationBandwidth,
  getReplicationRowId,
} from '../replications.utils'

import type { Registry } from '@matrixhub/api-ts/v1alpha1/registry.pb'
import type { SyncPolicyItem } from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import type { MRT_ColumnDef } from 'mantine-react-table'

type ReplicationCellProps = Parameters<NonNullable<MRT_ColumnDef<SyncPolicyItem>['Cell']>>[0]

type ReplicationActionCellProps = Parameters<NonNullable<DataTableProps<SyncPolicyItem>['renderRowActions']>>[0]

type ReplicationsTableProps = Omit<DataTableProps<SyncPolicyItem>, 'columns'>

const EMPTY_VALUE = '-'

function ReplicationNameCell({ row }: ReplicationCellProps) {
  return (
    <Text fw={500}>
      {row.original.name ?? EMPTY_VALUE}
    </Text>
  )
}

function ReplicationStatusCell({ row }: ReplicationCellProps) {
  const { t } = useTranslation()
  const isDisabled = row.original.isDisabled

  return (
    <Badge color={isDisabled ? 'gray' : 'green'} variant="light">
      {isDisabled
        ? t('routes.admin.replications.status.disabled')
        : t('routes.admin.replications.status.enabled')}
    </Badge>
  )
}

function ReplicationActionsCell({
  row,
}: ReplicationActionCellProps) {
  const { t } = useTranslation()
  const toggleLabel = row.original.isDisabled
    ? t('routes.admin.replications.actions.enable')
    : t('routes.admin.replications.actions.disable')
  const isDisabled = row.original.id == null

  return (
    <Group gap={4} wrap="nowrap">
      <Button
        variant="transparent"
        size="compact-sm"
        color="blue"
        disabled={isDisabled}
      >
        {t('routes.admin.replications.actions.edit')}
      </Button>
      <Button
        variant="transparent"
        size="compact-sm"
        color="blue"
        disabled={isDisabled}
      >
        {t('routes.admin.replications.actions.sync')}
      </Button>
      <Button
        variant="transparent"
        size="compact-sm"
        color="blue"
        disabled={isDisabled}
      >
        {toggleLabel}
      </Button>
      <Button
        variant="transparent"
        size="compact-sm"
        color="blue"
        disabled={isDisabled}
      >
        {t('routes.admin.replications.actions.delete')}
      </Button>
    </Group>
  )
}

function getRegistryLabel(registry?: Registry) {
  return registry?.name || registry?.url || ''
}

function formatLocation(parts: (string | undefined)[]) {
  const location = parts
    .map(part => part?.trim())
    .filter((part): part is string => Boolean(part))
    .join(' : ')

  return location || EMPTY_VALUE
}

function getReplicationSource(item: SyncPolicyItem, localLabel: string) {
  if (item.policyType === SyncPolicyType.SYNC_POLICY_TYPE_PULL_BASE) {
    return getRegistryLabel(item.pullBasePolicy?.sourceRegistry) || EMPTY_VALUE
  }

  if (item.policyType === SyncPolicyType.SYNC_POLICY_TYPE_PUSH_BASE) {
    return localLabel
  }

  return EMPTY_VALUE
}

function getReplicationTarget(item: SyncPolicyItem, localLabel: string) {
  if (item.policyType === SyncPolicyType.SYNC_POLICY_TYPE_PULL_BASE) {
    return formatLocation([
      localLabel,
      item.pullBasePolicy?.targetProjectName,
    ])
  }

  if (item.policyType === SyncPolicyType.SYNC_POLICY_TYPE_PUSH_BASE) {
    return formatLocation([
      getRegistryLabel(item.pushBasePolicy?.targetRegistry),
      item.pushBasePolicy?.targetProjectName,
    ])
  }

  return EMPTY_VALUE
}

export function ReplicationsTable({
  tableOptions,
  ...props
}: ReplicationsTableProps) {
  const { t } = useTranslation()
  const localLabel = 'Local'

  const columns = useMemo<MRT_ColumnDef<SyncPolicyItem>[]>(() => [
    {
      accessorKey: 'name',
      header: t('routes.admin.replications.table.name'),
      Cell: ReplicationNameCell,
    },
    {
      id: 'status',
      header: t('routes.admin.replications.table.status'),
      Cell: ReplicationStatusCell,
    },
    {
      id: 'source',
      header: t('routes.admin.replications.table.source'),
      accessorFn: row => getReplicationSource(row, localLabel),
    },
    {
      id: 'syncMode',
      header: t('routes.admin.replications.table.syncMode'),
      accessorFn: (row) => {
        if (row.policyType === SyncPolicyType.SYNC_POLICY_TYPE_PULL_BASE) {
          return 'pull'
        }

        if (row.policyType === SyncPolicyType.SYNC_POLICY_TYPE_PUSH_BASE) {
          return 'push'
        }

        return '-'
      },
    },
    {
      id: 'target',
      header: t('routes.admin.replications.table.target'),
      accessorFn: row => getReplicationTarget(row, localLabel),
    },
    {
      id: 'triggerType',
      header: t('routes.admin.replications.table.triggerType'),
      accessorFn: (row) => {
        if (row.triggerType === 'TRIGGER_TYPE_MANUAL') {
          return t('routes.admin.replications.trigger.manual')
        }

        if (row.triggerType === 'TRIGGER_TYPE_SCHEDULED') {
          return t('routes.admin.replications.trigger.scheduled')
        }

        return EMPTY_VALUE
      },
    },
    {
      id: 'bandwidth',
      header: t('routes.admin.replications.table.bandwidth'),
      accessorFn: (row) => {
        const formatted = formatReplicationBandwidth(row.bandwidth)

        return formatted || t('routes.admin.replications.bandwidth.unlimited')
      },
    },
  ], [localLabel, t])

  return (
    <DataTable
      {...props}
      columns={columns}
      emptyTitle={t('routes.admin.replications.table.empty')}
      searchPlaceholder={t('routes.admin.replications.searchPlaceholder')}
      getRowId={getReplicationRowId}
      enableRowSelection
      enableRowActions
      renderRowActions={ReplicationActionsCell}
      positionActionsColumn="last"
      tableOptions={{
        ...tableOptions,
        enableBatchRowSelection: true,
        enableMultiRowSelection: true,
      }}
    />
  )
}
