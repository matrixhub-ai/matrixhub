import {
  type AddProjectMemberWithRoleRequest,
  Projects,
  type RemoveProjectMembersRequest,
  type UpdateProjectMemberRoleRequest,
} from '@matrixhub/api-ts/v1alpha1/project.pb'
import { mutationOptions } from '@tanstack/react-query'

import { authKeys } from '@/features/auth/auth.query'

import { memberKeys } from './members.query'

export function addMemberMutationOptions() {
  return mutationOptions({
    mutationFn: (input: AddProjectMemberWithRoleRequest) =>
      Projects.AddProjectMemberWithRole(input),
    meta: {
      invalidates: [memberKeys.lists(), authKeys.projectRoles()],
    },
  })
}

export function updateMemberRoleMutationOptions() {
  return mutationOptions({
    mutationFn: (input: UpdateProjectMemberRoleRequest) =>
      Projects.UpdateProjectMemberRole(input),
    meta: {
      invalidates: [memberKeys.lists(), authKeys.projectRoles()],
    },
  })
}

export function removeMembersMutationOptions() {
  return mutationOptions({
    mutationFn: (input: RemoveProjectMembersRequest) =>
      Projects.RemoveProjectMembers(input),
    meta: {
      invalidates: [memberKeys.lists(), authKeys.projectRoles()],
    },
  })
}
