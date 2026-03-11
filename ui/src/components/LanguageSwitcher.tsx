import { Select } from '@mantine/core'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import {
  DEFAULT_LANGUAGE,
  LANGUAGE_STORAGE_KEY,
  normalizeLanguage,
  SUPPORTED_LANGUAGES,
} from '@/i18n'

export function LanguageSwitcher() {
  const { i18n } = useTranslation()

  const currentLanguage = useMemo(() => {
    const resolved = normalizeLanguage(i18n.resolvedLanguage)
    const raw = normalizeLanguage(i18n.language)

    return resolved ?? raw ?? DEFAULT_LANGUAGE
  }, [i18n.language, i18n.resolvedLanguage])

  const handleChange = (value: string | null) => {
    if (!value) {
      return
    }

    const normalized = normalizeLanguage(value) ?? DEFAULT_LANGUAGE

    if (!SUPPORTED_LANGUAGES.includes(normalized)) {
      return
    }

    if (typeof window !== 'undefined') {
      window.localStorage.setItem(LANGUAGE_STORAGE_KEY, normalized)
    }

    void i18n.changeLanguage(normalized)
  }

  return (
    <Select
      size="xs"
      value={currentLanguage}
      onChange={handleChange}
      allowDeselect={false}
      data={[
        {
          value: 'en',
          label: 'English',
        },
        {
          value: 'zh',
          label: '中文',
        },
      ]}
    />
  )
}
