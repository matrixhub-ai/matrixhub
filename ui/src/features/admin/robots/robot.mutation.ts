import {
  type DeleteRobotAccountRequest, type GetRobotAccountResponse,
  type RefreshRobotAccountTokenRequest,
  RobotAccountStatus,
  Robots,
} from '@matrixhub/api-ts/v1alpha1/robot.pb'
import { mutationOptions } from '@tanstack/react-query'

import i18n from '@/i18n'

import { adminRobotKeys } from './robot.query'

import type { NotificationMeta } from '@/types/tanstack-query'

function requireRobotId(id?: number) {
  if (id == null) {
    throw new Error(i18n.t('routes.admin.robots.errors.missingRobotId'))
  }

  return id
}

function robotStatusMutationOptions({
  status,
  successMessage,
  errorMessage,
}: {
  status: RobotAccountStatus
  successMessage: string
  errorMessage: string
}) {
  return mutationOptions({
    mutationFn: (robot: GetRobotAccountResponse) => Robots.UpdateRobotAccount({
      ...robot,
      status,
    }),
    meta: {
      successMessage,
      errorMessage,
      invalidates: [adminRobotKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function enableRobotAccountMutationOptions() {
  return robotStatusMutationOptions({
    status: RobotAccountStatus.ROBOT_ACCOUNT_STATUS_ENABLED,
    successMessage: i18n.t('routes.admin.robots.notifications.enableSuccess'),
    errorMessage: i18n.t('routes.admin.robots.notifications.enableError'),
  })
}

export function disableRobotAccountMutationOptions() {
  return robotStatusMutationOptions({
    status: RobotAccountStatus.ROBOT_ACCOUNT_STATUS_DISABLED,
    successMessage: i18n.t('routes.admin.robots.notifications.disableSuccess'),
    errorMessage: i18n.t('routes.admin.robots.notifications.disableError'),
  })
}

export function deleteRobotAccountMutationOptions() {
  return mutationOptions({
    mutationFn: ({ id }: DeleteRobotAccountRequest) => Robots.DeleteRobotAccount({
      id: requireRobotId(id),
    }),
    meta: {
      successMessage: i18n.t('routes.admin.robots.notifications.deleteSuccess'),
      errorMessage: i18n.t('routes.admin.robots.notifications.deleteError'),
      invalidates: [adminRobotKeys.lists()],
    } satisfies NotificationMeta,
  })
}

export function refreshRobotAccountTokenMutationOptions() {
  return mutationOptions({
    mutationFn: (input: RefreshRobotAccountTokenRequest) => Robots.RefreshRobotAccountToken({
      ...input,
      id: requireRobotId(input.id),
      autoGenerate: input.autoGenerate ?? true,
    }),
    meta: {
      successMessage: i18n.t('routes.admin.robots.notifications.refreshTokenSuccess'),
      errorMessage: i18n.t('routes.admin.robots.notifications.refreshTokenError'),
      invalidates: [adminRobotKeys.lists()],
    } satisfies NotificationMeta,
  })
}
