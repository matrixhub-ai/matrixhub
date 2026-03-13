import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'
import {
  useCallback,
  useEffect,
  useReducer,
} from 'react'

import type {
  ListProjectsResponse,
  Project,
} from '@matrixhub/api-ts/v1alpha1/project.pb'
import type { Pagination } from '@matrixhub/api-ts/v1alpha1/utils.pb'

const DEFAULT_PAGE_SIZE = 10

interface UseProjectsOptions {
  name?: string
  page?: number
  pageSize?: number
}

interface UseProjectsReturn {
  projects: Project[]
  pagination?: Pagination
  loading: boolean
  refresh: () => void
}

interface State {
  data: ListProjectsResponse
  loading: boolean
  version: number
}

type Action
  = { type: 'fetch' }
    | {
      type: 'success'
      data: ListProjectsResponse
    }
    | { type: 'refresh' }

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case 'fetch':
      return {
        ...state,
        loading: true,
        data: {},
      }
    case 'success':
      return {
        ...state,
        loading: false,
        data: action.data,
      }
    case 'refresh':
      return {
        ...state,
        version: state.version + 1,
      }
  }
}

const initialState: State = {
  data: {},
  loading: true,
  version: 0,
}

export function useProjects({
  name,
  page = 1,
  pageSize = DEFAULT_PAGE_SIZE,
}: UseProjectsOptions = {}): UseProjectsReturn {
  const [state, dispatch] = useReducer(reducer, initialState)

  useEffect(() => {
    let cancelled = false

    dispatch({ type: 'fetch' })

    Projects.ListProjects({
      name: name || undefined,
      page,
      pageSize,
    }).then((res) => {
      if (!cancelled) {
        dispatch({
          type: 'success',
          data: res,
        })
      }
    })

    return () => {
      cancelled = true
    }
  }, [name, page, pageSize, state.version])

  const refresh = useCallback(() => {
    dispatch({ type: 'refresh' })
  }, [])

  return {
    projects: state.data.projects ?? [],
    pagination: state.data.pagination,
    loading: state.loading,
    refresh,
  }
}
