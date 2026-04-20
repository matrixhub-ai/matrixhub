import { SyncTaskStatus } from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import { z } from 'zod'

import { DEFAULT_PAGE } from '@/utils/constants'

export const replicationExecutionStatusFilterValues = [
  SyncTaskStatus.SYNC_TASK_STATUS_RUNNING,
  SyncTaskStatus.SYNC_TASK_STATUS_STOPPED,
  SyncTaskStatus.SYNC_TASK_STATUS_SUCCEEDED,
  SyncTaskStatus.SYNC_TASK_STATUS_FAILED,
] as const

export type ReplicationExecutionStatusFilterValue
  = typeof replicationExecutionStatusFilterValues[number]

export const replicationExecutionsSearchDefaults = {
  page: DEFAULT_PAGE,
  status: undefined as ReplicationExecutionStatusFilterValue | undefined,
}

export const replicationExecutionsSearchSchema = z.object({
  page: z.coerce
    .number()
    .int()
    .positive()
    .default(replicationExecutionsSearchDefaults.page)
    .catch(replicationExecutionsSearchDefaults.page),
  status: z.enum(replicationExecutionStatusFilterValues)
    .optional()
    .catch(replicationExecutionsSearchDefaults.status),
})

export type ReplicationExecutionsSearch = z.infer<typeof replicationExecutionsSearchSchema>
