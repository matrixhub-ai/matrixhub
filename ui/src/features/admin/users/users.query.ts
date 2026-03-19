import { Users } from '@matrixhub/api-ts/v1alpha1/user.pb'
import {
  keepPreviousData,
  queryOptions,
  useQuery,
} from '@tanstack/react-query'

export const ADMIN_USERS_PAGE_SIZE = 10

export interface AdminUsersSearch {
  page: number
  query: string
}

export const adminUsersKeys = {
  all: ['admin', 'users'] as const,
  lists: () => [...adminUsersKeys.all, 'list'] as const,
  list: (search: AdminUsersSearch) => [...adminUsersKeys.lists(), search] as const,
}

export function adminUsersQueryOptions(search: AdminUsersSearch) {
  return queryOptions({
    queryKey: adminUsersKeys.list(search),
    queryFn: () => Users.ListUsers({
      page: search.page,
      pageSize: ADMIN_USERS_PAGE_SIZE,
      search: search.query || undefined,
    }),
  })
}

export function useAdminUsers(search: AdminUsersSearch) {
  return useQuery({
    ...adminUsersQueryOptions(search),
    placeholderData: keepPreviousData,
  })
}
