import {
  Anchor,
  Badge,
  Group,
  Stack,
  Text,
} from '@mantine/core'
import {
  SyncPolicyType,
  TriggerType,
} from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import { Link } from '@tanstack/react-router'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import {
  DataTable,
  type DataTableProps,
  type DataTableRowActionsProps,
} from '@/shared/components/DataTable'
import { FieldHintLabel } from '@/shared/components/FieldHintLabel'

import { DeleteReplicationAction } from './DeleteReplicationAction'
import { EditReplicationAction } from './EditReplicationAction'
import { SyncReplicationAction } from './SyncReplicationAction'
import { ToggleReplicationAction } from './ToggleReplicationAction'
import {
  formatReplicationBandwidth,
  getCronExpressionDescription,
  getReplicationRowId,
} from '../replications.utils'

import type { Registry } from '@matrixhub/api-ts/v1alpha1/registry.pb'
import type { SyncPolicyItem } from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import type { MRT_ColumnDef } from 'mantine-react-table'

type ReplicationCellProps = Parameters<NonNullable<MRT_ColumnDef<SyncPolicyItem>['Cell']>>[0]

type ReplicationsTableProps = Omit<DataTableProps<SyncPolicyItem>, 'columns'>

const EMPTY_VALUE = '-'

function ReplicationNameCell({ row }: ReplicationCellProps) {
  const name = row.original.name
  const replicationId = row.original.id

  if (replicationId == null) {
    return (
      <Text fw={500}>
        {name ?? EMPTY_VALUE}
      </Text>
    )
  }

  return (
    <Anchor
      fw={500}
      underline="never"
      renderRoot={props => (
        <Link
          {...props}
          to="/admin/replications/$replicationId/executions"
          params={{ replicationId: String(replicationId) }}
        />
      )}
    >
      {name ?? EMPTY_VALUE}
    </Anchor>
  )
}

function ReplicationStatusCell({ row }: ReplicationCellProps) {
  const { t } = useTranslation()
  const isDisabled = row.original.isDisabled

  return (
    <Badge color={isDisabled ? 'red' : 'green'} variant="light">
      {isDisabled
        ? t('routes.admin.replications.status.disabled')
        : t('routes.admin.replications.status.enabled')}
    </Badge>
  )
}

function ReplicationTriggerTypeCell({ row }: ReplicationCellProps) {
  const {
    t,
    i18n,
  } = useTranslation()
  const triggerType = row.original.triggerType

  if (triggerType === TriggerType.TRIGGER_TYPE_MANUAL) {
    return t('routes.admin.replications.trigger.manual')
  }

  if (triggerType !== TriggerType.TRIGGER_TYPE_SCHEDULED) {
    return EMPTY_VALUE
  }

  const cronExpression = row.original.triggerTypeSchedule?.cron?.trim() ?? ''
  const cronDescription = getCronExpressionDescription(
    cronExpression,
    i18n.resolvedLanguage ?? i18n.language,
  )
  const cronExpressionHint = cronDescription
    ? t('routes.admin.replications.form.cronExpressionHelp', {
        description: cronDescription,
      })
    : t('routes.admin.replications.form.cronExpressionHint')

  return (
    <FieldHintLabel
      label={t('routes.admin.replications.trigger.scheduled')}
      hint={(
        <Stack gap={4}>
          <Text size="xs">
            {t('routes.admin.replications.form.cronExpressionRaw', {
              cron: cronExpression || EMPTY_VALUE,
            })}
          </Text>
          <Text size="xs">
            {cronExpressionHint}
          </Text>
        </Stack>
      )}
      tooltipProps={{ w: 360 }}
    />
  )
}

function ReplicationActionsCell({
  row,
}: DataTableRowActionsProps<SyncPolicyItem>) {
  const isDisabled = row.original.id == null

  return (
    <Group gap={4} wrap="nowrap">
      <EditReplicationAction syncPolicy={row.original} disabled={isDisabled} />
      <SyncReplicationAction syncPolicy={row.original} disabled={isDisabled} />
      <ToggleReplicationAction syncPolicy={row.original} disabled={isDisabled} />
      <DeleteReplicationAction syncPolicy={row.original} disabled={isDisabled} />
    </Group>
  )
}

function getRegistryLabel(registry?: Registry) {
  return registry?.name || registry?.url || ''
}

function formatLocation(parts: (string | undefined)[]) {
  const normalized = parts.map(part => part?.trim() || '')

  if (normalized.every(part => !part)) {
    return EMPTY_VALUE
  }

  return normalized
    .map(part => part || EMPTY_VALUE)
    .join(' : ')
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

export function ReplicationsTable(props: ReplicationsTableProps) {
  const { t } = useTranslation()
  const localLabel = 'Local'

  const columns = useMemo<MRT_ColumnDef<SyncPolicyItem>[]>(() => [
    {
      id: 'name',
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
      accessorFn: row => row.triggerType ?? EMPTY_VALUE,
      Cell: ReplicationTriggerTypeCell,
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
      enableRowActions
      renderRowActions={ReplicationActionsCell}
    />
  )
}
