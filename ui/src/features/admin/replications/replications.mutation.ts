import {
  type CreateSyncPolicyRequest,
  type CreateSyncTaskRequest,
  type CreateSyncTaskResponse,
  type DeleteSyncPolicyRequest,
  SyncPolicy,
  type UpdateSyncPolicyRequest,
  type UpdateSyncPolicySwitchRequest,
} from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import { mutationOptions } from '@tanstack/react-query'

import i18n from '@/i18n'

import { adminReplicationKeys } from './replications.query'

import type { NotificationMeta } from '@/types/tanstack-query'

function requireSyncPolicyId(id?: number) {
  if (id == null) {
    throw new Error(i18n.t('routes.admin.replications.errors.missingSyncPolicyId'))
  }

  return id
}

export function createReplicationMutationOptions() {
  return mutationOptions({
    mutationFn: (input: CreateSyncPolicyRequest) => SyncPolicy.CreateSyncPolicy(input),
    meta: {
      successMessage: i18n.t('routes.admin.replications.notifications.createSuccess'),
      errorMessage: i18n.t('routes.admin.replications.notifications.createError'),
      invalidates: [adminReplicationKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function updateReplicationMutationOptions() {
  return mutationOptions({
    mutationFn: (input: UpdateSyncPolicyRequest) =>
      SyncPolicy.UpdateSyncPolicy({
        ...input,
        syncPolicyId: requireSyncPolicyId(input.syncPolicyId),
      }),
    meta: {
      successMessage: i18n.t('routes.admin.replications.notifications.updateSuccess'),
      errorMessage: i18n.t('routes.admin.replications.notifications.updateError'),
      invalidates: [adminReplicationKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function deleteReplicationMutationOptions() {
  return mutationOptions({
    mutationFn: ({ syncPolicyId }: DeleteSyncPolicyRequest) => SyncPolicy.DeleteSyncPolicy({
      syncPolicyId: requireSyncPolicyId(syncPolicyId),
    }),
    meta: {
      successMessage: i18n.t('routes.admin.replications.notifications.deleteSuccess'),
      errorMessage: i18n.t('routes.admin.replications.notifications.deleteError'),
      invalidates: [adminReplicationKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function syncReplicationMutationOptions() {
  return mutationOptions<CreateSyncTaskResponse, Error, CreateSyncTaskRequest>({
    mutationFn: ({ syncPolicyId }) => SyncPolicy.CreateSyncTask({
      syncPolicyId: requireSyncPolicyId(syncPolicyId),
    }),
    meta: {
      successMessage: i18n.t('routes.admin.replications.notifications.syncSuccess'),
      errorMessage: i18n.t('routes.admin.replications.notifications.syncError'),
    } satisfies NotificationMeta,
  })
}

export function switchReplicationMutationOptions() {
  return mutationOptions({
    mutationFn: ({
      syncPolicyId, isDisabled,
    }: UpdateSyncPolicySwitchRequest) =>
      SyncPolicy.UpdateSyncPolicySwitch({
        syncPolicyId: requireSyncPolicyId(syncPolicyId),
        isDisabled,
      }),
    meta: {
      successMessage: i18n.t('routes.admin.replications.notifications.switchSuccess'),
      errorMessage: i18n.t('routes.admin.replications.notifications.switchError'),
      invalidates: [adminReplicationKeys.lists()],
    } satisfies NotificationMeta,
  })
}
