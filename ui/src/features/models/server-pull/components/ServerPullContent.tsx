import {
  Alert,
  Stack,
  Text,
} from '@mantine/core'
import { IconCircleCheck } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import {
  ServerPullErrorAlert,
  ServerPullFailureDetails,
  ServerPullProgress,
  ServerPullSummary,
  ServerPullWarnings,
} from './ServerPullBlocks'

import type { ServerPull } from '../hooks/useServerPull'
import type { ServerPullView } from '../server-pull.utils'
import type { ModelCacheStatus } from '@matrixhub/api-ts/v1alpha1/model.pb'
import type { ReactNode } from 'react'

function viewStatus(view: ServerPullView): ModelCacheStatus | undefined {
  switch (view.kind) {
    case 'ready':
    case 'empty':
    case 'cached':
    case 'tracking':
      return view.status
    default:
      return undefined
  }
}

function SuccessAlert({ children }: { children: ReactNode }) {
  return (
    <Alert color="green" variant="light" icon={<IconCircleCheck size={18} />}>
      <Text size="sm">{children}</Text>
    </Alert>
  )
}

interface ServerPullContentProps {
  pull: ServerPull
  /** The selected revision; empty means the default branch. */
  revision: string
  /** Shown in the summary while there is no model name to preview. */
  idleHint?: ReactNode
}

/**
 * The server pull part of a download drawer: the summary of what is pulled, then the preview
 * result, the progress, or what went wrong. The footer actions come from `ServerPullActions`.
 */
export function ServerPullContent({
  pull,
  revision,
  idleHint,
}: ServerPullContentProps) {
  const { t } = useTranslation()
  const { view } = pull
  const status = viewStatus(view)

  return (
    <Stack gap="sm">
      <ServerPullSummary
        revision={revision}
        defaultBranch={pull.defaultBranch}
        status={status}
        loading={view.kind === 'checking' || view.kind === 'trackingPending'}
        hint={view.kind === 'idle' ? idleHint : undefined}
      />
      {status && <ServerPullWarnings status={status} />}

      {view.kind === 'checkFailed' && (
        <ServerPullErrorAlert
          error={view.error}
          retrying={pull.rechecking}
          onRetry={pull.recheck}
        />
      )}
      {view.kind === 'empty' && (
        <Text size="sm" c="dimmed">{t('model.serverPull.states.nothingToPull')}</Text>
      )}
      {view.kind === 'cached' && (
        <SuccessAlert>{t('model.serverPull.states.alreadyCached')}</SuccessAlert>
      )}
      {/* The last pull of this revision failed; starting again retries the missing files. */}
      {view.kind === 'ready' && view.state === 'failed' && (
        <ServerPullFailureDetails status={view.status} />
      )}

      {view.kind === 'tracking' && (
        <>
          {view.pollError !== undefined && (
            <ServerPullErrorAlert error={view.pollError} titles="pollErrors" />
          )}
          <ServerPullProgress status={view.status} frozen={view.pollError !== undefined} />
          {view.state === 'failed' && <ServerPullFailureDetails status={view.status} />}
          {view.state === 'cached' && (
            <SuccessAlert>{t('model.serverPull.states.succeeded')}</SuccessAlert>
          )}
        </>
      )}
      {view.kind === 'trackingFailed' && (
        <ServerPullErrorAlert
          error={view.error}
          titles="pollErrors"
          retrying={pull.reloading}
          onRetry={pull.reload}
        />
      )}
    </Stack>
  )
}
