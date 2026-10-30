import {
  CacheFileState,
  CacheState,
  CacheTaskState,
  type ModelCacheFile,
  type ModelCacheStatus,
  type ModelCacheTotals,
} from '@matrixhub/api-ts/v1alpha1/model.pb'
import {
  describe, expect, it,
} from 'vitest'

import {
  classifyServerPullError,
  listFailedFiles,
  resolveDownloadHintKey,
  resolveDownloadMethod,
  resolveServerPullFooter,
  resolveServerPullView,
  type ServerPullView,
  type ServerPullViewInput,
  shortCommit,
} from './server-pull.utils'

function cacheStatus(
  totals: ModelCacheTotals | undefined,
  taskState?: CacheTaskState,
  files?: ModelCacheFile[],
  state = CacheState.CACHE_STATE_PARTIAL,
): ModelCacheStatus {
  return {
    project: 'hf-proxy',
    name: 'Qwen3-0.6B',
    commit: 'c1a2b3d4e5f60718293a4b5c6d7e8f9012345678',
    state,
    totals,
    files,
    task: taskState
      ? {
          id: 'run-1',
          state: taskState,
        }
      : undefined,
  }
}

function cacheFile(path: string, state: CacheFileState, size: string, downloadedBytes = '0', error?: string): ModelCacheFile {
  return {
    path,
    size,
    state,
    downloadedBytes,
    error,
  }
}

const RUNNING = CacheTaskState.CACHE_TASK_STATE_RUNNING
const SUCCEEDED = CacheTaskState.CACHE_TASK_STATE_SUCCEEDED
const FAILED = CacheTaskState.CACHE_TASK_STATE_FAILED
const CACHED = CacheState.CACHE_STATE_CACHED
const MISSING = CacheState.CACHE_STATE_MISSING

const files = { files: '3' }
const missing = cacheStatus(files, undefined, undefined, MISSING)
const running = cacheStatus(files, RUNNING)
const cached = cacheStatus(files, SUCCEEDED, undefined, CACHED)

function viewInput(overrides: Partial<ServerPullViewInput> = {}): ServerPullViewInput {
  return {
    modelName: 'Qwen3-0.6B',
    settling: false,
    started: false,
    preview: {
      status: missing,
      error: null,
      pending: false,
    },
    tracked: {
      status: undefined,
      error: null,
    },
    ...overrides,
  }
}

describe('classifyServerPullError', () => {
  it('maps gateway error codes', () => {
    expect(classifyServerPullError({
      code: 5,
      message: 'model or revision not found',
    })).toBe('notFound')
    expect(classifyServerPullError({ code: 7 })).toBe('permissionDenied')
    expect(classifyServerPullError({ code: 16 })).toBe('unauthenticated')
    expect(classifyServerPullError({ code: 3 })).toBe('invalidArgument')
    expect(classifyServerPullError({ code: 9 })).toBe('failedPrecondition')
    expect(classifyServerPullError({ code: 14 })).toBe('unavailable')
    expect(classifyServerPullError({ code: 4 })).toBe('unavailable')
  })

  it('accepts numeric string codes', () => {
    expect(classifyServerPullError({ code: '5' })).toBe('notFound')
  })

  it('falls back to unknown for other errors', () => {
    expect(classifyServerPullError({ code: 13 })).toBe('unknown')
    expect(classifyServerPullError({ code: null })).toBe('unknown')
    expect(classifyServerPullError(new TypeError('Failed to fetch'))).toBe('unknown')
    expect(classifyServerPullError('boom')).toBe('unknown')
    expect(classifyServerPullError(undefined)).toBe('unknown')
  })
})

describe('shortCommit', () => {
  it('keeps the first 7 characters of a commit', () => {
    expect(shortCommit('c1a2b3d4e5f60718293a4b5c6d7e8f9012345678')).toBe('c1a2b3d')
    expect(shortCommit(' abc ')).toBe('abc')
    expect(shortCommit(undefined)).toBe('')
    expect(shortCommit(null)).toBe('')
  })
})

