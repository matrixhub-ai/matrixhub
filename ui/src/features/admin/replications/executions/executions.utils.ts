import {
  SyncTaskStatus,
  type SyncTask,
} from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'

import i18n from '@/i18n'
import { toMilliseconds } from '@/shared/utils/date'

const SECOND_IN_MS = 1000
const MINUTE_IN_MS = 60 * SECOND_IN_MS
const HOUR_IN_MS = 60 * MINUTE_IN_MS

export function getReplicationExecutionRowId(task: SyncTask) {
  return String(task.id ?? `${task.startedTimestamp ?? '0'}-${task.status ?? 'unknown'}`)
}

export function parseReplicationId(replicationId: string) {
  const value = Number(replicationId)

  if (!Number.isInteger(value) || value < 1) {
    throw new Error(i18n.t('routes.admin.replications.executions.errors.invalidSyncPolicyId'))
  }

  return value
}

export function formatReplicationExecutionDuration(task: SyncTask) {
  const startedAt = task.startedTimestamp == null
    ? undefined
    : toMilliseconds(task.startedTimestamp)
  const completedAt = task.completedTimestamp == null
    ? undefined
    : toMilliseconds(task.completedTimestamp)
  const endedAt = completedAt ?? (
    task.status === SyncTaskStatus.SYNC_TASK_STATUS_RUNNING
      ? Date.now()
      : undefined
  )

  if (startedAt == null || endedAt == null || endedAt < startedAt) {
    return '-'
  }

  const durationMs = Math.max(0, endedAt - startedAt)

  if (durationMs < SECOND_IN_MS) {
    return `${Math.round(durationMs)} ms`
  }

  if (durationMs < MINUTE_IN_MS) {
    const seconds = durationMs / SECOND_IN_MS

    return Number.isInteger(seconds) ? `${seconds} s` : `${seconds.toFixed(1)} s`
  }

  if (durationMs < HOUR_IN_MS) {
    const minutes = Math.floor(durationMs / MINUTE_IN_MS)
    const seconds = Math.floor((durationMs % MINUTE_IN_MS) / SECOND_IN_MS)

    return seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`
  }

  const hours = Math.floor(durationMs / HOUR_IN_MS)
  const minutes = Math.floor((durationMs % HOUR_IN_MS) / MINUTE_IN_MS)

  return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`
}
