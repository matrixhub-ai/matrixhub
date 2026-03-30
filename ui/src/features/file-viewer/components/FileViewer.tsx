import {
  Alert,
  Center,
  Loader,
  Paper,
} from '@mantine/core'
import {
  lazy,
  Suspense,
  useState,
} from 'react'

import { classifyFile, getMonacoLanguage } from '../utils'
import { FallbackCard } from './FallbackCard'
import { FileToolbar } from './FileToolbar'

import type { FileViewerFile } from '../types'

type ViewMode = 'preview' | 'code'

const MarkdownViewer = lazy(() =>
  import('./MarkdownViewer').then(m => ({ default: m.MarkdownViewer })),
)

const CodeViewer = lazy(() =>
  import('./CodeViewer').then(m => ({ default: m.CodeViewer })),
)

interface FileViewerProps {
  /** File metadata (name, size, url, commit, etc.) */
  file: FileViewerFile
  /** Text content — omit for binary/fallback files */
  content?: string
  /** Whether the file content is still loading */
  loading?: boolean
  /** Any error encountered while loading file content */
  error?: Error | null
}

function LazyFallback() {
  return (
    <Center p="xl" bg="var(--mantine-color-default-hover)">
      <Loader size="sm" />
    </Center>
  )
}

export function FileViewer({
  file,
  content,
  loading,
  error,
}: FileViewerProps) {
  const category = classifyFile(file.name)
  const isMarkdown = category === 'markdown'
  const isReady = !loading && !error && content != null
  const [viewMode, setViewMode] = useState<ViewMode>(isMarkdown ? 'preview' : 'code')

  return (
    <Paper
      radius="md"
      style={{ overflow: 'hidden' }}
      bg="var(--mantine-color-default-hover)"
    >
      {/* ── Toolbar: always visible ──────────────────────────── */}
      <FileToolbar
        file={file}
        isMarkdown={isMarkdown && isReady}
        viewMode={viewMode}
        onViewModeChange={setViewMode}
      />

      {/* ── Content Area ─────────────────────────────────────── */}
      {loading
        ? (
            <Center p="xl" bg="var(--mantine-color-default-hover)">
              <Loader size="sm" />
            </Center>
          )
        : error
          ? (
              <Center p="xl">
                <Alert variant="light" color="red" title="Failed to load file">
                  {error.message}
                </Alert>
              </Center>
            )
          : category === 'binary' || content == null
            ? (
                <FallbackCard file={file} />
              )
            : (
                <Suspense fallback={<LazyFallback />}>
                  {isMarkdown && viewMode === 'preview'
                    ? (
                        <MarkdownViewer content={content ?? ''} />
                      )
                    : (
                        <CodeViewer
                          content={content ?? ''}
                          language={getMonacoLanguage(file.name)}
                        />
                      )}
                </Suspense>
              )}
    </Paper>
  )
}
