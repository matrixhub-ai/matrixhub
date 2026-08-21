import {
  Alert, Box, Center, Loader,
} from '@mantine/core'
import { useEffect, useState } from 'react'

import { renderMarkdown } from '../markdown/renderer'

import 'github-markdown-css/github-markdown-light.css'
import './MarkdownViewer.css'

interface MarkdownViewerProps {
  content: string
}

export function MarkdownViewer({ content }: MarkdownViewerProps) {
  const [result, setResult] = useState<{
    content: string
    value: Error | string
  } | null>(null)

  useEffect(() => {
    let cancelled = false

    renderMarkdown(content).then(
      html => !cancelled && setResult({
        content,
        value: html,
      }),
      error => !cancelled && setResult({
        content,
        value: error instanceof Error ? error : new Error(String(error)),
      }),
    )

    return () => {
      cancelled = true
    }
  }, [content])

  if (result?.content !== content) {
    return (
      <Center p="xl" bg="var(--mantine-color-default-hover)">
        <Loader size="sm" />
      </Center>
    )
  }

  if (result.value instanceof Error) {
    return (
      <Center p="xl">
        <Alert color="red">{result.value.message}</Alert>
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
      dangerouslySetInnerHTML={{ __html: result.value }}
    />
  )
}
