import {
  SyncTaskStatus,
  type SyncTask,
} from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import dayjs from 'dayjs'

import i18n from '@/i18n'
import { toMilliseconds } from '@/shared/utils/date'

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

  if (startedAt == null || endedAt == null || endedAt <= startedAt) {
    return '-'
  }

  return dayjs(endedAt).from(startedAt, true)
}
