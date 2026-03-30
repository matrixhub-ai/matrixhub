import {
  Anchor,
  Badge,
  Group,
  Text,
} from '@mantine/core'
import {
  RegistryStatus,
  RegistryType,
} from '@matrixhub/api-ts/v1alpha1/registry.pb'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTable, type DataTableProps } from '@/shared/components/DataTable'
import { formatDateTime } from '@/shared/utils/date'

import { DeleteRegistryAction } from './DeleteRegistryAction'
import { EditRegistryAction } from './EditRegistryAction'
import { getRegistryRowId } from '../registries.utils'

import type { Registry } from '@matrixhub/api-ts/v1alpha1/registry.pb'
import type { MRT_ColumnDef } from 'mantine-react-table'

type RegistryCellProps = Parameters<NonNullable<MRT_ColumnDef<Registry>['Cell']>>[0]

type RegistryActionCellProps = Parameters<NonNullable<DataTableProps<Registry>['renderRowActions']>>[0]

type RegistriesTableProps = Omit<DataTableProps<Registry>, 'columns'>

function RegistryTypeCell({ row }: RegistryCellProps) {
  const { t } = useTranslation()
  const type = row.original.type

  let label = '-'

  if (type === RegistryType.REGISTRY_TYPE_HUGGINGFACE) {
    label = t('routes.admin.registries.provider.huggingFace')
  }

  return <Text size="sm">{label}</Text>
}

function RegistryStatusCell({ row }: RegistryCellProps) {
  const { t } = useTranslation()
  const status = row.original.status

  let color = 'gray'
  let label = '-'

  if (status === RegistryStatus.REGISTRY_STATUS_HEALTHY) {
    color = 'green'
    label = t('routes.admin.registries.status.healthy')
  } else if (status === RegistryStatus.REGISTRY_STATUS_UNHEALTHY) {
    color = 'red'
    label = t('routes.admin.registries.status.unhealthy')
  }

  return (
    <Badge color={color} variant="light">
      {label}
    </Badge>
  )
}

function RegistryUrlCell({ row }: RegistryCellProps) {
  const url = row.original.url

  if (!url) {
    return <Text size="sm">-</Text>
  }

  return (
    <Anchor size="sm" href={url} target="_blank" rel="noopener noreferrer">
      {url}
    </Anchor>
  )
}

function RegistryInsecureCell({ row }: RegistryCellProps) {
  const { t } = useTranslation()
  const insecure = row.original.insecure

  if (insecure == null) {
    return <Text size="sm">-</Text>
  }

  const verified = !insecure

  return (
    <Badge color={verified ? 'green' : 'red'} variant="light">
      {verified
        ? t('routes.admin.registries.insecure.verified')
        : t('routes.admin.registries.insecure.unverified')}
    </Badge>
  )
}

function RegistryCredentialCell({ row }: RegistryCellProps) {
  const { t } = useTranslation()
  const basic = row.original.basic

  if (basic) {
    return <Text size="sm">{t('routes.admin.registries.credential.basic')}</Text>
  }

  return <Text size="sm">-</Text>
}

function RegistryActionsCell({
  row,
}: RegistryActionCellProps) {
  const isDisabled = row.original.id == null

  return (
    <Group gap={4} wrap="nowrap">
      <EditRegistryAction registry={row.original} disabled={isDisabled} />
      <DeleteRegistryAction registry={row.original} disabled={isDisabled} />
    </Group>
  )
}

export function RegistriesTable(props: RegistriesTableProps) {
  const { t } = useTranslation()

  const columns = useMemo<MRT_ColumnDef<Registry>[]>(() => [
    {
      id: 'name',
      header: t('routes.admin.registries.table.name'),
      accessorFn: row => row.name ?? '-',
    },
    {
      id: 'status',
      header: t('routes.admin.registries.table.status'),
      Cell: RegistryStatusCell,
    },
    {
      id: 'url',
      header: t('routes.admin.registries.table.url'),
      Cell: RegistryUrlCell,
    },
    {
      id: 'type',
      header: t('routes.admin.registries.table.type'),
      Cell: RegistryTypeCell,
    },
    {
      id: 'insecure',
      header: t('routes.admin.registries.table.insecure'),
      Cell: RegistryInsecureCell,
    },
    {
      id: 'credential',
      header: t('routes.admin.registries.table.credential'),
      Cell: RegistryCredentialCell,
    },
    {
      id: 'createdAt',
      header: t('routes.admin.registries.table.createdAt'),
      accessorFn: row => formatDateTime(row.createdAt),
    },
  ], [t])

  return (
    <DataTable
      {...props}
      columns={columns}
      emptyTitle={t('routes.admin.registries.table.empty')}
      searchPlaceholder={t('routes.admin.registries.searchPlaceholder')}
      getRowId={getRegistryRowId}
      enableRowActions
      renderRowActions={RegistryActionsCell}
      positionActionsColumn="last"
    />
  )
}