describe('listFailedFiles', () => {
  const failedStatus = cacheStatus({
    files: '5',
    failedFiles: '3',
  }, FAILED, [
    cacheFile('config.json', CacheFileState.CACHE_FILE_STATE_CACHED, '10'),
    cacheFile('a.safetensors', CacheFileState.CACHE_FILE_STATE_FAILED, '10', '0', 'upstream returned 502'),
    cacheFile('b.safetensors', CacheFileState.CACHE_FILE_STATE_FAILED, '10', '0', 'checksum mismatch'),
    cacheFile('c.safetensors', CacheFileState.CACHE_FILE_STATE_FAILED, '10'),
    cacheFile('d.safetensors', CacheFileState.CACHE_FILE_STATE_MISSING, '10'),
  ])

  it('lists failed files up to the limit and counts the rest', () => {
    const {
      items,
      rest,
    } = listFailedFiles(failedStatus, 2)

    expect(items.map(file => file.path)).toEqual(['a.safetensors', 'b.safetensors'])
    expect(items[0]?.error).toBe('upstream returned 502')
    expect(rest).toBe(1)
  })

  it('counts failures missing from the file list', () => {
    expect(listFailedFiles({
      ...failedStatus,
      totals: {
        files: '9',
        failedFiles: '7',
      },
    }, 5).rest).toBe(4)
  })

  it('is empty without failures', () => {
    expect(listFailedFiles(undefined, 5)).toEqual({
      items: [],
      rest: 0,
    })
  })
})

describe('resolveDownloadMethod', () => {
  it('falls back to the CLI download when server pull is not allowed', () => {
    expect(resolveDownloadMethod(true, 'server')).toBe('server')
    expect(resolveDownloadMethod(true, 'cli')).toBe('cli')
    expect(resolveDownloadMethod(false, 'server')).toBe('cli')
  })
})

describe('resolveDownloadHintKey', () => {
  it('keeps the CLI hint, and drops the server hint when there is nothing to pull', () => {
    expect(resolveDownloadHintKey('cli', { kind: 'idle' })).toBe('model.serverPull.hints.cli')
    expect(resolveDownloadHintKey('server', { kind: 'idle' })).toBe('model.serverPull.hints.server')
    expect(resolveDownloadHintKey('server', { kind: 'trackingPending' })).toBe('model.serverPull.hints.server')
    expect(resolveDownloadHintKey('server', {
      kind: 'cached',
      status: {},
    })).toBeUndefined()
    expect(resolveDownloadHintKey('server', {
      kind: 'empty',
      status: {},
    })).toBeUndefined()
  })
})

