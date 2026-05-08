import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'
import { Robots } from '@matrixhub/api-ts/v1alpha1/robot.pb'
import { Roles } from '@matrixhub/api-ts/v1alpha1/role.pb'
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
  details: () => [...adminRobotKeys.all, 'detail'] as const,
  detail: (robotId: number) => [...adminRobotKeys.details(), robotId] as const,
  projects: () => [...adminRobotKeys.all, 'projects'] as const,
  permissions: () => [...adminRobotKeys.all, 'permissions'] as const,
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

export function robotAccountDetailQueryOptions(robotId: number) {
  return queryOptions({
    queryKey: adminRobotKeys.detail(robotId),
    queryFn: () => Robots.GetRobotAccount({ id: robotId }),
  })
}

export function robotProjectsQueryOptions() {
  return queryOptions({
    queryKey: adminRobotKeys.projects(),
    queryFn: async () => {
      const response = await Projects.ListProjects({
        page: 1,
        pageSize: -1,
      })

      return response.projects ?? []
    },
  })
}

export function robotPermissionsQueryOptions() {
  return queryOptions({
    queryKey: adminRobotKeys.permissions(),
    queryFn: () => Roles.ListAllPermissions({}),
  })
}

export function useRobots(search: RobotsSearch) {
  return useQuery({
    ...robotsQueryOptions(search),
    placeholderData: keepPreviousData,
  })
}

export function useRobotProjects() {
  return useQuery(robotProjectsQueryOptions())
}

export function useRobotPermissions() {
  return useQuery(robotPermissionsQueryOptions())
}
