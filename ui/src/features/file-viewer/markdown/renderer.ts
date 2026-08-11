import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'
import MarkdownItGitHubAlerts from 'markdown-it-github-alerts'

import type { HighlighterCore } from 'shiki/core'

type Renderer = InstanceType<typeof MarkdownIt>

/**
 * Languages we can highlight, loaded on demand — only the grammars actually
 * used by the document being rendered are fetched (cpp alone is ~800 kB).
 */
const LANG_LOADERS: Record<string, () => Promise<unknown>> = {
  bash: () => import('@shikijs/langs/bash'),
  shell: () => import('@shikijs/langs/shell'),
  sh: () => import('@shikijs/langs/bash'),
  zsh: () => import('@shikijs/langs/bash'),
  python: () => import('@shikijs/langs/python'),
  py: () => import('@shikijs/langs/python'),
  javascript: () => import('@shikijs/langs/javascript'),
  js: () => import('@shikijs/langs/javascript'),
  typescript: () => import('@shikijs/langs/typescript'),
  ts: () => import('@shikijs/langs/typescript'),
  json: () => import('@shikijs/langs/json'),
  yaml: () => import('@shikijs/langs/yaml'),
  yml: () => import('@shikijs/langs/yaml'),
  markdown: () => import('@shikijs/langs/markdown'),
  md: () => import('@shikijs/langs/markdown'),
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
  dockerfile: () => import('@shikijs/langs/docker'),
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

/** Fence languages referenced by the document (```lang). */
function collectFenceLangs(content: string): string[] {
  const langs = new Set<string>()

  for (const match of content.matchAll(/^\s*(?:```|~~~)\s*([\w+-]+)/gm)) {
    langs.add(match[1].toLowerCase())
  }

  return [...langs]
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

/** Load the grammars used by `content` (unknown languages fall back to plain text). */
async function loadLangsFor(highlighter: HighlighterCore, content: string): Promise<void> {
  const loaded = new Set(highlighter.getLoadedLanguages())
  const wanted = collectFenceLangs(content)
    .filter(lang => !loaded.has(LANG_ALIASES[lang] ?? lang) && lang in LANG_LOADERS)

  await Promise.all(wanted.map(async (lang) => {
    const mod = await LANG_LOADERS[lang]() as { default: Parameters<HighlighterCore['loadLanguage']>[0] }

    await highlighter.loadLanguage(mod.default)
  }))
}

function createRenderer(highlighter: HighlighterCore): Renderer {
  const md = new MarkdownIt({
    html: true,
    linkify: true,
    highlight: (code, lang) => {
      const canonical = LANG_ALIASES[lang] ?? lang
      const language = highlighter.getLoadedLanguages().includes(canonical) ? canonical : 'text'

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
  const highlighter = await getHighlighter()
  const source = stripFrontmatter(content)

  await loadLangsFor(highlighter, source)
  rendererPromise ??= Promise.resolve(createRenderer(highlighter))
  const md = await rendererPromise
  const html = md.render(source)

  return DOMPurify.sanitize(html, { ADD_ATTR: ['target'] })
}
