import { CurrentUser } from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import { queryOptions, useQuery } from '@tanstack/react-query'

import { queryClient } from '@/queryClient'
// -- Query key factory --
export const authKeys = {
  all: ['auth'] as const,
  currentUser: () => [...authKeys.all, 'currentUser'] as const,
  projectRoles: () => [...authKeys.all, 'projectRoles'] as const,
}

// -- Query options factory --
export function currentUserQueryOptions() {
  return queryOptions({
    queryKey: authKeys.currentUser(),
    queryFn: () => CurrentUser.GetCurrentUser({}),
  })
}

export function projectRolesQueryOptions() {
  return queryOptions({
    queryKey: authKeys.projectRoles(),
    queryFn: () => CurrentUser.GetProjectRoles({}),
  })
}

// -- Custom hooks --
export function useCurrentUser() {
  return useQuery(currentUserQueryOptions())
}

/** Call after login / logout to force fresh data on next access. */
export function invalidateAuthCache() {
  return queryClient.invalidateQueries({
    queryKey: authKeys.all,
  })
}
