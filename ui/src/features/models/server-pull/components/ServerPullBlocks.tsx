import {
  Alert,
  Box,
  Button,
  Code,
  Group,
  List,
  Progress,
  Skeleton,
  Stack,
  Text,
} from '@mantine/core'
import {
  IconAlertCircle,
  IconAlertTriangle,
  IconRefresh,
} from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import {
  computeCacheProgress,
  type ModelCacheState,
  resolveModelCacheState,
  toSafeNumber,
} from '@/features/models/cache/cache.utils'
import { getErrorMessage } from '@/queryClient'
import {
  StatusIndicator,
  type StatusIndicatorType,
} from '@/shared/components/StatusIndicator'
import { TruncatedText } from '@/shared/components/TruncatedText'
import { formatStorageSize } from '@/shared/utils/format'

import {
  classifyServerPullError,
  listFailedFiles,
  shortCommit,
} from '../server-pull.utils'
import classes from './ServerPullBlocks.module.css'

import type { ModelCacheStatus } from '@matrixhub/api-ts/v1alpha1/model.pb'
import type { ReactNode } from 'react'

const MAX_FAILED_FILES = 5

const progressIndicators: Record<ModelCacheState, {
  type: StatusIndicatorType
  labelKey: 'running' | 'succeeded' | 'failed' | 'stopped'
}> = {
  running: {
    type: 'info',
    labelKey: 'running',
  },
  cached: {
    type: 'success',
    labelKey: 'succeeded',
  },
  failed: {
    type: 'danger',
    labelKey: 'failed',
  },
  partial: {
    type: 'neutral',
    labelKey: 'stopped',
  },
  uncached: {
    type: 'neutral',
    labelKey: 'stopped',
  },
  unknown: {
    type: 'neutral',
    labelKey: 'stopped',
  },
}

function formatBytes(bytes: number) {
  return formatStorageSize(String(bytes))
}

function SummaryRow({
  label,
  children,
}: {
  label: string
  children: ReactNode
}) {
  return (
    <Group justify="space-between" align="flex-start" wrap="nowrap" gap="md">
      <Text size="sm" c="dimmed" flex="none">{label}</Text>
      <Box miw={0} ta="right">{children}</Box>
    </Group>
  )
}

interface ServerPullSummaryProps {
  /** The selected revision; empty means the default branch. */
  revision: string
  /** Name of the default branch, when known. */
  defaultBranch?: string
  /** The preview or the latest status of the pull; adds what the revision holds. */
  status?: ModelCacheStatus
  /** Shows placeholders where the status will go. */
  loading?: boolean
  /** Dimmed note at the bottom, e.g. what to do before anything can be shown. */
  hint?: ReactNode
}

/** Where a pull stores files and what it covers: the revision, its files and what is cached. */
export function ServerPullSummary({
  revision,
  defaultBranch,
  status,
  loading,
  hint,
}: ServerPullSummaryProps) {
  const { t } = useTranslation()
  const files = toSafeNumber(status?.totals?.files)
  const commit = shortCommit(status?.commit)
  const revisionLabel = revision || defaultBranch || t('model.serverPull.revision.defaultBranch')

  return (
    <Box className={classes.summary}>
      <Stack gap="xs">
        <SummaryRow label={t('model.serverPull.summary.location')}>
          <Text size="sm">{t('model.serverPull.summary.locationValue')}</Text>
        </SummaryRow>
        <SummaryRow label={t('model.serverPull.summary.scope')}>
          <Text size="sm">{t('model.serverPull.summary.scopeValue')}</Text>
        </SummaryRow>

        {status && (
          <>
            <SummaryRow label={t('model.serverPull.summary.revision')}>
              <Group gap={6} wrap="nowrap" justify="flex-end">
                <TruncatedText size="sm" miw={0} value={revisionLabel} />
                {commit && <Code>{commit}</Code>}
              </Group>
            </SummaryRow>
            <SummaryRow label={t('model.serverPull.summary.files')}>
              <Text size="sm">
                {t('model.serverPull.summary.filesValue', {
                  count: files,
                  size: formatBytes(toSafeNumber(status.totals?.bytes)),
                })}
              </Text>
            </SummaryRow>
            <SummaryRow label={t('model.serverPull.summary.cached')}>
              <Text size="sm">
                {t('model.serverPull.summary.cachedValue', {
                  cached: toSafeNumber(status.totals?.cachedFiles),
                  count: files,
                  size: formatBytes(toSafeNumber(status.totals?.cachedBytes)),
                })}
              </Text>
            </SummaryRow>
          </>
        )}

        {loading && !status && (
          <Stack gap="xs" aria-busy>
            <Skeleton h={14} />
            <Skeleton h={14} />
            <Skeleton h={14} w="60%" />
          </Stack>
        )}

        {hint && <Text size="xs" c="dimmed">{hint}</Text>}
      </Stack>
    </Box>
  )
}

