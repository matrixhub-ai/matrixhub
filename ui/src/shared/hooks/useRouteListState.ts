import {
  startTransition,
  useState,
} from 'react'

import { DEFAULT_PAGE } from '@/utils/constants'

import type {
  MRT_ColumnFiltersState,
  MRT_RowSelectionState,
  MRT_TableOptions,
  MRT_Updater,
} from 'mantine-react-table'

interface RouteListSearchState {
  page?: number
  query?: string
}

interface RouteColumnFilterSyncConfig<TSearch extends RouteListSearchState> {
  columnId: string
  searchKey: keyof TSearch
  toColumnFilterValue?: (value: TSearch[keyof TSearch]) => unknown
  toSearchValue?: (value: unknown) => TSearch[keyof TSearch]
}

interface UseRouteListStateOptions<TSearch extends RouteListSearchState, TRecord> {
  search: TSearch
  navigate: (options: {
    replace?: boolean
    search: (prev: TSearch) => TSearch
  }) => unknown
  records: readonly TRecord[]
  getRecordId: (record: TRecord) => string
  refresh?: () => unknown
  defaultPage?: number
  normalizeQuery?: (value: string) => string | undefined
  columnFilterSync?: RouteColumnFilterSyncConfig<TSearch>[]
}

function defaultNormalizeQuery(value: string) {
  const nextQuery = value.trim()

  return nextQuery.length > 0 ? nextQuery : undefined
}

export function useRouteListState<TSearch extends RouteListSearchState, TRecord>({
  search,
  navigate,
  records,
  getRecordId,
  refresh,
  defaultPage = DEFAULT_PAGE,
  normalizeQuery = defaultNormalizeQuery,
  columnFilterSync,
}: UseRouteListStateOptions<TSearch, TRecord>) {
  const [rowSelection, setRowSelection] = useState<MRT_RowSelectionState>({})

  const currentRecordIds = new Set(records.map(record => getRecordId(record)))

  const clearRowSelection = () => {
    setRowSelection(prev => (
      Object.keys(prev).length === 0
        ? prev
        : {}
    ))
  }

  const selectedRowIds = Object.keys(rowSelection).filter(
    rowId => !!rowSelection[rowId] && currentRecordIds.has(rowId),
  )

  const selectedIdSet = new Set(selectedRowIds)
  const selectedRecords = records.filter(record => selectedIdSet.has(getRecordId(record)))

  const selectedCount = selectedRowIds.length
  const currentQuery = normalizeQuery(search.query ?? '')
  const columnFilters: MRT_ColumnFiltersState = (columnFilterSync ?? []).flatMap((filter) => {
    const rawValue = search[filter.searchKey]
    const value = filter.toColumnFilterValue
      ? filter.toColumnFilterValue(rawValue)
      : rawValue

    return value == null || value === ''
      ? []
      : [{
          id: filter.columnId,
          value,
        }]
  })

  const onSearchChange = (value: string) => {
    const nextQuery = normalizeQuery(value)

    if (nextQuery === currentQuery) {
      return
    }

    clearRowSelection()
    void navigate({
      replace: true,
      search: prev => ({
        ...prev,
        page: defaultPage,
        query: nextQuery,
      }),
    })
  }

  const onRefresh = () => {
    clearRowSelection()
    void refresh?.()
  }

  const onPageChange = (page: number) => {
    if (page === (search.page ?? defaultPage)) {
      return
    }

    clearRowSelection()
    startTransition(() => {
      void navigate({
        search: prev => ({
          ...prev,
          page,
        }),
      })
    })
  }

  const onColumnFiltersChange = (updater: MRT_Updater<MRT_ColumnFiltersState>) => {
    if (!columnFilterSync?.length) {
      return
    }

    const nextColumnFilters = typeof updater === 'function'
      ? updater(columnFilters)
      : updater

    const nextSearchPatch = {} as Partial<TSearch>
    let hasChanges = false

    for (const filter of columnFilterSync) {
      const rawValue = nextColumnFilters.find(
        columnFilter => columnFilter.id === filter.columnId,
      )?.value
      const nextValue = filter.toSearchValue
        ? filter.toSearchValue(rawValue)
        : rawValue as TSearch[keyof TSearch]

      if (nextValue !== search[filter.searchKey]) {
        hasChanges = true
      }

      nextSearchPatch[filter.searchKey] = nextValue
    }

    if (!hasChanges) {
      return
    }

    clearRowSelection()
    startTransition(() => {
      void navigate({
        replace: true,
        search: prev => ({
          ...prev,
          page: defaultPage,
          ...nextSearchPatch,
        }),
      })
    })
  }

  const routeColumnFilterTableOptions: Pick<
    MRT_TableOptions<object>,
    'manualFiltering' | 'onColumnFiltersChange' | 'state'
  > | undefined = columnFilterSync?.length
    ? {
        manualFiltering: true,
        onColumnFiltersChange,
        state: { columnFilters },
      }
    : undefined

  return {
    rowSelection,
    setRowSelection,
    clearRowSelection,
    selectedCount,
    selectedRowIds,
    selectedRecords,
    onSearchChange,
    onRefresh,
    onPageChange,
    routeColumnFilterTableOptions,
  }
}
