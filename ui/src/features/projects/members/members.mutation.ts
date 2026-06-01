import {
  type AddProjectMemberWithRoleRequest,
  Projects,
  type RemoveProjectMembersRequest,
  type UpdateProjectMemberRoleRequest,
} from '@matrixhub/api-ts/v1alpha1/project.pb'
import { mutationOptions } from '@tanstack/react-query'

import i18n from '@/i18n'

import { memberKeys } from './members.query'

import type { NotificationMeta } from '@/types/tanstack-query'

export function addMemberMutationOptions() {
  return mutationOptions({
    mutationFn: (input: AddProjectMemberWithRoleRequest) =>
      Projects.AddProjectMemberWithRole(input),
    meta: {
      errorMessage: i18n.t('projects.detail.membersPage.notifications.addError'),
      invalidates: [memberKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function updateMemberRoleMutationOptions() {
  return mutationOptions({
    mutationFn: (input: UpdateProjectMemberRoleRequest) =>
      Projects.UpdateProjectMemberRole(input),
    meta: {
      errorMessage: i18n.t('projects.detail.membersPage.notifications.updateRoleError'),
      invalidates: [memberKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function removeMembersMutationOptions() {
  return mutationOptions({
    mutationFn: (input: RemoveProjectMembersRequest) =>
      Projects.RemoveProjectMembers(input),
    meta: {
      errorMessage: i18n.t('projects.detail.membersPage.notifications.removeError'),
      invalidates: [memberKeys.lists()],
    } satisfies NotificationMeta,
  })
}
