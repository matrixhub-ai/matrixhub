import DOMPurify from 'dompurify'
import MarkdownItAnchor from 'markdown-it-anchor'
import MarkdownItAsync from 'markdown-it-async'
import MarkdownItGitHubAlerts from 'markdown-it-github-alerts'

import { highlightCode } from '@/shared/utils/shiki'

type Renderer = ReturnType<typeof MarkdownItAsync>

let rendererPromise: Promise<Renderer> | null = null

/**
 * GitHub-style heading slug: lowercase, drop punctuation, spaces to hyphens.
 * Duplicate headings are suffixed `-1`, `-2`, … by markdown-it-anchor.
 */
export function slugifyHeading(text: string): string {
  return text
    .trim()
    .toLowerCase()
    .replace(/[^\p{L}\p{N}\s_-]/gu, '')
    .replace(/\s+/g, '-')
}

/** A YAML mapping entry (`key: value`), sequence item, comment, or indented continuation. */
const YAML_LINE = /^(?:[\w.-]+\s*:(?:\s.*)?|-\s.*|\s+\S.*|#\s.*)?$/

/**
 * Strip a leading YAML frontmatter block (as found in Hugging Face model
 * READMEs). Only strips when the block looks like a YAML mapping, so a
 * document that merely opens with a `---` thematic break is left intact.
 */
export function stripFrontmatter(content: string): string {
  const match = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?/.exec(content)

  if (!match) {
    return content
  }

  const lines = match[1].split(/\r?\n/)
  const startsWithKey = /^[\w.-]+\s*:/.test(lines[0] ?? '')

  return startsWithKey && lines.every(line => YAML_LINE.test(line))
    ? content.slice(match[0].length)
    : content
}

/**
 * Sanitizer for untrusted README HTML. Beyond DOMPurify's script-stripping
 * defaults we drop `<style>` and form controls, and only keep inline `style`
 * when it is purely token colouring (what Shiki emits) so repository content
 * cannot restyle or overlay the surrounding UI.
 */
const purifier = DOMPurify()

purifier.setConfig({
  FORBID_TAGS: ['style', 'form', 'input', 'button', 'textarea', 'select', 'option', 'label', 'fieldset'],
  ADD_ATTR: ['target'],
})

const COLOR_VALUE = /^(?:#[0-9a-f]{3,8}|rgba?\([\d\s.,%/]+\)|inherit|transparent)$/i
const COLOR_PROPERTY = /^(?:color|background-color|--shiki-(?:dark|light)(?:-bg)?)$/

function isColorOnlyStyle(style: string): boolean {
  return style
    .split(';')
    .map(declaration => declaration.trim())
    .filter(Boolean)
    .every((declaration) => {
      const [property, ...rest] = declaration.split(':')
      const value = rest.join(':').trim()

      return COLOR_PROPERTY.test(property.trim()) && COLOR_VALUE.test(value)
    })
}

purifier.addHook('uponSanitizeAttribute', (_node, data) => {
  if (data.attrName === 'style' && !isColorOnlyStyle(data.attrValue)) {
    data.keepAttr = false
  }
})

// Every external link — markdown-generated or raw HTML — opens in a new tab.
purifier.addHook('afterSanitizeAttributes', (node) => {
  if (node.tagName !== 'A') {
    return
  }

  const href = node.getAttribute('href') ?? ''

  if (href.startsWith('#')) {
    node.removeAttribute('target')

    return
  }

  node.setAttribute('target', '_blank')
  node.setAttribute('rel', 'noopener noreferrer')
})

function createRenderer(): Renderer {
  const md = MarkdownItAsync({
    html: true,
    linkify: true,
    warnOnSyncRender: true,
    highlight: (code, lang) => highlightCode(code, lang),
  })

  md.use(MarkdownItGitHubAlerts)
  md.use(MarkdownItAnchor, {
    slugify: slugifyHeading,
    tabIndex: false,
    // Wrap heading text in a link to its own anchor so a click records the hash in the URL.
    permalink: MarkdownItAnchor.permalink.headerLink({ safariReaderFix: true }),
  })

  return md
}

/** Render markdown to sanitized HTML (GitHub-flavored: alerts, autolinks, Shiki-highlighted code). */
export async function renderMarkdown(content: string): Promise<string> {
  rendererPromise ??= Promise.resolve(createRenderer())
  const md = await rendererPromise
  const html = await md.renderAsync(stripFrontmatter(content))

  return purifier.sanitize(html)
}
