import {
  type BoxProps,
  Button,
  Center,
  type PaperProps,
  Stack,
  type TableProps as MantineTableProps,
  Text,
} from '@mantine/core'
import { MantineReactTable } from 'mantine-react-table'
import { useTranslation } from 'react-i18next'

import { Pagination } from './Pagination'
import {
  SearchToolbar,
  type SearchToolbarProps,
} from './SearchToolbar'

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

// -- Toolbar types --

export interface DataTableToolbarProps {
  /** Show search input. Provide a placeholder string or `true` to use default. */
  searchPlaceholder?: string | boolean
  /** Controlled search value. */
  searchValue?: string
  /** Called when search input changes. */
  onSearchChange?: (value: string) => void
  /** Extra props forwarded to the default SearchToolbar. */
  searchToolbarProps?: Omit<
    SearchToolbarProps,
    'searchPlaceholder' | 'searchValue' | 'onSearchChange' | 'children'
  >

  /** Show refresh button when provided. */
  onRefresh?: () => void

  /** Number of selected items. Shows batch-delete button when > 0 and `onBatchDelete` is provided. */
  selectedCount?: number
  /** Called when batch-delete button is clicked. */
  onBatchDelete?: () => void

  /** Slot: replaces the entire toolbar row. Receives default toolbar as children for composition. */
  renderToolbar?: (defaultToolbar: ReactNode) => ReactNode
  /** Slot: extra actions rendered after built-in buttons. */
  toolbarExtra?: ReactNode
}

/**
 * Common props for resource table wrappers (e.g. ProjectsTable, ModelsTable).
 * Provides the standard set of pagination, search, selection, and action props
 * so each resource table doesn't have to re-declare them.
 */
export interface TableProps<T> {
  records: T[]
  pagination?: PaginationData
  page: number
  loading?: boolean
  searchValue?: string
  onSearchChange?: (value: string) => void
  searchToolbarProps?: Omit<
    SearchToolbarProps,
    'searchPlaceholder' | 'searchValue' | 'onSearchChange' | 'children'
  >
  onRefresh?: () => void
  onDelete: (item: T) => void
  onBatchDelete?: () => void
  rowSelection?: MRT_RowSelectionState
  onRowSelectionChange?: Dispatch<SetStateAction<MRT_RowSelectionState>>
  onPageChange: (page: number) => void
  selectedCount?: number
  toolbarExtra?: ReactNode
}

interface DataTableProps<TData extends MRT_RowData> extends DataTableToolbarProps {
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
  getRowId?: (row: TData, index: number) => string

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

// -- Internal helpers --

function hasContent(value: ReactNode) {
  return value !== null && value !== undefined && value !== false && value !== ''
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

// -- DataTable --

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
  // Toolbar
  searchPlaceholder,
  searchValue,
  onSearchChange,
  searchToolbarProps,
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
  const { t } = useTranslation()
  const {
    initialState,
    mantinePaperProps,
    mantineTableContainerProps,
    mantineTableProps,
    state: extraState,
    ...restTableOptions
  } = tableOptions ?? {}

  const tableState = {
    isLoading: loading,
    showSkeletons: loading,
    ...extraState,
    ...(rowSelection !== undefined ? { rowSelection } : {}),
  }

  // Toolbar
  const showBatchDelete = (selectedCount ?? 0) > 0 && !!onBatchDelete
  const showSearch = !!searchPlaceholder
  const searchPlaceholderText = typeof searchPlaceholder === 'string'
    ? searchPlaceholder
    : t('shared.search')
  const showToolbar = !!(searchPlaceholder || onRefresh || showBatchDelete || toolbarExtra)
  const defaultToolbar = showToolbar
    ? (
        <SearchToolbar
          {...searchToolbarProps}
          searchPlaceholder={showSearch ? searchPlaceholderText : undefined}
          searchValue={searchValue}
          onSearchChange={onSearchChange}
          searchInputProps={{
            maw: 360,
            style: { flex: 1 },
            w: '100%',
            mb: 'md',
            ...searchToolbarProps?.searchInputProps,
          }}

        >
          {showBatchDelete && (
            <Button
              color="red"
              variant="light"
              onClick={onBatchDelete}
            >
              {t('shared.batchDelete', { count: selectedCount })}
            </Button>
          )}
          {onRefresh && (
            <Button
              variant="default"
              onClick={onRefresh}
              loading={loading}
            >
              {t('shared.refresh')}
            </Button>
          )}
          {toolbarExtra}
        </SearchToolbar>
      )
    : null

  const toolbar = renderToolbar
    ? renderToolbar(defaultToolbar)
    : defaultToolbar

  // Empty state
  const hasEmptyTitle = hasContent(emptyTitle)
  const hasEmptyDescription = hasContent(emptyDescription)
  const showEmptyState = data.length === 0 && !loading && (hasEmptyTitle || hasEmptyDescription)

  // Pagination
  const totalPages = pagination?.pages
    ?? (
      pagination?.total && pagination?.pageSize
        ? Math.ceil(pagination.total / pagination.pageSize)
        : 0
    )

  return (
    <Stack gap={0} miw={0}>
      {toolbar}

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
        state={tableState}
        mantinePaperProps={mergeTableOptionProps<TData, PaperProps>(
          {
            radius: 0,
            shadow: 'none',
            withBorder: false,
            style: {
              position: 'relative',
              overflow: 'hidden',
            },
          },
          mantinePaperProps,
        )}
        mantineTableContainerProps={mergeTableOptionProps<TData, BoxProps>(
          {
            style: {
              maxWidth: '100%',
              overflowX: 'auto',
            },
          },
          mantineTableContainerProps,
        )}
        mantineTableProps={mergeTableOptionProps<TData, MantineTableProps>(
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

      {showEmptyState && (
        <Center py="xl">
          <Stack align="center" gap="xs">
            {hasEmptyTitle && <Text fw={500}>{emptyTitle}</Text>}
            {hasEmptyDescription && (
              <Text size="sm" c="dimmed">
                {emptyDescription}
              </Text>
            )}
          </Stack>
        </Center>
      )}

      <Pagination
        total={pagination?.total ?? 0}
        totalPages={totalPages}
        page={page}
        onPageChange={onPageChange}
      />
    </Stack>
  )
}
