import { type GetProjectRolesResponse } from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb'
import { createContext, use } from 'react'

import { CurrentUserContext } from './current-user-context'

export const ProjectRolesContext = createContext<GetProjectRolesResponse | undefined>(undefined)

export function useProjectRole(projectId: string) {
  const user = use(CurrentUserContext)
  const projectRoles = use(ProjectRolesContext)

  if (user?.isAdmin) {
    return ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN
  }

  return projectRoles?.projectRoles?.[projectId]
}
