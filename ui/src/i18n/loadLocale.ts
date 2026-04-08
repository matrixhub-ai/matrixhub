type LocaleResource = Record<string, unknown>

const modules = import.meta.glob<Record<string, LocaleResource>>(
  '../locales/**/*.json',
  {
    eager: true,
  },
)

/**
 * Sort locale files so shallower paths load first.
 * Deeper paths load later and override duplicate keys
 * (e.g. `shared/components/A.json` overrides keys in `shared.json`).
 */
function sortLocalePaths(paths: string[]) {
  return [...paths].sort((a, b) => {
    const depthDiff = a.split('/').length - b.split('/').length

    if (depthDiff !== 0) {
      return depthDiff
    }

    return a.localeCompare(b)
  })
}

function isLocaleResource(value: unknown): value is LocaleResource {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function warnDuplicateKey(keyPath: string, sourcePath: string) {
  if (import.meta.env.DEV) {
    console.warn(
      `[i18n] Duplicate key "${keyPath}": overridden by "${sourcePath}".`,
    )
  }
}

/**
 * Wrap a value in nested objects following the given path segments.
 *
 * @example
 * wrapInPath(["shared", "components", "Foo"], { bar: "baz" })
 * // => { shared: { components: { Foo: { bar: "baz" } } } }
 */
function wrapInPath(path: string[], value: LocaleResource): LocaleResource {
  let result: LocaleResource = value

  for (let i = path.length - 1; i >= 0; i--) {
    result = { [path[i]]: result }
  }

  return result
}

/**
 * Recursively merge source into target.
 * Non-conflicting keys are merged in; duplicate leaf keys are overridden with a warning.
 */
function mergeDeep(
  target: Record<string, unknown>,
  source: Record<string, unknown>,
  sourcePath: string,
  parentKey: string,
) {
  for (const key in source) {
    const keyPath = parentKey ? `${parentKey}.${key}` : key
    const targetVal = target[key]
    const sourceVal = source[key]

    if (targetVal === undefined) {
      target[key] = sourceVal
      continue
    }

    if (isLocaleResource(targetVal) && isLocaleResource(sourceVal)) {
      mergeDeep(targetVal, sourceVal, sourcePath, keyPath)
      continue
    }

    warnDuplicateKey(keyPath, sourcePath)
    target[key] = sourceVal
  }
}

/**
 * Dynamically loads and structures locale files based on directory hierarchy.
 * Converts path-based structures (e.g., shared/components/A.json) into nested objects.
 */
export function loadLocale(lang: string) {
  const result: LocaleResource = {}

  for (const path of sortLocalePaths(Object.keys(modules))) {
    if (!path.includes(`/${lang}/`)) {
      continue
    }

    // Extract relative path from language directory, e.g., "shared/components/RouterErrorComponent"
    const pathSegments = path
      .split(`/${lang}/`)[1]
      .replace(/\.json$/, '')
      .split('/')
      .flatMap(segment => segment.split('.'))
      .filter(Boolean)

    const moduleContent = modules[path].default

    if (moduleContent) {
      mergeDeep(result, wrapInPath(pathSegments, moduleContent), path, '')
    }
  }

  if (!Object.keys(result).length) {
    throw new Error(`Missing locale: ${lang}`)
  }

  return result
}
