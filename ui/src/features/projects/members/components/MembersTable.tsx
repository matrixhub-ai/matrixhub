import {
  Anchor,
  Avatar,
  Group,
  Text,
} from '@mantine/core'
import { MemberType } from '@matrixhub/api-ts/v1alpha1/project.pb'
import { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb'
import { useTranslation } from 'react-i18next'

import { DataTable, type DataTableProps } from '@/shared/components/DataTable'

import type { ProjectMember } from '@matrixhub/api-ts/v1alpha1/project.pb'
import type { MRT_ColumnDef } from 'mantine-react-table'
import type { ReactNode } from 'react'

export interface MembersTableProps extends Omit<DataTableProps<ProjectMember>, 'columns'> {
  currentRole?: ProjectRoleType
  currentUsername?: string
  onEditRole: (member: ProjectMember) => void
  onRemove: (member: ProjectMember) => void
  toolbarExtra?: ReactNode
}

type MemberCellProps = Parameters<NonNullable<MRT_ColumnDef<ProjectMember>['Cell']>>[0]

function MemberNameCell({ row }: MemberCellProps) {
  const name = row.original.memberName

  return (
    <Group gap="sm" wrap="nowrap">
      <Avatar size={24} radius="xl" color="gray">
        {name?.charAt(0)?.toUpperCase()}
      </Avatar>
      <Text fw={600} size="sm" truncate>
        {name || '-'}
      </Text>
    </Group>
  )
}

function useRoleLabel() {
  const { t } = useTranslation()

  return (role?: ProjectRoleType) => {
    switch (role) {
      case ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN:
        return t('projects.detail.membersPage.role.admin')
      case ProjectRoleType.ROLE_TYPE_PROJECT_EDITOR:
        return t('projects.detail.membersPage.role.editor')
      case ProjectRoleType.ROLE_TYPE_PROJECT_VIEWER:
        return t('projects.detail.membersPage.role.viewer')
      default:
        return '-'
    }
  }
}

function MemberTypeCell({ row }: MemberCellProps) {
  const { t } = useTranslation()
  const memberType = row.original.memberType

  const label = memberType === MemberType.MEMBER_TYPE_GROUP
    ? t('projects.detail.membersPage.memberType.group')
    : t('projects.detail.membersPage.memberType.user')

  return (
    <Text
      size="sm"
      c={memberType === MemberType.MEMBER_TYPE_GROUP ? 'dimmed' : undefined}
    >
      {label}
    </Text>
  )
}

function RoleTypeCell({ row }: MemberCellProps) {
  const getRoleLabel = useRoleLabel()

  return (
    <Text size="sm">
      {getRoleLabel(row.original.role)}
    </Text>
  )
}

function ActionsCell({
  row, table,
}: MemberCellProps) {
  const { t } = useTranslation()
  const meta = table.options.meta as {
    currentUsername?: string
    onEditRole?: (member: ProjectMember) => void
    onRemove?: (member: ProjectMember) => void
  } | undefined

  const isSelf = meta?.currentUsername === row.original.memberName

  return (
    <Group gap="md">
      <Anchor
        component="button"
        type="button"
        size="sm"
        c={isSelf ? 'dimmed' : undefined}
        style={isSelf ? { cursor: 'not-allowed' } : undefined}
        onClick={isSelf ? undefined : () => meta?.onEditRole?.(row.original)}
      >
        {t('projects.detail.membersPage.actions.editRole')}
      </Anchor>
      <Anchor
        component="button"
        type="button"
        size="sm"
        c={isSelf ? 'dimmed' : undefined}
        style={isSelf ? { cursor: 'not-allowed' } : undefined}
        onClick={isSelf ? undefined : () => meta?.onRemove?.(row.original)}
      >
        {t('projects.detail.membersPage.actions.remove')}
      </Anchor>
    </Group>
  )
}

export function MembersTable({
  data,
  pagination,
  page,
  loading,
  searchValue,
  onSearchChange,
  onRefresh,
  currentRole,
  currentUsername,
  onEditRole,
  onRemove,
  onBatchDelete,
  rowSelection,
  onRowSelectionChange,
  onPageChange,
  selectedCount,
  toolbarExtra,
  ...rest
}: MembersTableProps) {
  const {
    t,
  } = useTranslation()

  const isAdmin = currentRole === ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN

  const columns: MRT_ColumnDef<ProjectMember>[] = [
    {
      accessorKey: 'memberName',
      header: t('projects.detail.membersPage.table.name'),
      Cell: MemberNameCell,
    },
    {
      id: 'memberType',
      header: t('projects.detail.membersPage.table.memberType'),
      Cell: MemberTypeCell,
    },
    {
      id: 'role',
      header: t('projects.detail.membersPage.table.roleType'),
      Cell: RoleTypeCell,
    },
    ...isAdmin
      ? [{
          id: 'actions',
          header: t('projects.detail.membersPage.table.actions'),
          Cell: ActionsCell,
        }]
      : [],
  ]

  return (
    <DataTable
      {...rest}
      data={data}
      columns={columns}
      pagination={pagination}
      page={page}
      loading={loading}
      emptyTitle={t('projects.detail.membersPage.table.empty')}
      searchPlaceholder={t('projects.detail.membersPage.searchPlaceholder')}
      searchValue={searchValue}
      onSearchChange={onSearchChange}
      onRefresh={onRefresh}
      onBatchDelete={onBatchDelete}
      selectedCount={selectedCount}
      toolbarExtra={toolbarExtra}
      onPageChange={onPageChange}
      rowSelection={rowSelection}
      onRowSelectionChange={onRowSelectionChange}
      getRowId={row => `${row.memberType}:${row.memberId}`}
      tableOptions={{
        enableBatchRowSelection: true,
        enableMultiRowSelection: true,
        meta: {
          currentRole,
          currentUsername,
          onEditRole,
          onRemove,
        },
      }}
    />
  )
}
