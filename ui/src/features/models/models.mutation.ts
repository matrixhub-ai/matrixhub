import {
  type CreateModelRequest,
  type DeleteModelRequest,
  type UpdateModelSettingRequest,
  Models,
} from '@matrixhub/api-ts/v1alpha1/model.pb'
import { mutationOptions } from '@tanstack/react-query'

import { modelKeys } from '@/features/models/models.query'
import i18n from '@/i18n'

import type { NotificationMeta } from '@/types/tanstack-query'

export function deleteModelMutationOptions() {
  return mutationOptions({
    mutationFn: (params: DeleteModelRequest) => Models.DeleteModel(params),
    meta: {
      successMessage: i18n.t('model.settings.delete.success'),
      errorMessage: i18n.t('model.settings.delete.error'),
      invalidates: [modelKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function createModelMutationOptions() {
  return mutationOptions({
    mutationFn: (params: CreateModelRequest) => Models.CreateModel(params),
    meta: {
      successMessage: i18n.t('model.create.success'),
      errorMessage: i18n.t('model.create.error'),
      invalidates: [modelKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function updateModelSettingMutationOptions(targetPopular: boolean) {
  return mutationOptions({
    mutationFn: (params: UpdateModelSettingRequest) => Models.UpdateModelSetting(params),
    meta: {
      successMessage: targetPopular
        ? i18n.t('model.settings.recommended.set.success')
        : i18n.t('model.settings.recommended.unset.success'),
      errorMessage: targetPopular
        ? i18n.t('model.settings.recommended.set.error')
        : i18n.t('model.settings.recommended.unset.error'),
      invalidates: [modelKeys.details(), modelKeys.lists()],
    } satisfies NotificationMeta,
  })
}
