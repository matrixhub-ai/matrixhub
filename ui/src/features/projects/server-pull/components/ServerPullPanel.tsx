import {
  Alert,
  Button,
  Group,
  Skeleton,
  Stack,
  Text,
  TextInput,
} from '@mantine/core'
import { useDebouncedValue } from '@mantine/hooks'
import {
  IconCircleCheck,
  IconCloudDownload,
} from '@tabler/icons-react'
import { useStore } from '@tanstack/react-form'
import {
  useMutation,
  useQuery,
} from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  type ModelCacheSelection,
  modelCachePreviewQueryOptions,
} from '@/features/models/cache/cache.query'
import {
  isTaskRunning,
  resolveModelCacheState,
  toSafeNumber,
} from '@/features/models/cache/cache.utils'
import { useForm } from '@/shared/hooks/useForm'

import { useModelCacheStatus } from '../hooks/useModelCacheStatus'
import {
  type CacheModelInput,
  cacheModelMutationOptions,
} from '../server-pull.mutation'
import { getServerPullActionKey } from '../server-pull.utils'
import {
  ServerPullErrorAlert,
  ServerPullFailureDetails,
  ServerPullModelLink,
  ServerPullProgress,
  ServerPullSummary,
} from './ServerPullPanels'

import type { ModelCacheStatus } from '@matrixhub/api-ts/v1alpha1/model.pb'
import type { ReactNode } from 'react'

const INPUT_DEBOUNCE_MS = 400
const LOADING_HEIGHT = 96

function SucceededAlert({
  projectId,
  modelName,
  message,
}: {
  projectId: string
  modelName: string
  message: string
}) {
  return (
    <Alert color="green" variant="light" icon={<IconCircleCheck size={18} />}>
      <Stack gap={4} align="flex-start">
        <Text size="sm">{message}</Text>
        <ServerPullModelLink projectId={projectId} modelName={modelName} />
      </Stack>
    </Alert>
  )
}

interface ServerPullOverviewProps {
  projectId: string
  modelName: string
  status: ModelCacheStatus
  starting: boolean
  refreshing: boolean
  onStart: () => void
}

function ServerPullOverview({
  projectId,
  modelName,
  status,
  starting,
  refreshing,
  onStart,
}: ServerPullOverviewProps) {
  const { t } = useTranslation()
  const state = resolveModelCacheState(status)
  const actionKey = getServerPullActionKey(state)
  const hasFiles = toSafeNumber(status.totals?.files) > 0

  return (
    <Stack gap="sm">
      <ServerPullSummary status={status} />

      {state === 'cached' && (
        <SucceededAlert
          projectId={projectId}
          modelName={modelName}
          message={t('projects.detail.proxyDownload.serverPull.alreadyCached')}
        />
      )}
      {!hasFiles && (
        <Text size="sm" c="dimmed">
          {t('projects.detail.proxyDownload.serverPull.nothingToPull')}
        </Text>
      )}
      {state === 'failed' && <ServerPullFailureDetails status={status} />}

      <Group>
        <Button
          leftSection={<IconCloudDownload size={16} />}
          loading={starting}
          disabled={!actionKey || !hasFiles || refreshing}
          onClick={onStart}
        >
          {t(`projects.detail.proxyDownload.serverPull.${actionKey ?? 'start'}`)}
        </Button>
      </Group>
    </Stack>
  )
}

interface ServerPullTrackerProps {
  projectId: string
  modelName: string
  selection: ModelCacheSelection
  initialStatus?: ModelCacheStatus
  retrying: boolean
  onRetry: () => void
}

function ServerPullTracker({
  projectId,
  modelName,
  selection,
  initialStatus,
  retrying,
  onRetry,
}: ServerPullTrackerProps) {
  const { t } = useTranslation()
  const statusQuery = useModelCacheStatus(projectId, modelName, selection, {
    placeholderData: initialStatus,
  })
  const status = statusQuery.data

  if (!status) {
    return statusQuery.isError
      ? (
          <ServerPullErrorAlert
            error={statusQuery.error}
            retrying={statusQuery.isFetching}
            onRetry={() => void statusQuery.refetch()}
          />
        )
      : <Skeleton h={LOADING_HEIGHT} />
  }

  const state = resolveModelCacheState(status)
  const actionKey = getServerPullActionKey(state)

  return (
    <Stack gap="sm">
      <ServerPullProgress status={status} />

      {state === 'cached' && (
        <SucceededAlert
          projectId={projectId}
          modelName={modelName}
          message={t('projects.detail.proxyDownload.serverPull.succeeded')}
        />
      )}
      {state === 'failed' && (
        <ServerPullFailureDetails
          status={status}
          retrying={retrying}
          onRetry={onRetry}
        />
      )}
      {/* The task stopped without finishing, e.g. the server restarted. */}
      {(state === 'partial' || state === 'uncached') && actionKey && (
        <Group>
          <Button
            leftSection={<IconCloudDownload size={16} />}
            loading={retrying}
            onClick={onRetry}
          >
            {t(`projects.detail.proxyDownload.serverPull.${actionKey}`)}
          </Button>
        </Group>
      )}
    </Stack>
  )
}

