import { useForm as _useForm } from '@tanstack/react-form'
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'

/**
 * Wraps TanStack's useForm with automatic i18n error refresh.
 * When the language changes, re-validates only fields that already have errors
 * so their messages update to the new language.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const useForm = ((...args: any[]) => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const form = (_useForm as any)(...args)
  const { i18n } = useTranslation()
  const prevLangRef = useRef(i18n.language)

  useEffect(() => {
    if (prevLangRef.current === i18n.language) {
      return
    }

    prevLangRef.current = i18n.language

    const meta = form.state.fieldMeta as Record<string, { errors?: unknown[] }>

    for (const name of Object.keys(meta)) {
      if (meta[name]?.errors?.length) {
        void form.validateField(name, 'change')
      }
    }
  }, [i18n.language, form])

  return form
}) as typeof _useForm
