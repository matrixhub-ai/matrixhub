import dayjs from 'dayjs'
import z from 'zod'

import i18n from '@/i18n'

function t(key: string, options?: Record<string, unknown>) {
  return i18n.getFixedT(i18n.resolvedLanguage ?? i18n.language)(key, options)
}

export const sshKeyNameSchema = z.string().min(1, {
  error: t('common.validation.fieldRequired', { field: t('profile.sshKey.create.name') }),
})

export const sshKeyPublicKeySchema = z.string().min(1, {
  error: t('common.validation.fieldRequired', { field: t('profile.sshKey.create.publicKey') }),
})

export const expireAtSchema = z.string().refine((value) => {
  if (!value) {
    return true
  }

  return !dayjs(value).startOf('day').isBefore(dayjs().startOf('day'))
}, { error: t('profile.expireTimeError') })