/** Notes from the server about the selection, e.g. files it skipped. */
export function ServerPullWarnings({ status }: { status: ModelCacheStatus }) {
  const { t } = useTranslation()
  const warnings = [...new Set(status.warnings ?? [])]

  if (warnings.length === 0) {
    return null
  }

  return (
    <Alert
      color="yellow"
      variant="light"
      icon={<IconAlertTriangle size={18} />}
      title={t('model.serverPull.summary.warnings')}
    >
      <List size="xs" spacing={4}>
        {warnings.map(warning => <List.Item key={warning}>{warning}</List.Item>)}
      </List>
    </Alert>
  )
}

interface ServerPullProgressProps {
  status: ModelCacheStatus
  /** The status can no longer be read, so the last progress stays still. */
  frozen?: boolean
}

/** Task status, percentage and file / byte counts of a pull. */
export function ServerPullProgress({
  status,
  frozen = false,
}: ServerPullProgressProps) {
  const { t } = useTranslation()
  const state = resolveModelCacheState(status)
  const progress = computeCacheProgress(status)
  const indicator = progressIndicators[state]
  const animated = state === 'running' && !frozen
  let barColor: string | undefined

  if (state === 'failed') {
    barColor = 'red'
  } else if (progress.complete) {
    barColor = 'green'
  } else if (frozen) {
    barColor = 'gray'
  }

  return (
    <Stack gap={6}>
      <Group justify="space-between" wrap="nowrap" gap="sm">
        <StatusIndicator
          type={frozen ? 'neutral' : indicator.type}
          label={t(`model.serverPull.progress.${frozen ? 'frozen' : indicator.labelKey}`)}
        />
        <Text size="sm" fw={600}>{`${progress.percent}%`}</Text>
      </Group>
      <Progress
        value={progress.percent}
        color={barColor}
        striped={animated}
        animated={animated}
        aria-label={t('model.serverPull.progress.label')}
      />
      <Group justify="space-between" gap="xs">
        <Text size="xs" c="dimmed">
          {t('model.serverPull.progress.files', {
            done: progress.doneFiles,
            count: progress.totalFiles,
          })}
        </Text>
        <Text size="xs" c="dimmed">
          {t('model.serverPull.progress.bytes', {
            done: formatBytes(progress.doneBytes),
            total: formatBytes(progress.totalBytes),
          })}
        </Text>
      </Group>
    </Stack>
  )
}

/**
 * Why a pull failed: the first failed files with their errors, or the task error when no file
 * failed on its own, e.g. when the server stopped.
 */
export function ServerPullFailureDetails({ status }: { status: ModelCacheStatus }) {
  const { t } = useTranslation()
  const {
    items,
    rest,
  } = listFailedFiles(status, MAX_FAILED_FILES)
  const taskError = status.task?.error?.trim()

  return (
    <Alert
      color="red"
      variant="light"
      icon={<IconAlertCircle size={18} />}
      title={t('model.serverPull.failedTitle')}
    >
      <Stack gap="xs">
        {taskError && items.length === 0 && (
          <Text size="xs" style={{ overflowWrap: 'anywhere' }}>{taskError}</Text>
        )}
        {items.length > 0 && (
          <List size="xs" spacing={6}>
            {items.map(file => (
              <List.Item key={file.path}>
                <Text size="xs" ff="monospace" style={{ overflowWrap: 'anywhere' }}>{file.path}</Text>
                {file.error && (
                  <Text size="xs" c="dimmed" style={{ overflowWrap: 'anywhere' }}>{file.error}</Text>
                )}
              </List.Item>
            ))}
          </List>
        )}
        {rest > 0 && (
          <Text size="xs" c="dimmed">
            {t('model.serverPull.moreFailed', { count: rest })}
          </Text>
        )}
      </Stack>
    </Alert>
  )
}

interface ServerPullErrorAlertProps {
  error: unknown
  /** `pollErrors` when the progress of a running pull can no longer be read. */
  titles?: 'errors' | 'pollErrors'
  retrying?: boolean
  /** Renders a button that repeats the failed request. */
  onRetry?: () => void
}

/** Inline error of a preview or status request, titled by the kind of failure. */
export function ServerPullErrorAlert({
  error,
  titles = 'errors',
  retrying,
  onRetry,
}: ServerPullErrorAlertProps) {
  const { t } = useTranslation()
  const kind = classifyServerPullError(error)
  const detail = getErrorMessage(error)

  return (
    <Alert
      color="red"
      variant="light"
      icon={<IconAlertCircle size={18} />}
      title={t(`model.serverPull.${titles}.${kind}`)}
    >
      <Stack gap="xs" align="flex-start">
        {detail && (
          <Text size="xs" c="dimmed" style={{ overflowWrap: 'anywhere' }}>{detail}</Text>
        )}
        {onRetry && (
          <Button
            size="xs"
            variant="light"
            leftSection={<IconRefresh size={14} />}
            loading={retrying}
            onClick={onRetry}
          >
            {t('model.serverPull.actions.recheck')}
          </Button>
        )}
      </Stack>
    </Alert>
  )
}
