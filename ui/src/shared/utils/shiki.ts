import type { HighlighterCore } from 'shiki/core'

/**
 * Grammar loaders keyed by language. Loaded lazily — only grammars actually
 * used are fetched (cpp alone is ~800 kB).
 */
const LANG_LOADERS: Record<string, () => Promise<unknown>> = {
  bash: () => import('@shikijs/langs/bash'),
  shell: () => import('@shikijs/langs/shell'),
  python: () => import('@shikijs/langs/python'),
  javascript: () => import('@shikijs/langs/javascript'),
  typescript: () => import('@shikijs/langs/typescript'),
  json: () => import('@shikijs/langs/json'),
  yaml: () => import('@shikijs/langs/yaml'),
  markdown: () => import('@shikijs/langs/markdown'),
  go: () => import('@shikijs/langs/go'),
  rust: () => import('@shikijs/langs/rust'),
  c: () => import('@shikijs/langs/c'),
  cpp: () => import('@shikijs/langs/cpp'),
  java: () => import('@shikijs/langs/java'),
  sql: () => import('@shikijs/langs/sql'),
  html: () => import('@shikijs/langs/html'),
  css: () => import('@shikijs/langs/css'),
  diff: () => import('@shikijs/langs/diff'),
  docker: () => import('@shikijs/langs/docker'),
  toml: () => import('@shikijs/langs/toml'),
  ini: () => import('@shikijs/langs/ini'),
}

/** Map language aliases to the canonical grammar name Shiki registers. */
const LANG_ALIASES: Record<string, string> = {
  'c++': 'cpp',
  sh: 'bash',
  zsh: 'bash',
  py: 'python',
  js: 'javascript',
  ts: 'typescript',
  yml: 'yaml',
  md: 'markdown',
  dockerfile: 'docker',
}

let highlighterPromise: Promise<HighlighterCore> | null = null

export async function getHighlighter(): Promise<HighlighterCore> {
  highlighterPromise ??= (async () => {
    const [{ createHighlighterCore }, { createJavaScriptRegexEngine }] = await Promise.all([
      import('shiki/core'),
      import('shiki/engine/javascript'),
    ])

    return createHighlighterCore({
      themes: [
        import('@shikijs/themes/catppuccin-latte'),
        import('@shikijs/themes/catppuccin-mocha'),
      ],
      langs: [],
      engine: createJavaScriptRegexEngine({ forgiving: true }),
    })
  })()

  return highlighterPromise
}

/** Resolve a language to a loaded grammar, fetching it on demand. Unknown languages fall back to `text`. */
export async function resolveLang(highlighter: HighlighterCore, lang: string): Promise<string> {
  const canonical = LANG_ALIASES[lang.toLowerCase()] ?? lang.toLowerCase()

  if (highlighter.getLoadedLanguages().includes(canonical)) {
    return canonical
  }

  const loader = LANG_LOADERS[canonical]

  if (!loader) {
    return 'text'
  }

  const mod = await loader() as { default: Parameters<HighlighterCore['loadLanguage']>[0] }

  await highlighter.loadLanguage(mod.default)

  return canonical
}

/**
 * Highlight `code` to HTML with dual light/dark themes. Both palettes are
 * emitted as CSS variables (`--shiki-light*` / `--shiki-dark*`) rather than
 * inline colours, so callers pick the active one via `color: var(--shiki-light)`
 * etc. and can reuse the same variables for surrounding chrome.
 */
export async function highlightCode(code: string, lang: string): Promise<string> {
  const highlighter = await getHighlighter()
  const language = await resolveLang(highlighter, lang)

  return highlighter.codeToHtml(code, {
    lang: language,
    themes: {
      light: 'catppuccin-latte',
      dark: 'catppuccin-mocha',
    },
    defaultColor: false,
  })
}
