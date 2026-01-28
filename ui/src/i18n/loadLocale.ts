type NamespaceBundle = Record<string, Record<string, unknown>>

const modules = import.meta.glob<NamespaceBundle>('../locales/**/*.json')

export async function loadLocale(lang: string) {
    const entries = Object.entries(modules).filter(([path]) =>
        path.startsWith(`../locales/${lang}/`)
    )

    if (!entries.length) {
        throw new Error(`Missing locale: ${lang}`)
    }

    const result: NamespaceBundle = {}

    for (const [path, loader] of entries) {
        const mod = await loader()

        const name = path.split('/').pop()!.replace('.json', '')

        result[name] = mod.default
    }

    return result
}
