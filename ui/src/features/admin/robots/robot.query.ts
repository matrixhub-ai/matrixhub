import { Robots } from '@matrixhub/api-ts/v1alpha1/robot.pb'
import {
  keepPreviousData,
  queryOptions,
  useQuery,
} from '@tanstack/react-query'

import type { RobotsSearch } from '@/routes/(auth)/admin/robots'

export const adminRobotKeys = {
  all: ['admin', 'robots'] as const,
  lists: () => [...adminRobotKeys.all, 'list'] as const,
  list: (search: RobotsSearch) => [...adminRobotKeys.lists(), search] as const,
}

export function robotsQueryOptions(search: RobotsSearch) {
  return queryOptions({
    queryKey: adminRobotKeys.list(search),
    queryFn: () => Robots.ListRobotAccounts({
      search: search.query || undefined,
      page: search.page,
      pageSize: search.pageSize,
    }),
  })
}

export function useRobots(search: RobotsSearch) {
  return useQuery({
    ...robotsQueryOptions(search),
    placeholderData: keepPreviousData,
  })
}