interface ServerPullSelectionViewProps {
  projectId: string
  modelName: string
  selection: ModelCacheSelection
}

/** Preview, start and progress of the pull of one model revision. */
function ServerPullSelectionView({
  projectId,
  modelName,
  selection,
}: ServerPullSelectionViewProps) {
  const [started, setStarted] = useState(false)
  const preview = useQuery({
    ...modelCachePreviewQueryOptions(projectId, modelName, selection),
    // The preview contacts upstream; once the pull runs, progress comes from the status query.
    enabled: !started,
  })
  const mutation = useMutation(cacheModelMutationOptions())

  // Pull the commit the preview resolved, so the pulled files match the preview.
  const previewInput: CacheModelInput = {
    project: projectId,
    name: modelName,
    revision: preview.data?.commit || selection.revision,
  }
  const startPull = (input: CacheModelInput) => {
    mutation.mutate(input, {
      onSuccess: () => setStarted(true),
    })
  }

  // Track the pull once started here, or attach to one already running (e.g. after a reload).
  if (started || isTaskRunning(preview.data)) {
    const trackedInput = mutation.variables ?? previewInput

    return (
      <ServerPullTracker
        projectId={projectId}
        modelName={modelName}
        selection={{ revision: trackedInput.revision }}
        initialStatus={mutation.data ?? preview.data}
        retrying={mutation.isPending}
        onRetry={() => startPull(trackedInput)}
      />
    )
  }

  if (preview.isPending) {
    return <Skeleton h={LOADING_HEIGHT} />
  }

  if (preview.isError) {
    return (
      <ServerPullErrorAlert
        error={preview.error}
        retrying={preview.isFetching}
        onRetry={() => void preview.refetch()}
      />
    )
  }

  return (
    <ServerPullOverview
      projectId={projectId}
      modelName={modelName}
      status={preview.data}
      starting={mutation.isPending}
      refreshing={preview.isFetching}
      onStart={() => startPull(previewInput)}
    />
  )
}

interface ServerPullPanelProps {
  projectId: string
  /** Upstream model name; empty while the name input is empty or invalid. */
  modelName: string
}

/**
 * Lets MatrixHub pull an upstream model into the proxy project on the server: previews the
 * files of the selected revision (dry run), starts the pull and shows its progress.
 */
export function ServerPullPanel({
  projectId,
  modelName,
}: ServerPullPanelProps) {
  const { t } = useTranslation()
  const form = useForm({
    defaultValues: {
      revision: '',
    },
  })
  const revision = useStore(form.store, state => state.values.revision.trim())
  // The preview contacts upstream, so wait until typing pauses.
  const [debouncedName] = useDebouncedValue(modelName, INPUT_DEBOUNCE_MS)
  const [debouncedRevision] = useDebouncedValue(revision, INPUT_DEBOUNCE_MS)
  const settling = modelName !== debouncedName || revision !== debouncedRevision

  let content: ReactNode

  if (!modelName) {
    content = (
      <Text size="sm" c="dimmed">
        {t('projects.detail.proxyDownload.serverPull.enterName')}
      </Text>
    )
  } else if (settling) {
    content = <Skeleton h={LOADING_HEIGHT} />
  } else {
    // Keyed by the selection so each one starts from its own preview and pull state.
    content = (
      <ServerPullSelectionView
        key={JSON.stringify([debouncedName, debouncedRevision])}
        projectId={projectId}
        modelName={debouncedName}
        selection={{ revision: debouncedRevision }}
      />
    )
  }

  return (
    <Stack gap="sm">
      <form.Field name="revision">
        {field => (
          <TextInput
            label={t('projects.detail.proxyDownload.serverPull.revision')}
            placeholder={t('projects.detail.proxyDownload.serverPull.revisionPlaceholder')}
            value={field.state.value}
            onChange={event => field.handleChange(event.currentTarget.value)}
            onBlur={field.handleBlur}
          />
        )}
      </form.Field>

      {content}
    </Stack>
  )
}
