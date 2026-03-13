import {
  Anchor,
  Badge,
  Button,
  Text,
} from '@mantine/core'
import { ProjectType } from '@matrixhub/api-ts/v1alpha1/project.pb'
import { Link } from '@tanstack/react-router'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTable } from '@/shared/components/DataTable'
import { formatDateTime } from '@/shared/utils/date'

import type { Project } from '@matrixhub/api-ts/v1alpha1/project.pb'
import type { Pagination as PaginationData } from '@matrixhub/api-ts/v1alpha1/utils.pb'
import type { TFunction } from 'i18next'
import type { MRT_ColumnDef } from 'mantine-react-table'

function isPublicProject(type?: ProjectType) {
  return type === ProjectType.PROJECT_TYPE_PUBLIC
}

function getProjectColumns(
  t: TFunction,
  onDelete: (project: Project) => void,
): MRT_ColumnDef<Project>[] {
  return [
    {
      accessorKey: 'name',
      header: t('routes.projects.table.name'),
      Cell: ({ row }) => (
        <Anchor
          fw={600}
          underline="never"
          renderRoot={props => (
            <Link
              {...props}
              to="/projects/$projectId"
              params={{ projectId: row.original.name ?? '' }}
            />
          )}
        >
          {row.original.name}
        </Anchor>
      ),
    },
    {
      id: 'type',
      header: t('routes.projects.table.visibility'),
      Cell: ({ row }) => {
        const isPublic = isPublicProject(row.original.type)

        return (
          <Badge
            color={isPublic ? 'green' : 'gray'}
            variant="light"
          >
            {isPublic
              ? t('routes.projects.visibility.public')
              : t('routes.projects.visibility.private')}
          </Badge>
        )
      },
    },
    {
      id: 'enabledRegistry',
      header: t('routes.projects.table.proxy'),
      Cell: ({ row }) => (
        <Text size="sm">
          {row.original.enabledRegistry
            ? t('routes.projects.boolean.yes')
            : t('routes.projects.boolean.no')}
        </Text>
      ),
    },
    {
      accessorKey: 'modelCount',
      header: t('routes.projects.table.modelCount'),
    },
    {
      accessorKey: 'datasetCount',
      header: t('routes.projects.table.datasetCount'),
    },
    {
      id: 'updatedAt',
      header: t('routes.projects.table.updatedAt'),
      accessorFn: row => formatDateTime(row.updatedAt),
    },
    {
      id: 'actions',
      header: t('routes.projects.table.actions'),
      Cell: ({ row }) => (
        <Button
          variant="subtle"
          color="red"
          size="compact-sm"
          onClick={() => onDelete(row.original)}
        >
          {t('routes.projects.actions.delete')}
        </Button>
      ),
    },
  ]
}

interface ProjectsTableProps {
  records: Project[]
  pagination?: PaginationData
  page: number
  loading?: boolean
  searchValue?: string
  onSearchChange?: (value: string) => void
  onRefresh?: () => void
  onDelete: (project: Project) => void
  onBatchDelete?: () => void
  onPageChange: (page: number) => void
  selectedCount?: number
  toolbarExtra?: React.ReactNode
}

export function ProjectsTable({
  records,
  pagination,
  page,
  loading,
  searchValue,
  onSearchChange,
  onRefresh,
  onDelete,
  onBatchDelete,
  onPageChange,
  selectedCount,
  toolbarExtra,
}: ProjectsTableProps) {
  const { t } = useTranslation()

  const columns = useMemo(() => getProjectColumns(t, onDelete), [t, onDelete])

  return (
    <DataTable
      data={records}
      columns={columns}
      pagination={pagination}
      page={page}
      loading={loading}
      searchPlaceholder={t('routes.projects.searchPlaceholder')}
      searchValue={searchValue}
      onSearchChange={onSearchChange}
      onRefresh={onRefresh}
      onBatchDelete={onBatchDelete}
      selectedCount={selectedCount}
      toolbarExtra={toolbarExtra}
      onPageChange={onPageChange}
      enableRowSelection
      getRowId={row => row.name ?? ''}
    />
  )
}
