import {
  type PaperProps,
  type TableProps,
} from '@mantine/core'
import { MantineReactTable } from 'mantine-react-table'

import { CollectionLayout } from './CollectionLayout'

import type { CollectionToolbarProps } from './CollectionLayout'
import type { Pagination as PaginationData } from '@matrixhub/api-ts/v1alpha1/utils.pb'
import type {
  MRT_ColumnDef,
  MRT_Row,
  MRT_RowData,
  MRT_RowSelectionState,
  MRT_TableInstance,
  MRT_TableOptions,
} from 'mantine-react-table'
import type {
  Dispatch, ReactNode, SetStateAction,
} from 'react'

interface DataTableProps<TData extends MRT_RowData> extends CollectionToolbarProps {
  /** Row data array */
  data: TData[]
  /** Column definitions */
  columns: MRT_ColumnDef<TData>[]

  // --- Pagination ---
  pagination?: PaginationData
  page: number
  onPageChange: (page: number) => void

  // --- Empty state ---
  emptyTitle?: ReactNode
  emptyDescription?: ReactNode

  // --- Row selection ---
  enableRowSelection?: boolean | ((row: MRT_Row<TData>) => boolean)
  enableSelectAll?: boolean
  rowSelection?: MRT_RowSelectionState
  onRowSelectionChange?: Dispatch<SetStateAction<MRT_RowSelectionState>>
  getRowId?: (row: TData) => string

  // --- Row actions ---
  enableRowActions?: boolean
  renderRowActions?: MRT_TableOptions<TData>['renderRowActions']
  positionActionsColumn?: 'first' | 'last'

  // --- Loading ---
  loading?: boolean

  // --- Empty rows fallback ---
  renderEmptyRowsFallback?: MRT_TableOptions<TData>['renderEmptyRowsFallback']

  // --- Display column overrides ---
  displayColumnDefOptions?: MRT_TableOptions<TData>['displayColumnDefOptions']

  // --- Escape hatch ---
  tableOptions?: Omit<
    Partial<MRT_TableOptions<TData>>,
    | 'columns'
    | 'data'
    | 'enableRowSelection'
    | 'enableSelectAll'
    | 'onRowSelectionChange'
    | 'getRowId'
    | 'enableRowActions'
    | 'renderRowActions'
    | 'positionActionsColumn'
    | 'displayColumnDefOptions'
  >
}

function mergeTableOptionProps<TData extends MRT_RowData, TProps extends object>(
  defaults: TProps,
  props:
    | TProps
    | ((args: { table: MRT_TableInstance<TData> }) => TProps)
    | undefined,
) {
  if (!props) {
    return defaults
  }

  if (typeof props === 'function') {
    return (args: { table: MRT_TableInstance<TData> }) => ({
      ...defaults,
      ...props(args),
    })
  }

  return {
    ...defaults,
    ...props,
  }
}

const emptyRowsFallback = () => null

export function DataTable<TData extends MRT_RowData>({
  data,
  columns,
  pagination,
  page,
  emptyTitle = '',
  emptyDescription = '',
  onPageChange,
  // Selection
  enableRowSelection = false,
  enableSelectAll = true,
  rowSelection,
  onRowSelectionChange,
  getRowId,
  // Row actions
  enableRowActions = false,
  renderRowActions,
  positionActionsColumn,
  // Toolbar (passed through to CollectionLayout)
  searchPlaceholder,
  searchValue,
  onSearchChange,
  onRefresh,
  selectedCount,
  onBatchDelete,
  renderToolbar,
  toolbarExtra,
  // Loading
  loading = false,
  renderEmptyRowsFallback = emptyRowsFallback,
  // Display column overrides
  displayColumnDefOptions,
  // Escape hatch
  tableOptions,
}: DataTableProps<TData>) {
  const {
    initialState,
    mantinePaperProps,
    mantineTableProps,
    state: extraState,
    ...restTableOptions
  } = tableOptions ?? {}

  return (
    <CollectionLayout
      hasItems={data.length > 0}
      pagination={pagination}
      page={page}
      loading={loading}
      emptyTitle={emptyTitle}
      emptyDescription={emptyDescription}
      onPageChange={onPageChange}
      searchPlaceholder={searchPlaceholder}
      searchValue={searchValue}
      onSearchChange={onSearchChange}
      onRefresh={onRefresh}
      selectedCount={selectedCount}
      onBatchDelete={onBatchDelete}
      renderToolbar={renderToolbar}
      toolbarExtra={toolbarExtra}
    >
      <MantineReactTable
        columns={columns}
        data={data}
        enableBottomToolbar={false}
        enableTopToolbar={false}
        enableColumnActions={false}
        enableColumnDragging={false}
        enableColumnOrdering={false}
        enableDensityToggle={false}
        enableFullScreenToggle={false}
        enableGlobalFilterModes={false}
        enableHiding={false}
        enablePagination={false}
        enableColumnFilters={false}
        enableSorting={false}
        // Selection
        enableRowSelection={enableRowSelection}
        enableSelectAll={enableSelectAll}
        layoutMode="grid"
        onRowSelectionChange={onRowSelectionChange}
        getRowId={getRowId}
        // Row actions
        enableRowActions={enableRowActions}
        renderRowActions={renderRowActions}
        positionActionsColumn={positionActionsColumn}
        renderEmptyRowsFallback={renderEmptyRowsFallback}
        localization={{ noRecordsToDisplay: '' }}
        // Display column overrides
        displayColumnDefOptions={{
          'mrt-row-select': {
            header: '',
            size: 44,
            grow: false,
            ...displayColumnDefOptions?.['mrt-row-select'],
          },
          ...displayColumnDefOptions,
        }}
        // Escape hatch
        {...restTableOptions}
        initialState={{
          density: 'xs',
          ...initialState,
        }}
        state={{
          isLoading: loading,
          showSkeletons: loading,
          rowSelection: rowSelection ?? {},
          ...extraState,
        }}
        mantinePaperProps={mergeTableOptionProps<TData, PaperProps>(
          {
            radius: 0,
            shadow: 'none',
            withBorder: false,
            style: {
              position: 'relative',
            },
          },
          mantinePaperProps,
        )}
        mantineTableProps={mergeTableOptionProps<TData, TableProps>(
          {
            highlightOnHover: true,
            style: {
              '--table-highlight-on-hover-color': 'var(--mantine-color-gray-0)',
            },
          },
          mantineTableProps,
        )}
        mantineTableHeadCellProps={{
          bg: 'var(--mantine-color-gray-0)',
          style: {
            height: 36,
            padding: '0 var(--mantine-spacing-sm)',
          },
        }}
        mantineTableBodyCellProps={{
          style: {
            height: 44,
            padding: '0 var(--mantine-spacing-sm)',
          },
        }}
        mantineTableBodyRowProps={({ row }) => ({
          bg: row.getIsSelected() ? 'var(--mantine-color-cyan-light)' : undefined,
        })}
      />
    </CollectionLayout>
  )
}
