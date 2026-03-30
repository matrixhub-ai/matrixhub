import { z } from 'zod'

export const DEFAULT_USERS_PAGE = 1
export const DEFAULT_USERS_PAGE_SIZE = 10

import i18n from '@/i18n'

// TODO: need confirm how to get error message in a better way
export function t(key: string) {
  return i18n.getFixedT(i18n.resolvedLanguage ?? i18n.language)(key)
}

export const usersSearchSchema = z.object({
  page: z.coerce.number().int().positive().default(DEFAULT_USERS_PAGE),
  query: z.string().trim().optional().catch(undefined),
})

export type UsersSearch = z.infer<typeof usersSearchSchema>

function requiredString(message: string) {
  return z.string().trim().min(1, message)
}

function refineConfirmPassword<
  TSchema extends z.ZodType<{
    password: string
    confirmPassword: string
  }, unknown>,
>(schema: TSchema, message: string) {
  return schema.refine(
    value => value.password === value.confirmPassword,
    {
      message,
      path: ['confirmPassword'],
    },
  )
}

export const createUserFormSchema = () => refineConfirmPassword(z.object({
  username: requiredString(t('routes.admin.users.validation.usernameRequired')),
  password: requiredString(t('routes.admin.users.validation.passwordRequired')),
  confirmPassword: requiredString(t('routes.admin.users.validation.confirmPasswordRequired')),
  isAdmin: z.boolean(),
}), t('routes.admin.users.validation.passwordMismatch'))

export const resetUserPasswordFormSchema = () => refineConfirmPassword(z.object({
  password: requiredString(t('routes.admin.users.validation.passwordRequired')),
  confirmPassword: requiredString(
    t('routes.admin.users.validation.confirmPasswordRequired'),
  ),
}), t('routes.admin.users.validation.passwordMismatch'))
