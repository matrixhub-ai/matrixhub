import {
  CacheFileState,
  CacheState,
  CacheTaskState,
  type ModelCacheFile,
  type ModelCacheStatus,
} from '@matrixhub/api-ts/v1alpha1/model.pb'

/** Cache state of a model revision in MatrixHub storage, derived from a `ModelCacheStatus`. */
export type ModelCacheState = 'unknown' | 'uncached' | 'partial' | 'running' | 'cached' | 'failed'

export interface ModelCacheProgress {
  doneFiles: number
  totalFiles: number
  doneBytes: number
  totalBytes: number
  /** 0–100; held at 99 until every selected file is cached. */
  percent: number
  complete: boolean
}

/** What the model detail header shows about the cache of a proxy-project model. */
export interface ModelCacheIndicatorState {
  state: Exclude<ModelCacheState, 'unknown'>
  doneFiles: number
  totalFiles: number
  /** Shown while a pull runs. */
  percent: number
}

export const CACHE_STATUS_POLL_INTERVAL_MS = 3000

// gRPC status codes, as returned in the `code` field of gateway error bodies.
export const GRPC_CODE = {
  invalidArgument: 3,
  deadlineExceeded: 4,
  notFound: 5,
  permissionDenied: 7,
  resourceExhausted: 8,
  failedPrecondition: 9,
  unavailable: 14,
  unauthenticated: 16,
} as const

/** Polling cannot recover from these: the model is gone or the user lost access. */
const TERMINAL_POLL_ERROR_CODES: readonly number[] = [
  GRPC_CODE.notFound,
  GRPC_CODE.permissionDenied,
  GRPC_CODE.unauthenticated,
]

/** int64 fields arrive as strings and any field may be missing or null. */
export function toSafeNumber(value: number | string | null | undefined): number {
  if (value === null || value === undefined || value === '') {
    return 0
  }

  const numericValue = typeof value === 'number' ? value : Number(value)

  return Number.isFinite(numericValue) ? numericValue : 0
}

export function isTaskRunning(status: ModelCacheStatus | null | undefined) {
  return status?.task?.state === CacheTaskState.CACHE_TASK_STATE_RUNNING
}

export function isFinishedTaskState(state: CacheTaskState | null | undefined) {
  return state === CacheTaskState.CACHE_TASK_STATE_SUCCEEDED
    || state === CacheTaskState.CACHE_TASK_STATE_FAILED
}

/**
 * The server decides how much is cached (`state`, counted by LFS objects); the latest pull
 * (`task`) only matters while it runs, or when it failed and files are still missing.
 */
export function resolveModelCacheState(status: ModelCacheStatus | null | undefined): ModelCacheState {
  if (!status) {
    return 'unknown'
  }

  if (isTaskRunning(status)) {
    return 'running'
  }

  if (status.state === CacheState.CACHE_STATE_CACHED) {
    return 'cached'
  }

  if (status.task?.state === CacheTaskState.CACHE_TASK_STATE_FAILED) {
    return 'failed'
  }

  switch (status.state) {
    case CacheState.CACHE_STATE_PARTIAL:
      return 'partial'
    case CacheState.CACHE_STATE_MISSING:
      return 'uncached'
    default:
      return 'unknown'
  }
}

function sumFiles(files: readonly ModelCacheFile[], pick: (file: ModelCacheFile) => number) {
  return files.reduce((total, file) => total + pick(file), 0)
}

function fileDoneBytes(file: ModelCacheFile) {
  if (file.state === CacheFileState.CACHE_FILE_STATE_CACHED) {
    return toSafeNumber(file.size)
  }

  if (file.state === CacheFileState.CACHE_FILE_STATE_DOWNLOADING) {
    return toSafeNumber(file.downloadedBytes)
  }

  return 0
}

export function computeCacheProgress(status: ModelCacheStatus | null | undefined): ModelCacheProgress {
  const totals = status?.totals
  const files = status?.files ?? []
  const totalFiles = toSafeNumber(totals?.files) || files.length
  const doneFiles = toSafeNumber(totals?.cachedFiles)
    || files.filter(file => file.state === CacheFileState.CACHE_FILE_STATE_CACHED).length
  const totalBytes = toSafeNumber(totals?.bytes) || sumFiles(files, file => toSafeNumber(file.size))
  // Per-file data includes partial downloads, which makes the bar move during large files.
  const rawDoneBytes = files.length > 0
    ? sumFiles(files, fileDoneBytes)
    : toSafeNumber(totals?.cachedBytes)
  const doneBytes = totalBytes > 0 ? Math.min(rawDoneBytes, totalBytes) : rawDoneBytes
  const complete = status?.state === CacheState.CACHE_STATE_CACHED

  let rawPercent = 0

  if (totalBytes > 0) {
    rawPercent = Math.floor((doneBytes * 100) / totalBytes)
  } else if (totalFiles > 0) {
    rawPercent = Math.floor((doneFiles * 100) / totalFiles)
  }

  return {
    doneFiles,
    totalFiles,
    doneBytes,
    totalBytes,
    percent: complete ? 100 : Math.min(99, Math.max(0, rawPercent)),
    complete,
  }
}

/** The SDK throws the raw gateway error body (`{ code, message }`), not an `Error`. */
export function getSdkErrorCode(error: unknown): number | undefined {
  if (typeof error !== 'object' || error === null || !('code' in error)) {
    return undefined
  }

  const { code } = error as { code: unknown }

  if (typeof code !== 'number' && typeof code !== 'string') {
    return undefined
  }

  const numericCode = Number(code)

  return Number.isInteger(numericCode) ? numericCode : undefined
}

/** `refetchInterval` for the cache status query: poll only while a pull task runs. */
export function getCacheStatusPollInterval(
  data: ModelCacheStatus | undefined,
  error: unknown,
): number | false {
  const code = getSdkErrorCode(error)

  if (code !== undefined && TERMINAL_POLL_ERROR_CODES.includes(code)) {
    return false
  }

  return isTaskRunning(data) ? CACHE_STATUS_POLL_INTERVAL_MS : false
}

/** Returns undefined when there is nothing to show: no status yet, or a revision without files. */
export function resolveModelCacheIndicator(status: ModelCacheStatus | null | undefined): ModelCacheIndicatorState | undefined {
  const state = resolveModelCacheState(status)
  const progress = computeCacheProgress(status)

  if (state === 'unknown' || progress.totalFiles === 0) {
    return undefined
  }

  return {
    state,
    doneFiles: progress.doneFiles,
    totalFiles: progress.totalFiles,
    percent: progress.percent,
  }
}
