import {
  Alert,
  Box,
  Button,
  Code,
  Group,
  List,
  Paper,
  Progress,
  Stack,
  Text,
} from '@mantine/core'
import {
  IconAlertCircle,
  IconAlertTriangle,
  IconArrowRight,
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
import { AnchorLink } from '@/shared/components/AnchorLink'
import {
  StatusIndicator,
  type StatusIndicatorType,
} from '@/shared/components/StatusIndicator'
import { formatStorageSize } from '@/shared/utils/format'

import {
  classifyServerPullError,
  listFailedFiles,
  shortCommit,
} from '../server-pull.utils'

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
    <Group justify="space-between" align="flex-start" wrap="nowrap" gap="sm">
      <Text size="sm" c="dimmed" flex="none">{label}</Text>
      <Box miw={0} ta="right">{children}</Box>
    </Group>
  )
}

/** What a pull covers: the resolved revision, its files and what is already cached. */
export function ServerPullSummary({ status }: { status: ModelCacheStatus }) {
  const { t } = useTranslation()
  const files = toSafeNumber(status.totals?.files)
  const commit = shortCommit(status.commit)
  const warnings = [...new Set(status.warnings ?? [])]

  return (
    <Stack gap="xs">
      <Paper withBorder radius="sm" p="sm">
        <Stack gap={6}>
          <SummaryRow label={t('projects.detail.proxyDownload.serverPull.summary.revision')}>
            <Group gap={6} wrap="nowrap" justify="flex-end">
              <Text size="sm" miw={0} truncate>
                {status.requestedRevision || t('projects.detail.proxyDownload.serverPull.summary.defaultBranch')}
              </Text>
              {commit && <Code>{commit}</Code>}
            </Group>
          </SummaryRow>
          <SummaryRow label={t('projects.detail.proxyDownload.serverPull.summary.files')}>
            <Text size="sm">
              {t('projects.detail.proxyDownload.serverPull.summary.filesValue', {
                files,
                size: formatBytes(toSafeNumber(status.totals?.bytes)),
              })}
            </Text>
          </SummaryRow>
          <SummaryRow label={t('projects.detail.proxyDownload.serverPull.summary.cached')}>
            <Text size="sm">
              {t('projects.detail.proxyDownload.serverPull.summary.cachedValue', {
                cached: toSafeNumber(status.totals?.cachedFiles),
                files,
                size: formatBytes(toSafeNumber(status.totals?.cachedBytes)),
              })}
            </Text>
          </SummaryRow>
        </Stack>
      </Paper>

      {warnings.length > 0 && (
        <Alert
          color="yellow"
          variant="light"
          icon={<IconAlertTriangle size={18} />}
          title={t('projects.detail.proxyDownload.serverPull.summary.warnings')}
        >
          <List size="xs" spacing={4}>
            {warnings.map(warning => <List.Item key={warning}>{warning}</List.Item>)}
          </List>
        </Alert>
      )}
    </Stack>
  )
}

/** Task status, percentage and file / byte counts of a pull. */
export function ServerPullProgress({ status }: { status: ModelCacheStatus }) {
  const { t } = useTranslation()
  const state = resolveModelCacheState(status)
  const progress = computeCacheProgress(status)
  const indicator = progressIndicators[state]
  const running = state === 'running'
  let barColor: string | undefined

  if (state === 'failed') {
    barColor = 'red'
  } else if (progress.complete) {
    barColor = 'green'
  }

  return (
    <Stack gap={6}>
      <Group justify="space-between" wrap="nowrap" gap="sm">
        <StatusIndicator
          type={indicator.type}
          label={t(`projects.detail.proxyDownload.serverPull.progress.${indicator.labelKey}`)}
        />
        <Text size="sm" fw={600}>{`${progress.percent}%`}</Text>
      </Group>
      <Progress
        value={progress.percent}
        color={barColor}
        striped={running}
        animated={running}
        aria-label={t('projects.detail.proxyDownload.serverPull.progress.label')}
      />
      <Group justify="space-between" gap="xs">
        <Text size="xs" c="dimmed">
          {t('projects.detail.proxyDownload.serverPull.progress.files', {
            done: progress.doneFiles,
            total: progress.totalFiles,
          })}
        </Text>
        <Text size="xs" c="dimmed">
          {t('projects.detail.proxyDownload.serverPull.progress.bytes', {
            done: formatBytes(progress.doneBytes),
            total: formatBytes(progress.totalBytes),
          })}
        </Text>
      </Group>
      {running && (
        <Text size="xs" c="dimmed">
          {t('projects.detail.proxyDownload.serverPull.backgroundHint')}
        </Text>
      )}
    </Stack>
  )
}

interface ServerPullFailureDetailsProps {
  status: ModelCacheStatus
  retrying?: boolean
  /** Renders a retry button when set. */
  onRetry?: () => void
}

/** Why a pull failed: the task error and the first failed files. */
export function ServerPullFailureDetails({
  status,
  retrying,
  onRetry,
}: ServerPullFailureDetailsProps) {
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
      title={t('projects.detail.proxyDownload.serverPull.failedTitle')}
    >
      <Stack gap="xs" align="flex-start">
        {taskError && (
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
            {t('projects.detail.proxyDownload.serverPull.moreFailed', { rest })}
          </Text>
        )}
        {onRetry && (
          <Button
            size="xs"
            variant="light"
            leftSection={<IconRefresh size={14} />}
            loading={retrying}
            onClick={onRetry}
          >
            {t('projects.detail.proxyDownload.serverPull.retry')}
          </Button>
        )}
      </Stack>
    </Alert>
  )
}

interface ServerPullErrorAlertProps {
  error: unknown
  retrying: boolean
  onRetry: () => void
}

/** Inline error of a preview or status request, titled by the kind of failure. */
export function ServerPullErrorAlert({
  error,
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
      title={t(`projects.detail.proxyDownload.serverPull.errors.${kind}`)}
    >
      <Stack gap="xs" align="flex-start">
        {detail && (
          <Text size="xs" c="dimmed" style={{ overflowWrap: 'anywhere' }}>{detail}</Text>
        )}
        <Button
          size="xs"
          variant="light"
          leftSection={<IconRefresh size={14} />}
          loading={retrying}
          onClick={onRetry}
        >
          {t('projects.detail.proxyDownload.serverPull.retryCheck')}
        </Button>
      </Stack>
    </Alert>
  )
}

export function ServerPullModelLink({
  projectId,
  modelName,
}: {
  projectId: string
  modelName: string
}) {
  const { t } = useTranslation()

  return (
    <AnchorLink
      to="/projects/$projectId/models/$modelId"
      params={{
        projectId,
        modelId: modelName,
      }}
      size="sm"
      fw={500}
    >
      <Group component="span" gap={4} wrap="nowrap" display="inline-flex">
        {t('projects.detail.proxyDownload.serverPull.viewModel')}
        <IconArrowRight size={14} />
      </Group>
    </AnchorLink>
  )
}
