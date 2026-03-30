import { Box } from '@mantine/core'
import Markdown from 'react-markdown'

import classes from './FileViewer.module.css'

interface MarkdownViewerProps {
  content: string
}

export function MarkdownViewer({ content }: MarkdownViewerProps) {
  return (
    <Box className={classes.markdownRoot} p="md" bg="var(--mantine-color-default-hover)">
      <Markdown>{content}</Markdown>
    </Box>
  )
}
