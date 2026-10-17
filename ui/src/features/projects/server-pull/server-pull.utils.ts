import {
  CacheFileState,
  type ModelCacheFile,
  type ModelCacheStatus,
} from '@matrixhub/api-ts/v1alpha1/model.pb'

import {
  getSdkErrorCode,
  GRPC_CODE,
  toSafeNumber,
  type ModelCacheState,
} from '@/features/models/cache/cache.utils'

export type ServerPullErrorKind = 'notFound' | 'permissionDenied' | 'invalidArgument' | 'failedPrecondition' | 'resourceExhausted' | 'unavailable' | 'unknown'

/** Locale key suffix of the button that starts a pull. */
export type ServerPullActionKey = 'start' | 'resume' | 'retry'

export interface FailedFilesSummary {
  items: ModelCacheFile[]
  /** Failed files not included in `items`. */
  rest: number
}

export function classifyServerPullError(error: unknown): ServerPullErrorKind {
  switch (getSdkErrorCode(error)) {
    case GRPC_CODE.notFound:
      return 'notFound'
    case GRPC_CODE.permissionDenied:
    case GRPC_CODE.unauthenticated:
      return 'permissionDenied'
    case GRPC_CODE.invalidArgument:
      return 'invalidArgument'
    case GRPC_CODE.failedPrecondition:
      return 'failedPrecondition'
    case GRPC_CODE.resourceExhausted:
      return 'resourceExhausted'
    case GRPC_CODE.unavailable:
    case GRPC_CODE.deadlineExceeded:
      return 'unavailable'
    default:
      return 'unknown'
  }
}

export function shortCommit(sha: string | null | undefined) {
  return sha?.trim().slice(0, 7) ?? ''
}

export function listFailedFiles(status: ModelCacheStatus | null | undefined, limit: number): FailedFilesSummary {
  const failed = (status?.files ?? []).filter(file => file.state === CacheFileState.CACHE_FILE_STATE_FAILED)
  const items = failed.slice(0, Math.max(0, limit))
  // The totals count every failure, also when `files` is missing.
  const totalFailed = Math.max(failed.length, toSafeNumber(status?.totals?.failedFiles))

  return {
    items,
    rest: Math.max(0, totalFailed - items.length),
  }
}

/** Label of the start button for a state; `null` when there is nothing to start. */
export function getServerPullActionKey(state: ModelCacheState): ServerPullActionKey | null {
  switch (state) {
    case 'running':
    case 'cached':
      return null
    case 'partial':
      return 'resume'
    case 'failed':
      return 'retry'
    default:
      return 'start'
  }
}
