import i18n from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'

import { setDayjsLocal } from '@/i18n/dayjs.ts'

import { loadLocale } from './loadLocale'

export const SUPPORTED_LANGUAGES = ['en', 'zh'] as const
export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number]

export const DEFAULT_LANGUAGE: SupportedLanguage = 'en'
export const LANGUAGE_STORAGE_KEY = 'lang'

export function normalizeLanguage(
  value: string | null | undefined,
): SupportedLanguage | null {
  if (!value) {
    return null
  }

  const normalized = value.toLowerCase().split('-')[0]

  if (SUPPORTED_LANGUAGES.includes(normalized as SupportedLanguage)) {
    return normalized as SupportedLanguage
  }

  return null
}

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    fallbackLng: DEFAULT_LANGUAGE,
    supportedLngs: [...SUPPORTED_LANGUAGES],
    load: 'languageOnly',
    interpolation: {
      escapeValue: false,
    },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: [],
      lookupLocalStorage: LANGUAGE_STORAGE_KEY,
    },
  })

const initialLanguage = normalizeLanguage(i18n.language) ?? DEFAULT_LANGUAGE

setDayjsLocal(initialLanguage)

i18n.on('languageChanged', (lng) => {
  const normalized = normalizeLanguage(lng) ?? DEFAULT_LANGUAGE

  setDayjsLocal(normalized)
  const bundles = loadLocale(normalized)

  let resourceBundle: Record<string, unknown> = {}

  Object.entries(bundles).forEach(([ns, data]) => {
    resourceBundle = {
      ...resourceBundle,
      [ns]: data,
    }
  })

  i18n.addResourceBundle(normalized, 'translation', resourceBundle)
})

export default i18n
