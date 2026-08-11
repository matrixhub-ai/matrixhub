import {
  Box, Center, Loader,
} from '@mantine/core'
import { useEffect, useState } from 'react'

import { renderMarkdown } from '../markdown/renderer'

import 'github-markdown-css/github-markdown-light.css'
import './MarkdownViewer.css'

interface MarkdownViewerProps {
  content: string
}

export function MarkdownViewer({ content }: MarkdownViewerProps) {
  const [html, setHtml] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    renderMarkdown(content).then((rendered) => {
      if (!cancelled) {
        setHtml(rendered)
      }
    })

    return () => {
      cancelled = true
    }
  }, [content])

  if (html == null) {
    return (
      <Center p="xl" bg="var(--mantine-color-default-hover)">
        <Loader size="sm" />
      </Center>
    )
  }

  return (
    <Box
      className="markdown-body"
      p="md"
      bg="var(--mantine-color-default-hover)"
      bdrs="md"
      // eslint-disable-next-line @eslint-react/dom/no-dangerously-set-innerhtml -- html is sanitized by DOMPurify in renderMarkdown
      dangerouslySetInnerHTML={{ __html: html }}
    />
  )
}
