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
  getServerPullActionKey,
  listFailedFiles,
  shortCommit,
} from './server-pull.utils'

function cacheStatus(
  totals: ModelCacheTotals | undefined,
  taskState: CacheTaskState,
  files?: ModelCacheFile[],
): ModelCacheStatus {
  return {
    project: 'hf-proxy',
    name: 'Qwen3-0.6B',
    commit: 'c1a2b3d4e5f60718293a4b5c6d7e8f9012345678',
    state: CacheState.CACHE_STATE_PARTIAL,
    totals,
    files,
    task: {
      id: 'run-1',
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

describe('classifyServerPullError', () => {
  it('maps gateway error codes', () => {
    expect(classifyServerPullError({
      code: 5,
      message: 'model or revision not found',
    })).toBe('notFound')
    expect(classifyServerPullError({ code: 7 })).toBe('permissionDenied')
    expect(classifyServerPullError({ code: 16 })).toBe('permissionDenied')
    expect(classifyServerPullError({ code: 3 })).toBe('invalidArgument')
    expect(classifyServerPullError({ code: 9 })).toBe('failedPrecondition')
    expect(classifyServerPullError({ code: 8 })).toBe('resourceExhausted')
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
  }, CacheTaskState.CACHE_TASK_STATE_FAILED, [
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

describe('getServerPullActionKey', () => {
  it('starts, resumes or retries a pull depending on the state', () => {
    expect(getServerPullActionKey('uncached')).toBe('start')
    expect(getServerPullActionKey('unknown')).toBe('start')
    expect(getServerPullActionKey('partial')).toBe('resume')
    expect(getServerPullActionKey('failed')).toBe('retry')
  })

  it('offers nothing to start while running or once cached', () => {
    expect(getServerPullActionKey('running')).toBeNull()
    expect(getServerPullActionKey('cached')).toBeNull()
  })
})
