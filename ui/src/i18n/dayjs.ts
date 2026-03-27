import dayjs from 'dayjs'
import 'dayjs/locale/en'
import 'dayjs/locale/zh-cn'
import relativeTime from 'dayjs/plugin/relativeTime'

import type { SupportedLanguage } from '@/i18n/index.ts'

dayjs.extend(relativeTime)

const dayjsLocaleMap: Record<SupportedLanguage, string> = {
  en: 'en',
  zh: 'zh-cn',
}

export function setDayjsLocale(lang: SupportedLanguage) {
  dayjs.locale(dayjsLocaleMap[lang])

  return lang
}
