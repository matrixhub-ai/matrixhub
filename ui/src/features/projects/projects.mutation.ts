import {
  ProjectType, Projects, type UpdateProjectRequest,
} from '@matrixhub/api-ts/v1alpha1/project.pb'
import { mutationOptions } from '@tanstack/react-query'

import { authKeys } from '@/features/auth/auth.query'
import i18n from '@/i18n'

import { projectKeys } from './projects.query'

import type { CreateProjectInput } from './projects.schema'
import type { NotificationMeta } from '@/types/tanstack-query'

export function createProjectMutationOptions() {
  return mutationOptions({
    mutationFn: (input: CreateProjectInput) =>
      Projects.CreateProject({
        name: input.name,
        type: input.isPublic
          ? ProjectType.PROJECT_TYPE_PUBLIC
          : ProjectType.PROJECT_TYPE_PRIVATE,
        ...(input.enabledProxy && {
          registryId: input.registryId,
          organization: input.organization,
        }),
      }),
    meta: {
      errorMessage: i18n.t('projects.createModal.createFailed'),
      invalidates: [projectKeys.lists(), authKeys.projectRoles()],
    } satisfies NotificationMeta,
  })
}

export function deleteProjectMutationOptions() {
  return mutationOptions({
    mutationFn: (name: string) =>
      Projects.DeleteProject({ name }),
    meta: {
      errorMessage: i18n.t('projects.deleteModal.deleteFailed'),
      invalidates: [projectKeys.lists(), authKeys.projectRoles()],
    } satisfies NotificationMeta,
  })
}

export function updateProjectMutationOptions() {
  return mutationOptions({
    mutationFn: (input: UpdateProjectRequest) =>
      Projects.UpdateProject(input),
    meta: {
      errorMessage: i18n.t('projects.detail.settingsPage.updateError'),
      invalidates: [projectKeys.all],
    } satisfies NotificationMeta,
  })
}
