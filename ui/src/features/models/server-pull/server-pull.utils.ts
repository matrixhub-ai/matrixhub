import {
  CacheFileState,
  type ModelCacheFile,
  type ModelCacheStatus,
} from '@matrixhub/api-ts/v1alpha1/model.pb'

import {
  isTaskRunning,
  isTerminalPollError,
  type ModelCacheState,
  resolveModelCacheState,
  toSafeNumber,
} from '@/features/models/cache/cache.utils'
import {
  getSdkErrorCode,
  GRPC_CODE,
} from '@/utils/sdkError'

/** How a proxy-project model is downloaded: pulled by the MatrixHub server, or with the hf CLI. */
export type DownloadMethod = 'server' | 'cli'

/** Locale key suffix (under `model.serverPull.errors` / `pollErrors`) of a failed request. */
export type ServerPullErrorKind = 'notFound' | 'permissionDenied' | 'unauthenticated' | 'invalidArgument' | 'failedPrecondition' | 'unavailable' | 'unknown'

export interface FailedFilesSummary {
  items: ModelCacheFile[]
  /** Failed files not included in `items`. */
  rest: number
}

interface PlainView {
  kind: 'idle' | 'checking' | 'trackingPending'
}

interface CheckFailedView {
  kind: 'checkFailed'
  error: unknown
}

interface ReadyView {
  kind: 'ready'
  status: ModelCacheStatus
  state: ModelCacheState
}

interface EmptyOrCachedView {
  kind: 'empty' | 'cached'
  status: ModelCacheStatus
}

interface TrackingView {
  kind: 'tracking'
  status: ModelCacheStatus
  state: ModelCacheState
  /** Set when the status can no longer be read, so the progress is frozen. */
  pollError?: unknown
}

interface TrackingFailedView {
  kind: 'trackingFailed'
  error: unknown
}

/**
 * What the server pull section shows for the selected model and revision:
 * - `idle`: no valid model name yet.
 * - `checking`: waiting for typing to settle, or for the preview (dry run).
 * - `checkFailed`: the preview failed.
 * - `ready`: the preview is loaded and a pull can start.
 * - `empty`: the revision has no files.
 * - `cached`: every file is already cached.
 * - `tracking`: a pull started here, or one already running, with its latest status.
 * - `trackingPending` / `trackingFailed`: a pull started but its status has not loaded.
 */
export type ServerPullView = PlainView | CheckFailedView | ReadyView | EmptyOrCachedView | TrackingView | TrackingFailedView

export interface ServerPullViewInput {
  /** Validated and settled model name; empty while there is none. */
  modelName: string
  /** The name or revision is still being typed. */
  settling: boolean
  /** A pull of this selection was started from this drawer. */
  started: boolean
  preview: {
    status?: ModelCacheStatus
    error: unknown
    pending: boolean
  }
  tracked: {
    status?: ModelCacheStatus
    error: unknown
  }
}

/** Primary footer action: start, retry, or open the pulled model. */
export type ServerPullAction = 'start' | 'retry' | 'viewModel'

export interface ServerPullFooterState {
  primary: {
    action: ServerPullAction
    disabled: boolean
  } | null
  /** "Cancel" before a pull exists, "Close" after, so closing never reads as cancelling a pull. */
  dismiss: 'cancel' | 'close'
}

/**
 * Locale key of the drawer footer's hint. With the server method it says that a pull keeps
 * running after the drawer closes, so it is left out when there is nothing to pull.
 */
export function resolveDownloadHintKey(method: DownloadMethod, view: ServerPullView) {
  if (method === 'cli') {
    return 'model.serverPull.hints.cli'
  }

  return view.kind === 'cached' || view.kind === 'empty' ? undefined : 'model.serverPull.hints.server'
}

/** Users who cannot pull on the server keep the CLI-only flow. */
export function resolveDownloadMethod(canPullOnServer: boolean, method: DownloadMethod): DownloadMethod {
  return canPullOnServer ? method : 'cli'
}

export function classifyServerPullError(error: unknown): ServerPullErrorKind {
  switch (getSdkErrorCode(error)) {
    case GRPC_CODE.notFound:
      return 'notFound'
    case GRPC_CODE.permissionDenied:
      return 'permissionDenied'
    case GRPC_CODE.unauthenticated:
      return 'unauthenticated'
    case GRPC_CODE.invalidArgument:
      return 'invalidArgument'
    case GRPC_CODE.failedPrecondition:
      return 'failedPrecondition'
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

export function resolveServerPullView({
  modelName,
  settling,
  started,
  preview,
  tracked,
}: ServerPullViewInput): ServerPullView {
  if (!modelName) {
    return { kind: 'idle' }
  }

  if (settling) {
    return { kind: 'checking' }
  }

  // Track a pull started here, or attach to one already running (e.g. after a reload).
  if (started || isTaskRunning(preview.status)) {
    if (!tracked.status) {
      return tracked.error
        ? {
            kind: 'trackingFailed',
            error: tracked.error,
          }
        : { kind: 'trackingPending' }
    }

    return {
      kind: 'tracking',
      status: tracked.status,
      state: resolveModelCacheState(tracked.status),
      // Transient errors keep polling; only errors that stop it are worth showing.
      pollError: isTerminalPollError(tracked.error) ? tracked.error : undefined,
    }
  }

  if (preview.error) {
    return {
      kind: 'checkFailed',
      error: preview.error,
    }
  }

  if (preview.pending || !preview.status) {
    return { kind: 'checking' }
  }

  const { status } = preview

  if (toSafeNumber(status.totals?.files) === 0) {
    return {
      kind: 'empty',
      status,
    }
  }

  const state = resolveModelCacheState(status)

  if (state === 'cached') {
    return {
      kind: 'cached',
      status,
    }
  }

  return {
    kind: 'ready',
    status,
    state,
  }
}

/**
 * Footer buttons for a view. `canViewModel` offers a link to the pulled model; the model
 * page itself has no use for it.
 */
export function resolveServerPullFooter(view: ServerPullView, canViewModel: boolean): ServerPullFooterState {
  const viewModel = canViewModel
    ? {
        action: 'viewModel' as const,
        disabled: false,
      }
    : null

  switch (view.kind) {
    case 'idle':
    case 'checking':
    case 'checkFailed':
    case 'empty':
      return {
        primary: {
          action: 'start',
          disabled: true,
        },
        dismiss: 'cancel',
      }
    case 'ready':
      return {
        primary: {
          // A failed pull is retried; a partial cache is a plain start, because its cached
          // files may come from other models that share LFS objects.
          action: view.state === 'failed' ? 'retry' : 'start',
          disabled: false,
        },
        dismiss: 'cancel',
      }
    case 'cached':
      return {
        primary: viewModel,
        dismiss: 'close',
      }
    case 'tracking':
      return {
        primary: view.pollError ? null : trackingAction(view.state, viewModel),
        dismiss: 'close',
      }
    case 'trackingPending':
    case 'trackingFailed':
      return {
        primary: null,
        dismiss: 'close',
      }
  }
}

function trackingAction(state: ModelCacheState, viewModel: ServerPullFooterState['primary']): ServerPullFooterState['primary'] {
  switch (state) {
    case 'running':
      return null
    case 'cached':
      return viewModel
    case 'failed':
      return {
        action: 'retry',
        disabled: false,
      }
    default:
      // The task stopped without finishing, e.g. the server restarted.
      return {
        action: 'start',
        disabled: false,
      }
  }
}
