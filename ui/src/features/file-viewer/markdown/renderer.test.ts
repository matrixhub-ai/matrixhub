import {
  describe, expect, it,
} from 'vitest'

import {
  renderMarkdown, slugifyHeading, stripFrontmatter,
} from './renderer'

describe('slugifyHeading', () => {
  it('matches GitHub slugs', () => {
    expect(slugifyHeading('Best Practices')).toBe('best-practices')
    expect(slugifyHeading('Qwen3 Highlights')).toBe('qwen3-highlights')
    expect(slugifyHeading('What\'s new? (v2.0)')).toBe('whats-new-v20')
    expect(slugifyHeading('模型 概览')).toBe('模型-概览')
  })
})

describe('stripFrontmatter', () => {
  it('strips a leading YAML mapping block', () => {
    const input = '---\nlicense: apache-2.0\nbase_model:\n- Qwen/Qwen3-0.6B-Base\n---\n# Title\n'

    expect(stripFrontmatter(input)).toBe('# Title\n')
  })

  it('keeps a document that opens with a thematic break', () => {
    const input = '---\n# Introduction\nSome text.\n---\n# Details\n'

    expect(stripFrontmatter(input)).toBe(input)
  })

  it('leaves content without frontmatter untouched', () => {
    expect(stripFrontmatter('# Title\n')).toBe('# Title\n')
  })
})

describe('renderMarkdown', () => {
  it('opens markdown links in a new tab', async () => {
    const html = await renderMarkdown('[blog](https://example.com)')

    expect(html).toContain('target="_blank"')
    expect(html).toContain('rel="noopener noreferrer"')
  })

  it('opens raw HTML anchors in a new tab', async () => {
    const html = await renderMarkdown('<a href="https://example.com"><img src="badge.svg"></a>')

    expect(html).toContain('target="_blank"')
    expect(html).toContain('rel="noopener noreferrer"')
  })

  it('does not force in-page anchors into a new tab', async () => {
    const html = await renderMarkdown('[Best Practices](#best-practices)')

    expect(html).not.toContain('target=')
  })

  it('gives headings GitHub-style ids so in-page anchors resolve', async () => {
    const html = await renderMarkdown('## Best Practices\n\n## Best Practices\n\n[go](#best-practices)')

    expect(html).toContain('<h2 id="best-practices">')
    expect(html).toContain('<h2 id="best-practices-1">')
    expect(html).not.toContain('tabindex')
  })

  it('links each heading to its own anchor without opening a new tab', async () => {
    const html = await renderMarkdown('## Best Practices')

    expect(html).toContain('<a class="header-anchor" href="#best-practices">')
    expect(html).not.toMatch(/header-anchor[^>]*target=/)
  })

  it('drops style blocks and non-colour inline styles from raw HTML', async () => {
    const html = await renderMarkdown(
      '<style>body{display:none}</style>\n<div style="position:fixed;top:0;color:red">overlay</div>',
    )

    expect(html).not.toContain('<style')
    expect(html).not.toContain('position')
    expect(html).toContain('overlay')
  })

  it('drops form controls from raw HTML', async () => {
    const html = await renderMarkdown('<form><input name="x"><button>go</button></form>')

    expect(html).not.toMatch(/<(form|input|button)/)
  })

  it('keeps Shiki token colours for both schemes', async () => {
    const html = await renderMarkdown('```python\nx = 1\n```')

    expect(html).toContain('class="shiki')
    expect(html).toMatch(/--shiki-light:#[0-9a-f]{6};--shiki-dark:#[0-9a-f]{6}/i)
    expect(html).toMatch(/--shiki-light-bg:#[0-9a-f]{6};--shiki-dark-bg:#[0-9a-f]{6}/i)
  })

  it('renders GitHub alerts', async () => {
    const html = await renderMarkdown('> [!TIP]\n> Use it.')

    expect(html).toContain('markdown-alert-tip')
  })
})
