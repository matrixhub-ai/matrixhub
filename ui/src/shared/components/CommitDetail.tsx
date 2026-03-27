import {
  Box,
  Flex,
  Group,
  Paper,
  rem,
  Stack,
  Text,
  Tooltip,
} from '@mantine/core'
import { IconCopy } from '@tabler/icons-react'
import dayjs from 'dayjs'
import { html as formatDiffHtml } from 'diff2html'
import { useTranslation } from 'react-i18next'

import { CopyValueButton } from '@/shared/components/CopyValueButton'
import { parseConventionalCommit } from '@/shared/utils'
import { formatDateTime } from '@/shared/utils/date'

import 'diff2html/bundles/css/diff2html.min.css'

import type { Commit } from '@matrixhub/api-ts/v1alpha1/model.pb'

interface CommitDetailProps {
  commit?: Commit
}

interface CommitHashCardProps {
  commitId?: string
}

interface CommitMessageSectionProps {
  message?: string
}

function CommitHashCard({ commitId }: CommitHashCardProps) {
  const shownCommitId = commitId ?? ''
  const shortCommitId = shownCommitId.slice(0, 7)

  if (!shortCommitId) {
    return null
  }

  return (
    <Group gap={8} wrap="nowrap">
      <Text fw={600} size="sm">Commit</Text>
      <Box
        bdrs="sm"
        bg="#fff"
        style={{
          border: '1px solid var(--mantine-color-gray-3)',
        }}
      >
        <Flex align="center">
          <Text
            py={3}
            pl={12}
            pr={6}
            size="xs"
            c="gray.5"
            lh={rem(16)}
            style={{ borderRight: '1px solid var(--mantine-color-gray-3)' }}
          >
            {shortCommitId}
          </Text>
          <CopyValueButton value={shownCommitId}>
            {({ copy }) => {
              return (
                <Box onClick={copy} px={8} style={{ cursor: 'pointer' }} lh={1}>
                  <IconCopy size={12} style={{ color: 'var(--mantine-color-gray-6)' }} />
                </Box>
              )
            }}
          </CopyValueButton>
        </Flex>
      </Box>
    </Group>
  )
}

function CommitMessageSection({ message }: CommitMessageSectionProps) {
  // TODO: some npm packages like `conventional-commits-parser` can parse the commit message into structured data,
  //  But it runs in Node environment and cannot be directly used in the browser. We may need to find a similar package that supports browser environment
  const parsedMessage = parseConventionalCommit(message ?? '')

  return (
    <Stack p="sm" gap="xs">
      <Text fw={600} size="lg" lh={rem(20)}>{parsedMessage?.header || '-'}</Text>
      {parsedMessage.body ? <Text size="sm" lh={rem(20)}>{parsedMessage.body}</Text> : null}
      {parsedMessage.footer ? <Text size="sm" lh={rem(20)}>{parsedMessage.footer}</Text> : null}
    </Stack>
  )
}

export function CommitDetail({
  commit,
}: CommitDetailProps) {
  const {
    t,
  } = useTranslation()

  const authorDate = formatDateTime(commit?.authorDate, 'YYYY-MM-DD HH:mm:ss')

  const diffText = commit?.diff ?? ''
  const diffHtml = diffText
    ? formatDiffHtml(diffText, {
        drawFileList: true,
        matching: 'lines',
        outputFormat: 'side-by-side',
      })
    : ''

  return (
    <Stack gap="sm">
      <Paper withBorder radius="md" bg="#F8F9FA">
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
        diffHtml
          ? (
              <Box
                className="model-commit-diff"
                // diff2html returns escaped markup intended for direct rendering.
                // eslint-disable-next-line @eslint-react/dom/no-dangerously-set-innerhtml
                dangerouslySetInnerHTML={{ __html: diffHtml }}
              />
            )
          : <Text size="sm" c="dimmed">{t('shared.commitDetail.emptyDiff')}</Text>
      }
    </Stack>
  )
}
