import i18n from '@/i18n'

import type { ZodString } from 'zod'

export const NAME_START_CHAR_REGEX = /^[a-z0-9]/u
export const NAME_VALID_CHARS_REGEX = /^[a-z0-9._-]+$/u

function t(key: string) {
  return i18n.getFixedT(i18n.resolvedLanguage ?? i18n.language)(key)
}

export function withNameStartCharRule(schema: ZodString) {
  return schema.regex(NAME_START_CHAR_REGEX, t('common.validation.nameStartChar'))
}

export function withNameValidCharsRule(schema: ZodString) {
  return schema.regex(NAME_VALID_CHARS_REGEX, t('common.validation.nameInvalidChars'))
}
