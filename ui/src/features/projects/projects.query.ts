import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'
import {
  keepPreviousData,
  queryOptions,
  useQuery,
} from '@tanstack/react-query'

import { DEFAULT_PROJECTS_PAGE_SIZE } from '@/utils/constants'

export interface ProjectsSearch {
  query: string
  page: number
}

// -- Query key factory --
export const projectKeys = {
  all: ['projects'] as const,
  lists: () => [...projectKeys.all, 'list'] as const,
  list: (params: { query: string
    page: number }) =>
    [...projectKeys.lists(), params] as const,
  detail: (projectId: string) => ['projects', projectId] as const,
}

// -- Query options factory --

// list owned projects, support pagination and name filter
export function projectsQueryOptions(search: ProjectsSearch) {
  return queryOptions({
    queryKey: projectKeys.list({
      query: search.query,
      page: search.page,
    }),
    queryFn: () => Projects.ListProjects({
      name: search.query || undefined,
      page: search.page,
      pageSize: DEFAULT_PROJECTS_PAGE_SIZE,
      managedOnly: true,
    }),
  })
}

export function projectDetailQueryOptions(projectId: string) {
  return queryOptions({
    queryKey: projectKeys.detail(projectId),
    queryFn: () => Projects.GetProject({ name: projectId }),
  })
}

// -- Custom hook --
export function useProjects(search: ProjectsSearch) {
  return useQuery({
    ...projectsQueryOptions(search),
    placeholderData: keepPreviousData,
  })
}
