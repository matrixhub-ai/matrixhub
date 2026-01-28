import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { loadLocale } from './loadLocale'

i18n
    .use(LanguageDetector)
    .use(initReactI18next)
    .init({
        fallbackLng: 'en',
        supportedLngs: ['en', 'zh'],
        interpolation: { escapeValue: false },
    })

i18n.on('languageChanged', async (lng) => {
    const bundles = await loadLocale(lng)

    let resourceBundle: Record<string, unknown> = {}

    Object.entries(bundles).forEach(([ns, data]) => {
        resourceBundle = {
            ...resourceBundle,
            [ns]: data,
        }
    })

    i18n.addResourceBundle(lng, 'translation', resourceBundle)
})


export default i18n
