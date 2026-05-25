import {
  RobotAccountProjectScope,
  type GetRobotAccountResponse,
} from '@matrixhub/api-ts/v1alpha1/robot.pb'
import { z } from 'zod'

import i18n from '@/i18n'

import type { TFunction } from 'i18next'

export const ROBOT_DESCRIPTION_MAX_LENGTH = 50
export const DEFAULT_ROBOT_EXPIRE_DAYS = 30

export const robotProjectScopeSchema = z.enum([
  RobotAccountProjectScope.ROBOT_ACCOUNT_PROJECT_SCOPE_ALL,
  RobotAccountProjectScope.ROBOT_ACCOUNT_PROJECT_SCOPE_SELECTED,
])

export const robotExpiryModeSchema = z.enum(['never', 'days'])

export const createRobotAccountFormSchema = (t: TFunction) => z.object({
  name: z.string().trim().min(1, t('common.validation.fieldRequired', { field: t('routes.admin.robots.fields.name') })),
  description: z.string().max(ROBOT_DESCRIPTION_MAX_LENGTH, t('common.validation.maxLength', {
    max: ROBOT_DESCRIPTION_MAX_LENGTH,
    field: t('routes.admin.robots.fields.description'),
  })),
  expiryMode: robotExpiryModeSchema,
  expireDays: z.number().int().min(1, t('common.validation.minNumber', {
    min: 1,
    field: t('routes.admin.robots.fields.expiry'),
  })).optional(),
  platformPermissions: z.array(z.string()),
  projectPermissions: z.array(z.string()),
  projectScope: robotProjectScopeSchema,
  projects: z.array(z.string()),
}).superRefine((value, ctx) => {
  if (value.expiryMode === 'days' && value.expireDays == null) {
    ctx.addIssue({
      code: 'custom',
      message: t('common.validation.fieldRequired', {
        field: t('routes.admin.robots.fields.expiry'),
      }),
      path: ['expireDays'],
    })
  }
})

export const refreshRobotTokenFormDefaults = {
  autoGenerate: true,
  token: '',
  confirmToken: '',
}

function t(key: string, args: Record<string, unknown> = {}) {
  return i18n.getFixedT(i18n.resolvedLanguage ?? i18n.language)(key, args)
}

export const refreshRobotTokenSchema = z.object({
  autoGenerate: z.boolean(),
  token: z.string().trim(),
  confirmToken: z.string().trim(),
}).superRefine((value, ctx) => {
  if (value.autoGenerate) {
    return
  }

  if (!value.token) {
    ctx.addIssue({
      code: 'custom',
      message: t('routes.admin.robots.refreshTokenModal.validation.tokenRequired'),
      path: ['token'],
    })
  }

  if (!value.confirmToken) {
    ctx.addIssue({
      code: 'custom',
      message: t('routes.admin.robots.refreshTokenModal.validation.confirmTokenRequired'),
      path: ['confirmToken'],
    })
  }

  if (value.token && value.confirmToken && value.token !== value.confirmToken) {
    ctx.addIssue({
      code: 'custom',
      message: t('routes.admin.robots.refreshTokenModal.validation.tokenMismatch'),
      path: ['confirmToken'],
    })
  }
})

export type RobotAccountFormValues = z.infer<ReturnType<typeof createRobotAccountFormSchema>>
export type RefreshRobotTokenFormValues = z.infer<typeof refreshRobotTokenSchema>

export function getRobotAccountFormDefaults(
  robot?: GetRobotAccountResponse,
): RobotAccountFormValues {
  return {
    name: robot?.name?.replace(/^robot\$/, '') ?? '',
    description: robot?.description ?? '',
    expiryMode: robot?.expireDays === 0 ? 'never' : 'days',
    expireDays: robot?.expireDays || DEFAULT_ROBOT_EXPIRE_DAYS,
    platformPermissions: robot?.platformPermissions ?? [],
    projectPermissions: robot?.projectPermissions ?? [],
    projectScope: robot?.projectScope ?? RobotAccountProjectScope.ROBOT_ACCOUNT_PROJECT_SCOPE_ALL,
    projects: robot?.projects ?? [],
  }
}
