import {
  ResourceType,
  SyncJobStatus,
} from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import { z } from 'zod'

import { DEFAULT_PAGE } from '@/utils/constants'

export const syncJobStatusFilterValues = [
  SyncJobStatus.SYNC_JOB_STATUS_RUNNING,
  SyncJobStatus.SYNC_JOB_STATUS_SUCCEEDED,
  SyncJobStatus.SYNC_JOB_STATUS_FAILED,
  SyncJobStatus.SYNC_JOB_STATUS_STOPPED,
] as const

export const syncJobResourceTypeFilterValues = [
  ResourceType.RESOURCE_TYPE_MODEL,
  ResourceType.RESOURCE_TYPE_DATASET,
] as const

export const executionDetailSearchSchema = z.object({
  page: z.coerce.number().int().positive().optional().catch(DEFAULT_PAGE),
  status: z.enum(syncJobStatusFilterValues).optional(),
  resourceType: z.enum(syncJobResourceTypeFilterValues).optional(),
})

export type ExecutionDetailSearch = z.infer<typeof executionDetailSearchSchema>

export const executionDetailSearchDefaults: ExecutionDetailSearch = {
  page: DEFAULT_PAGE,
  status: undefined,
  resourceType: undefined,
}
