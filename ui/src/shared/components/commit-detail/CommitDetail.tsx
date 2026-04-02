import {
  Box,
  Group,
  Loader,
  Paper,
  Stack,
  Text,
  Tooltip,
} from '@mantine/core'
import dayjs from 'dayjs'
import { html as formatDiffHtml, parse } from 'diff2html'
import {
  type MouseEvent,
  useEffect,
  useState,
} from 'react'
import { useTranslation } from 'react-i18next'

import { CommitHashCard } from '@/shared/components/commit-detail/CommitHashCard'
import { CommitMessageSection } from '@/shared/components/commit-detail/CommitMessageSection'
import { DiffLimitAlert } from '@/shared/components/commit-detail/DiffLimitAlert'
import { formatDateTime } from '@/shared/utils/date'

import 'diff2html/bundles/css/diff2html.min.css'
import './CommitDetail.css'

import type { Commit } from '@matrixhub/api-ts/v1alpha1/model.pb'

interface CommitDetailProps {
  commit?: Commit
}

const MAX_FILES = 50
const DIFF_PARSE_DELAY_MS = 30
const RAW_DIFF_URL_REVOKE_DELAY_MS = 60_000

interface DiffRenderState {
  html: string
  hasLimit: boolean
  loading: boolean
}

const EMPTY_DIFF_RENDER_STATE: DiffRenderState = {
  html: '',
  hasLimit: false,
  loading: false,
}

function EmptyDiffPrompt() {
  const { t } = useTranslation()

  return (
    <Text size="sm" c="dimmed">
      {t('shared.commitDetail.emptyDiff')}
    </Text>
  )
}

export function CommitDetail({ commit }: CommitDetailProps) {
  const { t } = useTranslation()
  const hasRawDiff = Boolean(commit?.diff)
  const authorDate = formatDateTime(commit?.authorDate, 'YYYY-MM-DD HH:mm:ss')

  const [diffRender, setDiffRender] = useState<DiffRenderState>(EMPTY_DIFF_RENDER_STATE)

  useEffect(() => {
    const rawDiff = commit?.diff ?? ''
    let cancelled = false

    if (!rawDiff) {
      const resetTimerId = window.setTimeout(() => {
        if (!cancelled) {
          setDiffRender(EMPTY_DIFF_RENDER_STATE)
        }
      }, 0)

      return () => {
        cancelled = true
        window.clearTimeout(resetTimerId)
      }
    }

    const loadingTimerId = window.setTimeout(() => {
      if (cancelled) {
        return
      }

      setDiffRender(prev => ({
        ...prev,
        loading: true,
      }))
    }, 0)

    const parseTimerId = window.setTimeout(() => {
      if (cancelled) {
        return
      }

      const diffJson = parse(rawDiff)
      const limitedDiffJson = diffJson.slice(0, MAX_FILES)
      const nextDiffHtml = limitedDiffJson.length > 0
        ? formatDiffHtml(limitedDiffJson, {
            drawFileList: true,
            matching: 'lines',
            outputFormat: 'side-by-side',
          })
        : ''

      if (!cancelled) {
        setDiffRender({
          html: nextDiffHtml,
          hasLimit: diffJson.length > MAX_FILES,
          loading: false,
        })
      }
    }, DIFF_PARSE_DELAY_MS)

    return () => {
      cancelled = true
      window.clearTimeout(loadingTimerId)
      window.clearTimeout(parseTimerId)
    }
  }, [commit?.diff])

  const handleOpenRawDiff = (event: MouseEvent<HTMLAnchorElement>) => {
    event.preventDefault()

    const rawDiffUrl = URL.createObjectURL(
      new Blob([commit?.diff ?? ''], { type: 'text/plain;charset=utf-8' }),
    )

    window.open(rawDiffUrl, '_blank', 'noopener,noreferrer')
    setTimeout(() => URL.revokeObjectURL(rawDiffUrl), RAW_DIFF_URL_REVOKE_DELAY_MS)
  }

  return (
    <Stack gap="sm">
      <Paper withBorder radius="md" bg="gray.0">
        <Stack gap="0">
          <Group
            justify="space-between"
            wrap="nowrap"
            align="center"
            py="6"
            px="sm"
            style={{
              borderBottom: '1px solid var(--mantine-color-gray-3)',
            }}
          >
            <Stack gap={2}>
              <Group gap="xs" align="center">
                <Text size="sm" component="span">{commit?.authorName || '-'}</Text>
                <Text size="sm" c="dimmed" component="span">
                  <span>
                    {t('shared.commitDetail.committedOn')}
                  </span>
                  <Tooltip label={authorDate} withArrow>
                    <span>{authorDate === '-' ? '-' : dayjs(authorDate).fromNow()}</span>
                  </Tooltip>
                </Text>
              </Group>
            </Stack>

            <CommitHashCard commitId={commit?.id} />
          </Group>

          <CommitMessageSection message={commit?.message} />
        </Stack>
      </Paper>

      {
        hasRawDiff
          ? diffRender.loading
            ? (
                <Group gap="xs">
                  <Loader size="sm" />
                  <Text size="sm" c="dimmed">{t('shared.commitDetail.loadingDiff')}</Text>
                </Group>
              )
            : (
                <>
                  {diffRender.hasLimit
                    ? (
                        <DiffLimitAlert
                          maxFiles={MAX_FILES}
                          onOpenRawDiff={handleOpenRawDiff}
                        />
                      )
                    : null}

                  {
                    diffRender.html
                      ? (
                          <Box
                            pos="relative"
                            // diff2html returns escaped markup intended for direct rendering.
                            // eslint-disable-next-line @eslint-react/dom/no-dangerously-set-innerhtml
                            dangerouslySetInnerHTML={{ __html: diffRender.html }}
                          />
                        )
                      : <EmptyDiffPrompt />
                  }
                </>
              )
          : <EmptyDiffPrompt />
      }
    </Stack>
  )
}
