import dayjs from 'dayjs'
import 'dayjs/locale/en'
import 'dayjs/locale/zh-cn'
import relativeTime from 'dayjs/plugin/relativeTime'

import type { SupportedLanguage } from '@/i18n/index.ts'

dayjs.extend(relativeTime)

export function toDayjsLocale(language: SupportedLanguage) {
  return language === 'zh' ? 'zh-cn' : 'en'
}

export function setDayjsLocal(lang: SupportedLanguage) {
  dayjs.locale(toDayjsLocale(lang))

  return lang
}
