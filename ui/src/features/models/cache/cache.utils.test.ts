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
  CACHE_STATUS_POLL_INTERVAL_MS,
  computeCacheProgress,
  getCacheStatusPollInterval,
  isFinishedTaskState,
  resolveModelCacheIndicator,
  resolveModelCacheState,
  toSafeNumber,
} from './cache.utils'

interface CacheStatusOptions {
  state?: CacheState
  /** Without it the status has no task, as when no pull ran. */
  taskState?: CacheTaskState
  files?: ModelCacheFile[]
}

function cacheStatus(totals: ModelCacheTotals | undefined, {
  state = CacheState.CACHE_STATE_PARTIAL,
  taskState,
  files,
}: CacheStatusOptions = {}): ModelCacheStatus {
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

const RUNNING = CacheTaskState.CACHE_TASK_STATE_RUNNING
const SUCCEEDED = CacheTaskState.CACHE_TASK_STATE_SUCCEEDED
const FAILED = CacheTaskState.CACHE_TASK_STATE_FAILED
const CACHED = CacheState.CACHE_STATE_CACHED
const PARTIAL = CacheState.CACHE_STATE_PARTIAL
const MISSING = CacheState.CACHE_STATE_MISSING

function cacheFile(path: string, state: CacheFileState, size: string, downloadedBytes = '0', error?: string): ModelCacheFile {
  return {
    path,
    size,
    state,
    downloadedBytes,
    error,
  }
}

describe('toSafeNumber', () => {
  it('parses int64 strings and passes numbers through', () => {
    expect(toSafeNumber('1073741824')).toBe(1073741824)
    expect(toSafeNumber(12)).toBe(12)
  })

  it('treats missing or malformed values as zero', () => {
    expect(toSafeNumber(undefined)).toBe(0)
    expect(toSafeNumber(null)).toBe(0)
    expect(toSafeNumber('')).toBe(0)
    expect(toSafeNumber('not-a-number')).toBe(0)
    expect(toSafeNumber(Number.POSITIVE_INFINITY)).toBe(0)
  })
})

describe('resolveModelCacheState', () => {
  it('is unknown without a status or a cache state', () => {
    expect(resolveModelCacheState(undefined)).toBe('unknown')
    expect(resolveModelCacheState(null)).toBe('unknown')
    expect(resolveModelCacheState({ name: 'Qwen3-0.6B' })).toBe('unknown')
  })

  it('reports a running task first, even when every file is cached', () => {
    expect(resolveModelCacheState(cacheStatus({
      files: '3',
      cachedFiles: '3',
    }, {
      state: CACHED,
      taskState: RUNNING,
    }))).toBe('running')
  })

  it('is cached once every file is cached, whatever the last task did', () => {
    for (const taskState of [undefined, SUCCEEDED, FAILED]) {
      expect(resolveModelCacheState(cacheStatus({
        files: '3',
        cachedFiles: '3',
      }, {
        state: CACHED,
        taskState,
      }))).toBe('cached')
    }
  })

  it('reports a failed task before partial progress', () => {
    expect(resolveModelCacheState(cacheStatus({
      files: '3',
      cachedFiles: '1',
    }, { taskState: FAILED }))).toBe('failed')
  })

  it('follows the server state rather than file counts', () => {
    expect(resolveModelCacheState(cacheStatus({
      files: '3',
      cachedFiles: '2',
    }, { state: PARTIAL }))).toBe('partial')
    // Files stored in git are cached once the model is synced; without LFS objects
    // the model is still not cached.
    expect(resolveModelCacheState(cacheStatus({
      files: '13',
      cachedFiles: '10',
    }, { state: MISSING }))).toBe('uncached')
  })
})

describe('computeCacheProgress', () => {
  it('counts cached files and partial downloads from the file list', () => {
    const progress = computeCacheProgress(cacheStatus({
      files: '4',
      bytes: '1000',
      cachedFiles: '1',
      cachedBytes: '200',
    }, {
      taskState: RUNNING,
      files: [
        cacheFile('config.json', CacheFileState.CACHE_FILE_STATE_CACHED, '200'),
        cacheFile('model-00001.safetensors', CacheFileState.CACHE_FILE_STATE_DOWNLOADING, '400', '150'),
        cacheFile('model-00002.safetensors', CacheFileState.CACHE_FILE_STATE_QUEUED, '300'),
        cacheFile('tokenizer.json', CacheFileState.CACHE_FILE_STATE_FAILED, '100', '60'),
      ],
    }))

    expect(progress).toEqual({
      doneFiles: 1,
      totalFiles: 4,
      doneBytes: 350,
      totalBytes: 1000,
      percent: 35,
      complete: false,
    })
  })

  it('falls back to the cached bytes of the totals without a file list', () => {
    expect(computeCacheProgress(cacheStatus({
      files: '4',
      bytes: '1000',
      cachedFiles: '2',
      cachedBytes: '500',
    }, { taskState: RUNNING }))).toMatchObject({
      doneBytes: 500,
      totalBytes: 1000,
      percent: 50,
    })
  })

  it('holds at 99% until the server reports every file cached', () => {
    const almostDone = cacheStatus({
      files: '2',
      bytes: '1000',
      cachedFiles: '1',
      cachedBytes: '999',
    }, { taskState: RUNNING })

    expect(computeCacheProgress(almostDone).percent).toBe(99)
    expect(computeCacheProgress({
      ...almostDone,
      state: CACHED,
    }).percent).toBe(100)
  })

  it('is complete only when every file is cached', () => {
    expect(computeCacheProgress(cacheStatus({
      files: '2',
      bytes: '1000',
      cachedFiles: '2',
      cachedBytes: '1000',
    }, { state: CACHED }))).toMatchObject({
      percent: 100,
      complete: true,
    })
    // A pull of a narrower selection succeeded, but files of the revision are missing.
    expect(computeCacheProgress(cacheStatus({
      files: '2',
      bytes: '1000',
      cachedFiles: '1',
      cachedBytes: '500',
    }, { taskState: SUCCEEDED }))).toMatchObject({
      percent: 50,
      complete: false,
    })
  })

  it('uses the file ratio when sizes are unknown', () => {
    expect(computeCacheProgress(cacheStatus({
      files: '4',
      bytes: '0',
      cachedFiles: '1',
    }, { taskState: RUNNING })).percent).toBe(25)
  })

  it('never reports more bytes than the total', () => {
    expect(computeCacheProgress(cacheStatus({
      files: '1',
      bytes: '100',
    }, {
      taskState: RUNNING,
      files: [
        cacheFile('model.safetensors', CacheFileState.CACHE_FILE_STATE_DOWNLOADING, '100', '120'),
      ],
    }))).toMatchObject({
      doneBytes: 100,
      percent: 99,
    })
  })

  it('derives totals from the file list when totals are missing', () => {
    expect(computeCacheProgress(cacheStatus(undefined, {
      taskState: RUNNING,
      files: [
        cacheFile('config.json', CacheFileState.CACHE_FILE_STATE_CACHED, '100'),
        cacheFile('model.safetensors', CacheFileState.CACHE_FILE_STATE_MISSING, '300'),
      ],
    }))).toEqual({
      doneFiles: 1,
      totalFiles: 2,
      doneBytes: 100,
      totalBytes: 400,
      percent: 25,
      complete: false,
    })
  })

  it('is empty without a status', () => {
    expect(computeCacheProgress(undefined)).toEqual({
      doneFiles: 0,
      totalFiles: 0,
      doneBytes: 0,
      totalBytes: 0,
      percent: 0,
      complete: false,
    })
  })
})

describe('getCacheStatusPollInterval', () => {
  const running = cacheStatus({ files: '2' }, { taskState: RUNNING })

  it('polls only while the task runs', () => {
    expect(getCacheStatusPollInterval(running, null)).toBe(CACHE_STATUS_POLL_INTERVAL_MS)
    expect(CACHE_STATUS_POLL_INTERVAL_MS).toBe(3000)
    expect(getCacheStatusPollInterval(cacheStatus({ files: '2' }, { taskState: SUCCEEDED }), null)).toBe(false)
    expect(getCacheStatusPollInterval(cacheStatus({ files: '2' }, { taskState: FAILED }), null)).toBe(false)
    expect(getCacheStatusPollInterval(cacheStatus({ files: '2' }), null)).toBe(false)
    expect(getCacheStatusPollInterval(undefined, null)).toBe(false)
  })

  it('stops when the model is gone or access is denied', () => {
    expect(getCacheStatusPollInterval(running, {
      code: 5,
      message: 'model not found',
    })).toBe(false)
    expect(getCacheStatusPollInterval(running, {
      code: 7,
      message: 'permission denied',
    })).toBe(false)
  })

  it('keeps polling through transient errors', () => {
    expect(getCacheStatusPollInterval(running, {
      code: 14,
      message: 'unavailable',
    })).toBe(CACHE_STATUS_POLL_INTERVAL_MS)
    expect(getCacheStatusPollInterval(running, new TypeError('Failed to fetch'))).toBe(CACHE_STATUS_POLL_INTERVAL_MS)
  })
})

describe('isFinishedTaskState', () => {
  it('treats succeeded and failed tasks as finished', () => {
    expect(isFinishedTaskState(SUCCEEDED)).toBe(true)
    expect(isFinishedTaskState(FAILED)).toBe(true)
    expect(isFinishedTaskState(RUNNING)).toBe(false)
    expect(isFinishedTaskState(undefined)).toBe(false)
  })
})

describe('resolveModelCacheIndicator', () => {
  it('shows nothing without a status or files', () => {
    expect(resolveModelCacheIndicator(undefined)).toBeUndefined()
    expect(resolveModelCacheIndicator(cacheStatus({ files: '0' }, { state: MISSING }))).toBeUndefined()
  })

  it('reports how many files are cached', () => {
    expect(resolveModelCacheIndicator(cacheStatus({
      files: '13',
      bytes: '1000',
      cachedFiles: '13',
      cachedBytes: '1000',
    }, {
      state: CACHED,
      taskState: SUCCEEDED,
    }))).toEqual({
      state: 'cached',
      doneFiles: 13,
      totalFiles: 13,
      percent: 100,
    })
    expect(resolveModelCacheIndicator(cacheStatus({
      files: '13',
      cachedFiles: '10',
    }))?.state).toBe('partial')
  })

  it('shows the progress of a running pull', () => {
    expect(resolveModelCacheIndicator(cacheStatus({
      files: '13',
      bytes: '1000',
      cachedFiles: '10',
      cachedBytes: '200',
    }, { taskState: RUNNING }))).toMatchObject({
      state: 'running',
      percent: 20,
    })
  })
})
