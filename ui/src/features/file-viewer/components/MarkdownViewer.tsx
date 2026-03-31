import { Box } from '@mantine/core'
import Markdown from 'react-markdown'

import 'github-markdown-css/github-markdown.css'

interface MarkdownViewerProps {
  content: string
}

export function MarkdownViewer({ content }: MarkdownViewerProps) {
  return (
    <Box className="markdown-body" p="md" bg="var(--mantine-color-default-hover)" bdrs="md">
      <Markdown>{content}</Markdown>
    </Box>
  )
}
