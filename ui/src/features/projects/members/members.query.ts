import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'
import {
  keepPreviousData,
  queryOptions,
  useQuery,
} from '@tanstack/react-query'
import { z } from 'zod'

export const MEMBERS_PAGE_SIZE = 10

export const membersSearchSchema = z.object({
  q: z.string().transform(v => v.trim()).optional().catch(''),
  page: z.coerce.number().int().positive().optional().catch(1),
})

export type MembersSearch = z.infer<typeof membersSearchSchema>

// -- Query key factory --

export const memberKeys = {
  all: ['projectMembers'] as const,
  lists: () => [...memberKeys.all, 'list'] as const,
  list: (projectId: string, params: {
    q: string
    page: number
  }) => [...memberKeys.lists(), projectId, params] as const,
}

// -- Query options factory --

export function membersQueryOptions(projectId: string, search: MembersSearch) {
  return queryOptions({
    queryKey: memberKeys.list(projectId, {
      q: search.q ?? '',
      page: search.page ?? 1,
    }),
    queryFn: () => Projects.ListProjectMembers({
      name: projectId,
      memberName: search.q || undefined,
      page: search.page,
      pageSize: MEMBERS_PAGE_SIZE,
    }),
  })
}

// -- Custom hook --

export function useMembers(projectId: string, search: MembersSearch) {
  return useQuery({
    ...membersQueryOptions(projectId, search),
    placeholderData: keepPreviousData,
  })
}
