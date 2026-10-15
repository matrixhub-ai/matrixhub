import {
  CacheFileState,
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

function cacheStatus(
  totals: ModelCacheTotals | undefined,
  taskState: CacheTaskState = CacheTaskState.CACHE_TASK_STATE_IDLE,
  files?: ModelCacheFile[],
): ModelCacheStatus {
  return {
    project: 'hf-proxy',
    name: 'Qwen3-0.6B',
    commit: 'c1a2b3d4e5f60718293a4b5c6d7e8f9012345678',
    totals,
    files,
    task: {
      id: 'task-1',
      state: taskState,
    },
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
  it('is unknown without a status', () => {
    expect(resolveModelCacheState(undefined)).toBe('unknown')
    expect(resolveModelCacheState(null)).toBe('unknown')
  })

  it('reports a running task first, even when every file is cached', () => {
    expect(resolveModelCacheState(cacheStatus({
      files: '3',
      cachedFiles: '3',
    }, CacheTaskState.CACHE_TASK_STATE_RUNNING))).toBe('running')
  })

  it('is cached once every selected file is cached, whatever the last task did', () => {
    expect(resolveModelCacheState(cacheStatus({
      files: '3',
      cachedFiles: '3',
    }, CacheTaskState.CACHE_TASK_STATE_SUCCEEDED))).toBe('cached')
    expect(resolveModelCacheState(cacheStatus({
      files: '3',
      cachedFiles: '3',
    }, CacheTaskState.CACHE_TASK_STATE_FAILED))).toBe('cached')
  })

  it('reports a failed task before partial progress', () => {
    expect(resolveModelCacheState(cacheStatus({
      files: '3',
      cachedFiles: '1',
    }, CacheTaskState.CACHE_TASK_STATE_FAILED))).toBe('failed')
  })

  it('distinguishes partially cached from uncached models', () => {
    expect(resolveModelCacheState(cacheStatus({
      files: '3',
      cachedFiles: '1',
    }))).toBe('partial')
    expect(resolveModelCacheState(cacheStatus({
      files: '3',
      cachedFiles: '0',
    }))).toBe('uncached')
  })

  it('does not call an empty selection cached and tolerates missing totals and task', () => {
    expect(resolveModelCacheState(cacheStatus({
      files: '0',
      cachedFiles: '0',
    }))).toBe('uncached')
    expect(resolveModelCacheState({ name: 'Qwen3-0.6B' })).toBe('uncached')
  })
})

describe('computeCacheProgress', () => {
  it('counts cached files and partial downloads from the file list', () => {
    const progress = computeCacheProgress(cacheStatus({
      files: '4',
      bytes: '1000',
      cachedFiles: '1',
      cachedBytes: '200',
    }, CacheTaskState.CACHE_TASK_STATE_RUNNING, [
      cacheFile('config.json', CacheFileState.CACHE_FILE_STATE_CACHED, '200'),
      cacheFile('model-00001.safetensors', CacheFileState.CACHE_FILE_STATE_DOWNLOADING, '400', '150'),
      cacheFile('model-00002.safetensors', CacheFileState.CACHE_FILE_STATE_QUEUED, '300'),
      cacheFile('tokenizer.json', CacheFileState.CACHE_FILE_STATE_FAILED, '100', '60'),
    ]))

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
    }, CacheTaskState.CACHE_TASK_STATE_RUNNING))).toMatchObject({
      doneBytes: 500,
      totalBytes: 1000,
      percent: 50,
    })
  })

  it('holds at 99% until the pull completes', () => {
    const almostDone = cacheStatus({
      files: '2',
      bytes: '1000',
      cachedFiles: '1',
      cachedBytes: '999',
    }, CacheTaskState.CACHE_TASK_STATE_RUNNING)

    expect(computeCacheProgress(almostDone).percent).toBe(99)
    expect(computeCacheProgress(cacheStatus({
      files: '2',
      bytes: '1000',
      cachedFiles: '2',
      cachedBytes: '1000',
    }, CacheTaskState.CACHE_TASK_STATE_RUNNING)).percent).toBe(99)
  })

  it('is complete when every file is cached or the task succeeded', () => {
    expect(computeCacheProgress(cacheStatus({
      files: '2',
      bytes: '1000',
      cachedFiles: '2',
      cachedBytes: '1000',
    }))).toMatchObject({
      percent: 100,
      complete: true,
    })
    expect(computeCacheProgress(cacheStatus({
      files: '0',
      bytes: '0',
    }, CacheTaskState.CACHE_TASK_STATE_SUCCEEDED))).toMatchObject({
      percent: 100,
      complete: true,
    })
  })

  it('uses the file ratio when sizes are unknown', () => {
    expect(computeCacheProgress(cacheStatus({
      files: '4',
      bytes: '0',
      cachedFiles: '1',
    }, CacheTaskState.CACHE_TASK_STATE_RUNNING)).percent).toBe(25)
  })

  it('never reports more bytes than the total', () => {
    expect(computeCacheProgress(cacheStatus({
      files: '1',
      bytes: '100',
    }, CacheTaskState.CACHE_TASK_STATE_RUNNING, [
      cacheFile('model.safetensors', CacheFileState.CACHE_FILE_STATE_DOWNLOADING, '100', '120'),
    ]))).toMatchObject({
      doneBytes: 100,
      percent: 99,
    })
  })

  it('derives totals from the file list when totals are missing', () => {
    expect(computeCacheProgress(cacheStatus(undefined, CacheTaskState.CACHE_TASK_STATE_RUNNING, [
      cacheFile('config.json', CacheFileState.CACHE_FILE_STATE_CACHED, '100'),
      cacheFile('model.safetensors', CacheFileState.CACHE_FILE_STATE_MISSING, '300'),
    ]))).toEqual({
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
  const running = cacheStatus({ files: '2' }, CacheTaskState.CACHE_TASK_STATE_RUNNING)

  it('polls only while the task runs', () => {
    expect(getCacheStatusPollInterval(running, null)).toBe(CACHE_STATUS_POLL_INTERVAL_MS)
    expect(CACHE_STATUS_POLL_INTERVAL_MS).toBe(3000)
    expect(getCacheStatusPollInterval(cacheStatus({ files: '2' }, CacheTaskState.CACHE_TASK_STATE_SUCCEEDED), null)).toBe(false)
    expect(getCacheStatusPollInterval(cacheStatus({ files: '2' }, CacheTaskState.CACHE_TASK_STATE_FAILED), null)).toBe(false)
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
    expect(isFinishedTaskState(CacheTaskState.CACHE_TASK_STATE_SUCCEEDED)).toBe(true)
    expect(isFinishedTaskState(CacheTaskState.CACHE_TASK_STATE_FAILED)).toBe(true)
    expect(isFinishedTaskState(CacheTaskState.CACHE_TASK_STATE_RUNNING)).toBe(false)
    expect(isFinishedTaskState(CacheTaskState.CACHE_TASK_STATE_IDLE)).toBe(false)
    expect(isFinishedTaskState(undefined)).toBe(false)
  })
})

describe('resolveModelCacheIndicator', () => {
  it('shows nothing without a status or files', () => {
    expect(resolveModelCacheIndicator(undefined)).toBeUndefined()
    expect(resolveModelCacheIndicator(cacheStatus({ files: '0' }))).toBeUndefined()
  })

  it('reports how many files are cached', () => {
    expect(resolveModelCacheIndicator(cacheStatus({
      files: '13',
      bytes: '1000',
      cachedFiles: '13',
      cachedBytes: '1000',
    }, CacheTaskState.CACHE_TASK_STATE_SUCCEEDED))).toEqual({
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
    }, CacheTaskState.CACHE_TASK_STATE_RUNNING))).toMatchObject({
      state: 'running',
      percent: 20,
    })
  })
})
