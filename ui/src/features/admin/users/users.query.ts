import { Users } from '@matrixhub/api-ts/v1alpha1/user.pb'
import {
  keepPreviousData,
  queryOptions,
  useQuery,
} from '@tanstack/react-query'

import {
  DEFAULT_USERS_PAGE_SIZE,
  type UsersSearch,
} from './users.schema'

export const adminUserKeys = {
  all: ['admin', 'users'] as const,
  lists: () => [...adminUserKeys.all, 'list'] as const,
  list: (search: UsersSearch) => [...adminUserKeys.lists(), search] as const,
}

export function usersQueryOptions(search: UsersSearch) {
  return queryOptions({
    queryKey: adminUserKeys.list(search),
    queryFn: async () => {
      const response = await Users.ListUsers({
        page: search.page,
        pageSize: DEFAULT_USERS_PAGE_SIZE,
        search: search.query,
      })

      return {
        users: response.users ?? [],
        pagination: response.pagination,
      }
    },
  })
}

// -- Custom hook --

export function useUsers(search: UsersSearch) {
  return useQuery({
    ...usersQueryOptions(search),
    placeholderData: keepPreviousData,
  })
}
