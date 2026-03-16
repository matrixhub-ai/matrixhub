import { type GetProjectRolesResponse } from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import { createContext } from 'react'

export interface ProjectRolesContextValue {
  roles: GetProjectRolesResponse['projectRoles'] | null
  loading: boolean
  refetch: () => void
}

export const ProjectRolesContext = createContext<ProjectRolesContextValue>({
  roles: null,
  loading: true,
  refetch: () => null,
})
