type NamespaceBundle = Record<string, Record<string, unknown>>

const modules = import.meta.glob<NamespaceBundle>(
  '../locales/**/*.json',
  {
    eager: true,
  },
)

export function loadLocale(lang: string) {
  const result: NamespaceBundle = {}

  for (const path in modules) {
    if (!path.includes(`/${lang}/`)) {
      continue
    }

    const name = path
      .split(`/${lang}/`)[1]
      .replace(/\.json$/, '')
      .replaceAll('/', '.')

    result[name] = modules[path].default
  }

  if (!Object.keys(result).length) {
    throw new Error(`Missing locale: ${lang}`)
  }

  return result
}
