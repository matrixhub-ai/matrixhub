import {
  CurrentUser,
  type CreateAccessTokenRequest,
  type CreateSSHKeyRequest,
  type ResetPasswordRequest,
} from '@matrixhub/api-ts/v1alpha1/current_user.pb'
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
      errorMessage: i18n.t('profile.sshKey.delete.error'),
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
      errorMessage: i18n.t('profile.sshKey.create.error'),
      invalidates: [profileKeys.sshKeys],
    } satisfies NotificationMeta,
  })
}

export function resetPasswordMutationOptions() {
  return mutationOptions({
    mutationFn: (params: ResetPasswordRequest) =>
      CurrentUser.ResetPassword(params),
    meta: {
      successMessage: i18n.t('profile.passwordChanged'),
      errorMessage: i18n.t('profile.passwordChangeError'),
    } satisfies NotificationMeta,
  })
}

export function createAccessTokenMutationOptions() {
  return mutationOptions({
    mutationFn: (params: CreateAccessTokenRequest) =>
      CurrentUser.CreateAccessToken(params),
    meta: {
      successMessage: i18n.t('profile.tokenCreated'),
      errorMessage: i18n.t('profile.tokenCreateError'),
      invalidates: [profileKeys.accessTokens],
    } satisfies NotificationMeta,
  })
}

export function deleteAccessTokenMutationOptions() {
  return mutationOptions({
    mutationFn: (id: number) =>
      CurrentUser.DeleteAccessToken({ id }),
    meta: {
      successMessage: i18n.t('profile.tokenDeleted'),
      errorMessage: i18n.t('profile.tokenDeleteError'),
      invalidates: [profileKeys.accessTokens],
    } satisfies NotificationMeta,
  })
}
