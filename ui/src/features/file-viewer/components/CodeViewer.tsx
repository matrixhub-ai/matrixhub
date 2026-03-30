import {
  Box, Center, Loader,
} from '@mantine/core'
import Editor from '@monaco-editor/react'

import classes from './FileViewer.module.css'

interface CodeViewerProps {
  content: string
  language: string
}

export function CodeViewer({
  content, language,
}: CodeViewerProps) {
  const lineCount = content.split('\n').length

  return (
    <Box className={classes.codeRoot}>
      <Editor
        height={`${Math.min(Math.max(lineCount * 20, 200), 800)}px`}
        language={language}
        value={content}
        loading={(
          <Center p="xl">
            <Loader size="sm" />
          </Center>
        )}
        options={{
          readOnly: true,
          scrollbar: {
            alwaysConsumeMouseWheel: false,
          },
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          lineNumbers: 'on',
          renderLineHighlight: 'none',
          fontSize: 13,
          fontFamily: 'monospace',
          overviewRulerLanes: 0,
          hideCursorInOverviewRuler: true,
          overviewRulerBorder: false,
          contextmenu: false,
          domReadOnly: true,
          folding: true,
          wordWrap: 'on',
        }}
      />
    </Box>
  )
}
