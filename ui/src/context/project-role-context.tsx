import { type GetProjectRolesResponse } from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb'
import { use } from 'react'

import { createNamedContext } from '@/utils/createNamedContext'

import { CurrentUserContext } from './current-user-context'

export const ProjectRolesContext = createNamedContext<GetProjectRolesResponse | undefined>('ProjectRolesContext')

// eslint-disable-next-line react-refresh/only-export-components
export function useProjectRole(projectId: string) {
  const user = use(CurrentUserContext)
  const projectRoles = use(ProjectRolesContext)

  if (user?.isAdmin) {
    return ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN
  }

  return projectRoles?.projectRoles?.[projectId]
}
