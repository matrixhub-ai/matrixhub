import i18n from '@/i18n'
import { formatDateTime } from '@/shared/utils/date.ts'

import type { TFunction } from 'i18next'

export const formatExpiredAt = (
  expiredAt: string | undefined,
  t: TFunction = i18n.t.bind(i18n),
) => {
  if (!expiredAt) {
    return t('profile.tokenPermanent')
  }

  return formatDateTime(expiredAt)
}
