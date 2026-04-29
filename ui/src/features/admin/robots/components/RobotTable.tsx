import {
  Badge,
  Button,
  Group,
  Text,
} from '@mantine/core'
import {
  RobotAccountExpireStatus,
  RobotAccountProjectScope,
} from '@matrixhub/api-ts/v1alpha1/robot.pb'
import { useTranslation } from 'react-i18next'

import { DataTable, type DataTableProps } from '@/shared/components/DataTable'
import { formatDateTime } from '@/shared/utils/date'

import {
  getRobotRowId,
  isRobotEnabled,
  type RobotAccount,
} from '../robot.utils'

import type { MRT_ColumnDef } from 'mantine-react-table'

type RobotCellProps = Parameters<NonNullable<MRT_ColumnDef<RobotAccount>['Cell']>>[0]

type RobotActionCellProps = Parameters<NonNullable<DataTableProps<RobotAccount>['renderRowActions']>>[0]

type RobotTableProps = Omit<DataTableProps<RobotAccount>, 'columns'> & {
  onEdit?: (robot: RobotAccount) => void
  onDelete?: (robot: RobotAccount) => void
  onToggleStatus?: (robot: RobotAccount) => void
  onRefreshToken?: (robot: RobotAccount) => void
}

function RobotStatusCell({ row }: RobotCellProps) {
  const { t } = useTranslation()
  const enabled = isRobotEnabled(row.original)

  return (
    <Badge
      color={enabled ? 'green' : 'gray'}
      variant="light"
    >
      {enabled
        ? t('routes.admin.robots.boolean.yes')
        : t('routes.admin.robots.boolean.no')}
    </Badge>
  )
}

function RobotPlatformPermissionsCell({ row }: RobotCellProps) {
  const { t } = useTranslation()
  const count = row.original.platformPermissions?.length ?? 0

  return (
    <Text size="sm">
      {count > 0
        ? t('routes.admin.robots.permissionCount', { count })
        : t('routes.admin.robots.noPermissions')}
    </Text>
  )
}

function RobotProjectCoverageCell({ row }: RobotCellProps) {
  const { t } = useTranslation()
  const {
    projects, projectScope, projectPermissions,
  } = row.original
  const projectCount = projects?.length ?? 0
  const permissionCount = projectPermissions?.length ?? 0
  const hasAllProjects = projectScope === RobotAccountProjectScope.ROBOT_ACCOUNT_PROJECT_SCOPE_ALL

  if (!hasAllProjects && projectCount === 0 && permissionCount === 0) {
    return <Text size="sm">{t('routes.admin.robots.noPermissions')}</Text>
  }

  const projectLabel = hasAllProjects
    ? t('routes.admin.robots.projectScope.all')
    : t('routes.admin.robots.projectScope.selectedCount', { count: projectCount })
  const permissionLabel = permissionCount > 0
    ? t('routes.admin.robots.permissionCount', { count: permissionCount })
    : t('routes.admin.robots.noPermissions')

  return (
    <Group gap={8} wrap="nowrap">
      <Text size="sm">{projectLabel}</Text>
      <Text size="sm">{permissionLabel}</Text>
    </Group>
  )
}

function RobotExpireCell({ row }: RobotCellProps) {
  const { t } = useTranslation()
  const {
    expireStatus, remainPeriod,
  } = row.original

  if (expireStatus === RobotAccountExpireStatus.ROBOT_ACCOUNT_EXPIRE_STATUS_NEVER) {
    return <Text size="sm">{t('routes.admin.robots.expireStatus.never')}</Text>
  }

  if (expireStatus === RobotAccountExpireStatus.ROBOT_ACCOUNT_EXPIRE_STATUS_EXPIRED) {
    return <Text size="sm">{t('routes.admin.robots.expireStatus.expired')}</Text>
  }

  if (expireStatus === RobotAccountExpireStatus.ROBOT_ACCOUNT_EXPIRE_STATUS_VALID && remainPeriod) {
    return <Text size="sm">{remainPeriod}</Text>
  }

  return <Text size="sm">-</Text>
}

function RobotDescriptionCell({ row }: RobotCellProps) {
  return <Text size="sm">{row.original.description || '-'}</Text>
}

export function RobotTable({
  onEdit,
  onDelete,
  onToggleStatus,
  onRefreshToken,
  tableOptions,
  ...props
}: RobotTableProps) {
  const { t } = useTranslation()
  const {
    initialState,
    ...restTableOptions
  } = tableOptions ?? {}

  const columns: MRT_ColumnDef<RobotAccount>[] = [
    {
      id: 'name',
      header: t('routes.admin.robots.table.name'),
      accessorFn: row => row.name ?? '-',
    },
    {
      id: 'status',
      header: t('routes.admin.robots.table.status'),
      Cell: RobotStatusCell,
    },
    {
      id: 'platformPermissions',
      header: t('routes.admin.robots.table.platformPermissions'),
      Cell: RobotPlatformPermissionsCell,
    },
    {
      id: 'projectCoverage',
      header: t('routes.admin.robots.table.projectCoverage'),
      Cell: RobotProjectCoverageCell,
    },
    {
      id: 'createdAt',
      header: t('routes.admin.robots.table.createdAt'),
      accessorFn: row => formatDateTime(row.createdAt, 'YYYY-MM-DD HH:mm'),
    },
    {
      id: 'expireStatus',
      header: t('routes.admin.robots.table.expireStatus'),
      Cell: RobotExpireCell,
    },
    {
      id: 'description',
      header: t('routes.admin.robots.table.description'),
      Cell: RobotDescriptionCell,
    },
  ]

  const renderRobotActions = ({ row }: RobotActionCellProps) => {
    const robot = row.original
    const actionDisabled = robot.id == null
    const enabled = isRobotEnabled(robot)

    return (
      <Group gap={4} wrap="nowrap">
        <Button
          variant="transparent"
          size="compact-sm"
          color="blue"
          disabled={actionDisabled}
          onClick={() => onRefreshToken?.(robot)}
        >
          {t('routes.admin.robots.actions.refreshToken')}
        </Button>
        <Button
          variant="transparent"
          size="compact-sm"
          color="blue"
          disabled={actionDisabled}
          onClick={() => onEdit?.(robot)}
        >
          {t('routes.admin.robots.actions.edit')}
        </Button>
        <Button
          variant="transparent"
          size="compact-sm"
          color="blue"
          disabled={actionDisabled}
          onClick={() => onToggleStatus?.(robot)}
        >
          {enabled
            ? t('routes.admin.robots.actions.disable')
            : t('routes.admin.robots.actions.enable')}
        </Button>
        <Button
          variant="transparent"
          size="compact-sm"
          color="red"
          disabled={actionDisabled}
          onClick={() => onDelete?.(robot)}
        >
          {t('routes.admin.robots.actions.delete')}
        </Button>
      </Group>
    )
  }

  return (
    // FIXME: Should all lists use the component's built-in actions column with fixed enabled?
    <DataTable
      {...props}
      columns={columns}
      emptyTitle={t('routes.admin.robots.table.empty')}
      searchPlaceholder={t('routes.admin.robots.searchPlaceholder')}
      getRowId={getRobotRowId}
      enableRowActions
      renderRowActions={renderRobotActions}
      positionActionsColumn="last"
      tableOptions={{
        ...restTableOptions,
        enableColumnPinning: true,
        initialState: {
          ...initialState,
          columnPinning: {
            ...initialState?.columnPinning,
            right: [
              ...(initialState?.columnPinning?.right ?? []),
              'mrt-row-actions',
            ],
          },
        },
      }}
    />
  )
}
