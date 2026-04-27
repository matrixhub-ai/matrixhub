import { CurrentUser, type CreateSSHKeyRequest } from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import { mutationOptions } from '@tanstack/react-query'

import i18n from '@/i18n'

import { profileKeys } from './profile.query'

import type { NotificationMeta } from '@/types/tanstack-query'

export function deleteSshKeyMutationOptions() {
  return mutationOptions({
    mutationFn: (id: number) =>
      CurrentUser.DeleteSSHKey({ id }),
    meta: {
      successMessage: i18n.t('profile.sshKey.delete.success'),
      invalidates: [profileKeys.sshKeys],
    } satisfies NotificationMeta,
  })
}

export function createSshKeyMutationOptions() {
  return mutationOptions({
    mutationFn: (params: CreateSSHKeyRequest) =>
      CurrentUser.CreateSSHKey(params),
    meta: {
      successMessage: i18n.t('profile.sshKey.create.success'),
      invalidates: [profileKeys.sshKeys],
    } satisfies NotificationMeta,
  })
}
