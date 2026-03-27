import {
  type CreateRegistryRequest,
  type PingRegistryRequest,
  Registries,
  type UpdateRegistryRequest,
} from '@matrixhub/api-ts/v1alpha1/registry.pb'
import { mutationOptions } from '@tanstack/react-query'

import i18n from '@/i18n'

import { adminRegistryKeys } from './registries.query'

import type { NotificationMeta } from '@/types/tanstack-query'

function requireRegistryId(id?: number) {
  if (id == null) {
    throw new Error(i18n.t('routes.admin.registries.errors.missingRegistryId'))
  }

  return id
}

export function createRegistryMutationOptions() {
  return mutationOptions({
    mutationFn: (input: CreateRegistryRequest) => Registries.CreateRegistry(input),
    meta: {
      successMessage: i18n.t('routes.admin.registries.notifications.createSuccess'),
      errorMessage: i18n.t('routes.admin.registries.notifications.createError'),
      invalidates: [adminRegistryKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function updateRegistryMutationOptions() {
  return mutationOptions({
    mutationFn: (input: UpdateRegistryRequest) => Registries.UpdateRegistry({
      ...input,
      id: requireRegistryId(input.id),
    }),
    meta: {
      successMessage: i18n.t('routes.admin.registries.notifications.updateSuccess'),
      errorMessage: i18n.t('routes.admin.registries.notifications.updateError'),
      invalidates: [adminRegistryKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function deleteRegistryMutationOptions() {
  return mutationOptions({
    mutationFn: ({ id }: { id?: number }) => Registries.DeleteRegistry({
      id: requireRegistryId(id),
    }),
    meta: {
      successMessage: i18n.t('routes.admin.registries.notifications.deleteSuccess'),
      errorMessage: i18n.t('routes.admin.registries.notifications.deleteError'),
      invalidates: [adminRegistryKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function pingRegistryMutationOptions() {
  return mutationOptions({
    mutationFn: (input: PingRegistryRequest) => Registries.PingRegistry(input),
    meta: {
      successMessage: i18n.t('routes.admin.registries.notifications.pingSuccess'),
      errorMessage: i18n.t('routes.admin.registries.notifications.pingError'),
    } satisfies NotificationMeta,
  })
}
