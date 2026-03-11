import i18n from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'

import { loadLocale } from './loadLocale'

export const SUPPORTED_LANGUAGES = ['en', 'zh'] as const
export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number]

export const DEFAULT_LANGUAGE: SupportedLanguage = 'en'
export const LANGUAGE_STORAGE_KEY = 'lang'

const hasWindow = typeof window !== 'undefined'

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

function getStoredLanguage(): SupportedLanguage | null {
  if (!hasWindow) {
    return null
  }

  return normalizeLanguage(window.localStorage.getItem(LANGUAGE_STORAGE_KEY))
}

function getBrowserLanguage(): SupportedLanguage | null {
  if (!hasWindow || !window.navigator) {
    return null
  }

  const candidate = window.navigator.language
    || window.navigator.languages?.[0]

  return normalizeLanguage(candidate)
}

const initialLanguage = getStoredLanguage()
  ?? getBrowserLanguage()
  ?? DEFAULT_LANGUAGE

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    fallbackLng: DEFAULT_LANGUAGE,
    supportedLngs: [...SUPPORTED_LANGUAGES],
    load: 'languageOnly',
    nonExplicitSupportedLngs: true,
    interpolation: {
      escapeValue: false,
    },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: [],
      lookupLocalStorage: LANGUAGE_STORAGE_KEY,
    },
    lng: initialLanguage,
  })

i18n.on('languageChanged', (lng) => {
  const normalized = normalizeLanguage(lng) ?? DEFAULT_LANGUAGE
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
