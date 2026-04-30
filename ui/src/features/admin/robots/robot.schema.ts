import { z } from 'zod'

import i18n from '@/i18n'

export const refreshRobotTokenFormDefaults = {
  autoGenerate: true,
  token: '',
  confirmToken: '',
}

function t(key: string) {
  return i18n.getFixedT(i18n.resolvedLanguage ?? i18n.language)(key)
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

export type RefreshRobotTokenFormValues = z.infer<typeof refreshRobotTokenSchema>
