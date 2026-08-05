import {
  type CreateUserRequest,
  type DeleteUserRequest,
  type ResetUserPasswordRequest,
  type SetUserSysAdminRequest,
  type User,
  Users,
} from '@matrixhub/api-ts/v1alpha1/user.pb'
import { mutationOptions } from '@tanstack/react-query'

import i18n from '@/i18n'

import { adminUserKeys } from './users.query'

import type { NotificationMeta } from '@/types/tanstack-query'

function requireUserId(id?: number) {
  if (id == null) {
    throw new Error(i18n.t('routes.admin.users.errors.missingUserId'))
  }

  return id
}

type UserSysAdminMutationInput = Pick<SetUserSysAdminRequest, 'id'>

interface BatchDeleteUsersResult {
  partial: boolean
  successCount: number
  failureCount: number
  failed: PromiseRejectedResult[]
}

function userSysAdminMutationOptions({
  sysadminFlag,
  successMessage,
  errorMessage,
}: {
  sysadminFlag: boolean
  successMessage: string
  errorMessage: string
}) {
  return mutationOptions({
    mutationFn: ({ id }: UserSysAdminMutationInput) =>
      Users.SetUserSysAdmin({
        id: requireUserId(id),
        sysadminFlag,
      }),
    meta: {
      successMessage,
      errorMessage,
      invalidates: [adminUserKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function createUserMutationOptions() {
  return mutationOptions({
    mutationFn: (input: CreateUserRequest) => Users.CreateUser(input),
    meta: {
      successMessage: i18n.t('routes.admin.users.notifications.createSuccess'),
      errorMessage: i18n.t('routes.admin.users.notifications.createError'),
      invalidates: [adminUserKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function resetUserPasswordMutationOptions() {
  return mutationOptions({
    mutationFn: ({
      id,
      password,
    }: ResetUserPasswordRequest) =>
      Users.ResetUserPassword({
        id: requireUserId(id),
        password,
      }),
    meta: {
      successMessage: i18n.t('routes.admin.users.notifications.resetPasswordSuccess'),
      errorMessage: i18n.t('routes.admin.users.notifications.resetPasswordError'),
      invalidates: [adminUserKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function setUserAdminMutationOptions() {
  return userSysAdminMutationOptions({
    sysadminFlag: true,
    successMessage: i18n.t('routes.admin.users.notifications.setAdminSuccess'),
    errorMessage: i18n.t('routes.admin.users.notifications.setAdminError'),
  })
}

export function revokeUserAdminMutationOptions() {
  return userSysAdminMutationOptions({
    sysadminFlag: false,
    successMessage: i18n.t('routes.admin.users.notifications.revokeAdminSuccess'),
    errorMessage: i18n.t('routes.admin.users.notifications.revokeAdminError'),
  })
}

export function deleteUserMutationOptions() {
  return mutationOptions({
    mutationFn: ({ id }: DeleteUserRequest) =>
      Users.DeleteUser({
        id: requireUserId(id),
      }),
    meta: {
      successMessage: i18n.t('routes.admin.users.notifications.deleteSuccess'),
      errorMessage: i18n.t('routes.admin.users.notifications.deleteError'),
      invalidates: [adminUserKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function batchDeleteUsersMutationOptions() {
  return mutationOptions({
    mutationFn: async (users: readonly User[]): Promise<BatchDeleteUsersResult> => {
      if (users.length === 0) {
        throw new Error(i18n.t('routes.admin.users.errors.noUsersSelected'))
      }

      const ids = users.map(user => requireUserId(user.id))
      // TODO: Use a backend batch-delete API when available to avoid partial-success states.
      const results = await Promise.allSettled(
        ids.map(id => Users.DeleteUser({ id })),
      )
      const failed = results.filter(
        (result): result is PromiseRejectedResult => result.status === 'rejected',
      )

      if (failed.length === 0) {
        return {
          partial: false,
          successCount: ids.length,
          failureCount: 0,
          failed: [],
        }
      }

      if (failed.length === ids.length) {
        throw failed[0].reason instanceof Error
          ? failed[0].reason
          : new Error(String(failed[0].reason))
      }

      // Partial failure: return result object, do not throw
      return {
        partial: true,
        successCount: ids.length - failed.length,
        failureCount: failed.length,
        failed,
      }
    },
    meta: {
      successMessage: (data) => {
        const result = data as BatchDeleteUsersResult

        return result.partial
          ? i18n.t('routes.admin.users.notifications.batchDeletePartialError', {
              successCount: result.successCount,
              failureCount: result.failureCount,
            })
          : i18n.t('routes.admin.users.notifications.batchDeleteSuccess')
      },
      successColor: data => ((data as BatchDeleteUsersResult).partial ? 'yellow' : 'green'),
      errorMessage: i18n.t('routes.admin.users.notifications.batchDeleteError'),
      invalidates: [adminUserKeys.lists()],
    } satisfies NotificationMeta,
  })
}