describe('resolveServerPullView', () => {
  it('waits for a model name and for typing to settle', () => {
    expect(resolveServerPullView(viewInput({ modelName: '' }))).toEqual({ kind: 'idle' })
    expect(resolveServerPullView(viewInput({ settling: true }))).toEqual({ kind: 'checking' })
    expect(resolveServerPullView(viewInput({
      preview: {
        status: undefined,
        error: null,
        pending: true,
      },
    }))).toEqual({ kind: 'checking' })
  })

  it('shows a failed preview', () => {
    const error = {
      code: 5,
      message: 'not found',
    }

    expect(resolveServerPullView(viewInput({
      preview: {
        status: undefined,
        error,
        pending: false,
      },
    }))).toEqual({
      kind: 'checkFailed',
      error,
    })
  })

  it('offers a pull from the preview, unless nothing is left to pull', () => {
    expect(resolveServerPullView(viewInput())).toMatchObject({
      kind: 'ready',
      state: 'uncached',
    })
    expect(resolveServerPullView(viewInput({
      preview: {
        status: cached,
        error: null,
        pending: false,
      },
    }))).toMatchObject({ kind: 'cached' })
    expect(resolveServerPullView(viewInput({
      preview: {
        status: cacheStatus({ files: '0' }),
        error: null,
        pending: false,
      },
    }))).toMatchObject({ kind: 'empty' })
  })

  it('attaches to a pull that is already running', () => {
    const preview = {
      status: running,
      error: null,
      pending: false,
    }

    expect(resolveServerPullView(viewInput({ preview }))).toEqual({ kind: 'trackingPending' })
    expect(resolveServerPullView(viewInput({
      preview,
      tracked: {
        status: running,
        error: null,
      },
    }))).toMatchObject({
      kind: 'tracking',
      state: 'running',
      pollError: undefined,
    })
  })

  it('tracks a pull started here with its latest status', () => {
    expect(resolveServerPullView(viewInput({
      started: true,
      tracked: {
        status: cacheStatus(files, FAILED),
        error: null,
      },
    }))).toMatchObject({
      kind: 'tracking',
      state: 'failed',
    })
  })

  // A task can succeed while files of the revision are still missing, e.g. when it covered a
  // narrower selection; the cache state decides.
  it('reports a succeeded task with missing files as partial', () => {
    expect(resolveServerPullView(viewInput({
      started: true,
      tracked: {
        status: cacheStatus(files, SUCCEEDED),
        error: null,
      },
    }))).toMatchObject({
      kind: 'tracking',
      state: 'partial',
    })
  })

  it('freezes the progress when the status can no longer be read', () => {
    const unauthenticated = {
      code: 16,
      message: 'unauthenticated',
    }

    expect(resolveServerPullView(viewInput({
      started: true,
      tracked: {
        status: running,
        error: unauthenticated,
      },
    }))).toMatchObject({
      kind: 'tracking',
      state: 'running',
      pollError: unauthenticated,
    })
    // Transient errors keep polling and show nothing.
    expect(resolveServerPullView(viewInput({
      started: true,
      tracked: {
        status: running,
        error: { code: 14 },
      },
    }))).toMatchObject({ pollError: undefined })
    expect(resolveServerPullView(viewInput({
      started: true,
      tracked: {
        status: undefined,
        error: unauthenticated,
      },
    }))).toEqual({
      kind: 'trackingFailed',
      error: unauthenticated,
    })
  })
})

describe('resolveServerPullFooter', () => {
  const ready = resolveServerPullView(viewInput())
  const tracking = (status: ModelCacheStatus, error: unknown = null): ServerPullView => resolveServerPullView(viewInput({
    started: true,
    tracked: {
      status,
      error,
    },
  }))

  it('starts a pull only once the preview is ready', () => {
    expect(resolveServerPullFooter({ kind: 'idle' }, true)).toEqual({
      primary: {
        action: 'start',
        disabled: true,
      },
      dismiss: 'cancel',
    })
    expect(resolveServerPullFooter(ready, true)).toEqual({
      primary: {
        action: 'start',
        disabled: false,
      },
      dismiss: 'cancel',
    })
  })

  it('retries a failed pull, also when the preview shows it', () => {
    expect(resolveServerPullFooter(resolveServerPullView(viewInput({
      preview: {
        status: cacheStatus(files, FAILED),
        error: null,
        pending: false,
      },
    })), true).primary).toEqual({
      action: 'retry',
      disabled: false,
    })
    expect(resolveServerPullFooter(tracking(cacheStatus(files, FAILED)), true).primary?.action).toBe('retry')
  })

  it('only closes while a pull runs, or when its progress can no longer be read', () => {
    expect(resolveServerPullFooter(tracking(running), true)).toEqual({
      primary: null,
      dismiss: 'close',
    })
    expect(resolveServerPullFooter(tracking(running, { code: 5 }), true)).toEqual({
      primary: null,
      dismiss: 'close',
    })
  })

  it('links to the pulled model where it is not already shown', () => {
    const cachedView = resolveServerPullView(viewInput({
      preview: {
        status: cached,
        error: null,
        pending: false,
      },
    }))

    expect(resolveServerPullFooter(cachedView, true)).toEqual({
      primary: {
        action: 'viewModel',
        disabled: false,
      },
      dismiss: 'close',
    })
    expect(resolveServerPullFooter(cachedView, false).primary).toBeNull()
    expect(resolveServerPullFooter(tracking(cached), true).primary?.action).toBe('viewModel')
  })

  it('starts again when a pull stopped without finishing', () => {
    expect(resolveServerPullFooter(tracking(cacheStatus(files, SUCCEEDED)), true).primary).toEqual({
      action: 'start',
      disabled: false,
    })
  })
})
