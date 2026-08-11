import DOMPurify from 'dompurify'
import MarkdownItAsync from 'markdown-it-async'
import MarkdownItGitHubAlerts from 'markdown-it-github-alerts'

import type { HighlighterCore } from 'shiki/core'

type Renderer = ReturnType<typeof MarkdownItAsync>

/**
 * Grammar loaders keyed by fence language. Loaded lazily by the async
 * highlight callback — only grammars actually used by the document being
 * rendered are fetched (cpp alone is ~800 kB).
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

/** Map fence aliases to the canonical grammar name Shiki registers. */
const LANG_ALIASES: Record<string, string> = {
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
let rendererPromise: Promise<Renderer> | null = null

/** Strip a leading YAML frontmatter block (as found in Hugging Face model READMEs). */
export function stripFrontmatter(content: string): string {
  const match = /^---\r?\n[\s\S]*?\r?\n---\r?\n?/.exec(content)

  return match ? content.slice(match[0].length) : content
}

async function getHighlighter(): Promise<HighlighterCore> {
  highlighterPromise ??= (async () => {
    const [{ createHighlighterCore }, { createJavaScriptRegexEngine }] = await Promise.all([
      import('shiki/core'),
      import('shiki/engine/javascript'),
    ])

    return createHighlighterCore({
      themes: [
        import('@shikijs/themes/github-light'),
        import('@shikijs/themes/github-dark'),
      ],
      langs: [],
      engine: createJavaScriptRegexEngine({ forgiving: true }),
    })
  })()

  return highlighterPromise
}

/** Resolve a fence language to a loaded grammar, fetching it on demand. */
async function resolveLang(highlighter: HighlighterCore, lang: string): Promise<string> {
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

function createRenderer(): Renderer {
  const md = MarkdownItAsync({
    html: true,
    linkify: true,
    warnOnSyncRender: true,
    highlight: async (code, lang) => {
      const highlighter = await getHighlighter()
      const language = await resolveLang(highlighter, lang)

      return highlighter.codeToHtml(code, {
        lang: language,
        themes: {
          light: 'github-light',
          dark: 'github-dark',
        },
        defaultColor: 'light',
      })
    },
  })

  md.use(MarkdownItGitHubAlerts)

  // Open links in a new tab; DOMPurify keeps target only when rel is set.
  const defaultLinkRenderer = md.renderer.rules.link_open
    ?? ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))

  md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
    tokens[idx].attrSet('target', '_blank')
    tokens[idx].attrSet('rel', 'noopener noreferrer')

    return defaultLinkRenderer(tokens, idx, options, env, self)
  }

  return md
}

/** Render markdown to sanitized HTML (GitHub-flavored: alerts, autolinks, Shiki-highlighted code). */
export async function renderMarkdown(content: string): Promise<string> {
  rendererPromise ??= Promise.resolve(createRenderer())
  const md = await rendererPromise
  const html = await md.renderAsync(stripFrontmatter(content))

  return DOMPurify.sanitize(html, { ADD_ATTR: ['target'] })
}
