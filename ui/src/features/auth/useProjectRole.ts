import { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb'
import { useQuery } from '@tanstack/react-query'
import { use } from 'react'

import { CurrentUserContext } from '@/context/current-user-context'

import { projectRolesQueryOptions } from './auth.query'

export function useProjectRole(projectId: string) {
  const user = use(CurrentUserContext)
  const { data: projectRoles } = useQuery(projectRolesQueryOptions())

  if (user?.isAdmin) {
    return ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN
  }

  return projectRoles?.projectRoles?.[projectId]
}
