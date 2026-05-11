import {
  ActionIcon,
  Tooltip,
} from '@mantine/core'
import {
  ResourceType,
  SyncJobStatus,
  type SyncJob,
} from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import { IconFileText } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import {
  DataTable,
  type DataTableProps,
  type DataTableRowActionsProps,
} from '@/shared/components/DataTable'
import { StatusIndicator } from '@/shared/components/StatusIndicator'
import { formatDateTime } from '@/shared/utils/date'

import {
  syncJobStatusFilterValues,
  syncJobResourceTypeFilterValues,
} from '../execution-detail.schema'

import type { MRT_ColumnDef } from 'mantine-react-table'

type SyncJobTableProps = Omit<
  DataTableProps<SyncJob>,
  'columns' | 'renderToolbar'
> & {
  syncPolicyId: number
  syncTaskId: number
}

type JobCellProps = Parameters<NonNullable<MRT_ColumnDef<SyncJob>['Cell']>>[0]

function getSyncJobLogUrl(syncPolicyId: number, syncTaskId: number, syncJobId: number) {
  return `/api/v1alpha1/sync-policies/${syncPolicyId}/sync-tasks/${syncTaskId}/sync-jobs/${syncJobId}/log`
}

function JobStatusCell({ row }: JobCellProps) {
  const { t } = useTranslation()
  const status = row.original.status

  if (status === SyncJobStatus.SYNC_JOB_STATUS_RUNNING) {
    return (
      <StatusIndicator
        type="info"
        label={t('routes.admin.replications.executions.detail.status.running')}
      />
    )
  }

  if (status === SyncJobStatus.SYNC_JOB_STATUS_STOPPED) {
    return (
      <StatusIndicator
        type="warning"
        label={t('routes.admin.replications.executions.detail.status.stopped')}
      />
    )
  }

  if (status === SyncJobStatus.SYNC_JOB_STATUS_SUCCEEDED) {
    return (
      <StatusIndicator
        type="success"
        label={t('routes.admin.replications.executions.detail.status.succeeded')}
      />
    )
  }

  if (status === SyncJobStatus.SYNC_JOB_STATUS_FAILED) {
    return (
      <StatusIndicator
        type="danger"
        label={t('routes.admin.replications.executions.detail.status.failed')}
      />
    )
  }

  return (
    <StatusIndicator
      label={t('routes.admin.replications.executions.detail.status.unknown')}
    />
  )
}

function JobActionsCell({
  row,
  syncPolicyId,
  syncTaskId,
  logsLabel,
}: Pick<DataTableRowActionsProps<SyncJob>, 'row'> & {
  syncPolicyId: number
  syncTaskId: number
  logsLabel: string
}) {
  const syncJobId = row.original.id

  if (syncPolicyId == null || syncTaskId == null || syncJobId == null) {
    return null
  }

  return (
    <Tooltip label={logsLabel}>
      <ActionIcon
        variant="subtle"
        component="a"
        href={getSyncJobLogUrl(syncPolicyId, syncTaskId, syncJobId)}
        target="_blank"
        rel="noopener noreferrer"
        aria-label={logsLabel}
      >
        <IconFileText size={18} />
      </ActionIcon>
    </Tooltip>
  )
}

export function SyncJobTable({
  syncPolicyId,
  syncTaskId,
  onRefresh,
  fetching = false,
  tableOptions,
  ...props
}: SyncJobTableProps) {
  const { t } = useTranslation()
  const logsLabel = t('routes.admin.replications.executions.detail.table.logs')

  const columns: MRT_ColumnDef<SyncJob>[] = [
    {
      accessorKey: 'id',
      header: t('routes.admin.replications.executions.detail.table.id'),
    },
    {
      accessorKey: 'resourceType',
      header: t('routes.admin.replications.executions.detail.table.resourceType'),
      accessorFn: (row) => {
        if (row.resourceType === ResourceType.RESOURCE_TYPE_MODEL) {
          return t('routes.admin.replications.executions.detail.resourceType.model')
        }
        if (row.resourceType === ResourceType.RESOURCE_TYPE_DATASET) {
          return t('routes.admin.replications.executions.detail.resourceType.dataset')
        }

        return row.resourceType
      },
      enableColumnFilter: true,
      filterVariant: 'select',
      mantineFilterSelectProps: {
        data: syncJobResourceTypeFilterValues.map(value => ({
          value,
          label: t(`routes.admin.replications.executions.detail.resourceType.${value.replace('RESOURCE_TYPE_', '').toLowerCase()}`),
        })),
        placeholder: t('routes.admin.replications.executions.detail.filters.resourceType'),
      },
    },
    {
      accessorKey: 'resourceName',
      header: t('routes.admin.replications.executions.detail.table.source'),
      minSize: 200,
    },
    {
      accessorKey: 'targetResourceName',
      header: t('routes.admin.replications.executions.detail.table.target'),
      minSize: 200,
    },
    {
      accessorKey: 'action',
      header: t('routes.admin.replications.executions.detail.table.action'),
    },
    {
      accessorKey: 'status',
      header: t('routes.admin.replications.executions.detail.table.status'),
      Cell: JobStatusCell,
      enableColumnFilter: true,
      filterVariant: 'select',
      mantineFilterSelectProps: {
        data: syncJobStatusFilterValues.map(value => ({
          value,
          label: t(`routes.admin.replications.executions.detail.status.${value.replace('SYNC_JOB_STATUS_', '').toLowerCase()}`),
        })),
        placeholder: t('routes.admin.replications.executions.detail.filters.status'),
      },
    },
    {
      accessorKey: 'createdTimestamp',
      header: t('routes.admin.replications.executions.detail.table.createdAt'),
      accessorFn: row => formatDateTime(row.createdTimestamp, 'YYYY/M/D HH:mm'),
    },
    {
      accessorKey: 'completedTimestamp',
      header: t('routes.admin.replications.executions.detail.table.completedAt'),
      accessorFn: row => formatDateTime(row.completedTimestamp, 'YYYY/M/D HH:mm'),
    },
  ]

  return (
    <DataTable
      {...props}
      columns={columns}
      emptyTitle={t('routes.admin.replications.executions.table.empty')}
      fetching={fetching}
      onRefresh={onRefresh}
      enableRowActions
      renderRowActions={({ row }) => (
        <JobActionsCell
          row={row}
          syncPolicyId={syncPolicyId}
          syncTaskId={syncTaskId}
          logsLabel={logsLabel}
        />
      )}
      displayColumnDefOptions={{
        'mrt-row-actions': {
          header: logsLabel,
        },
      }}
      tableOptions={tableOptions}
    />
  )
}
