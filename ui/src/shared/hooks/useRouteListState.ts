import {
  startTransition,
  useCallback,
  useMemo,
  useState,
} from 'react'

import type { MRT_RowSelectionState } from 'mantine-react-table'

interface RouteListSearchState {
  page?: number
  query?: string
}

interface UseRouteListStateOptions<TSearch extends RouteListSearchState> {
  search: TSearch
  navigate: (options: {
    replace?: boolean
    search: (prev: TSearch) => TSearch
  }) => unknown
  refresh?: () => unknown
  defaultPage?: number
  normalizeQuery?: (value: string) => string | undefined
}

const DEFAULT_PAGE = 1

function defaultNormalizeQuery(value: string) {
  const nextQuery = value.trim()

  return nextQuery || undefined
}

export function useRouteListState<TSearch extends RouteListSearchState>({
  search,
  navigate,
  refresh,
  defaultPage = DEFAULT_PAGE,
  normalizeQuery = defaultNormalizeQuery,
}: UseRouteListStateOptions<TSearch>) {
  const [rowSelection, setRowSelection] = useState<MRT_RowSelectionState>({})

  const clearRowSelection = useCallback(() => {
    setRowSelection({})
  }, [])

  const selectedRowIds = useMemo(
    () => Object.keys(rowSelection).filter(rowId => !!rowSelection[rowId]),
    [rowSelection],
  )

  const selectedCount = selectedRowIds.length

  const handleSearchChange = useCallback((value: string) => {
    const nextQuery = normalizeQuery(value)
    const currentQuery = search.query || undefined

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
  }, [clearRowSelection, defaultPage, navigate, normalizeQuery, search.query])

  const handleRefresh = useCallback(() => {
    clearRowSelection()
    void refresh?.()
  }, [clearRowSelection, refresh])

  const handlePageChange = useCallback((page: number) => {
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
  }, [clearRowSelection, defaultPage, navigate, search.page])

  return {
    rowSelection,
    setRowSelection,
    clearRowSelection,
    selectedCount,
    selectedRowIds,
    handleSearchChange,
    handleRefresh,
    handlePageChange,
  }
}
