import { useDebouncedValue } from '@mantine/hooks'
import {
  useMutation,
  useQuery,
} from '@tanstack/react-query'
import { useState } from 'react'

import { modelCachePreviewQueryOptions } from '@/features/models/cache/cache.query'
import { isTaskRunning } from '@/features/models/cache/cache.utils'

import {
  type CacheModelInput,
  cacheModelMutationOptions,
} from '../server-pull.mutation'
import { resolveServerPullView } from '../server-pull.utils'
import { useModelCacheStatus } from './useModelCacheStatus'

const INPUT_DEBOUNCE_MS = 400

interface UseServerPullOptions {
  projectId: string
  /** Validated upstream model name; empty while there is none. */
  modelName: string
  /** Branch, tag or commit; empty means the upstream default branch. */
  revision: string
  /** False while the CLI method is chosen: nothing is previewed or polled. */
  enabled: boolean
  /** How long typing must pause before the preview contacts upstream. */
  debounceMs?: number
}

/**
 * Preview (dry run), start and progress of the server-side pull of one model revision. The
 * returned `view` says what to show, see `resolveServerPullView`.
 */
export function useServerPull({
  projectId,
  modelName,
  revision,
  enabled,
  debounceMs = INPUT_DEBOUNCE_MS,
}: UseServerPullOptions) {
  const trimmedRevision = revision.trim()
  const [debouncedName] = useDebouncedValue(modelName, debounceMs)
  const [debouncedRevision] = useDebouncedValue(trimmedRevision, debounceMs)
  const settling = modelName !== debouncedName || trimmedRevision !== debouncedRevision
  const selectionKey = JSON.stringify([debouncedName, debouncedRevision])
  // The selection whose pull started here. Another name or revision shows its own preview,
  // while the earlier pull keeps running on the server.
  const [startedKey, setStartedKey] = useState<string | null>(null)
  const started = startedKey === selectionKey

  const preview = useQuery({
    ...modelCachePreviewQueryOptions(projectId, debouncedName, { revision: debouncedRevision }),
    // The preview contacts upstream; once the pull runs, progress comes from the status query.
    enabled: enabled && debouncedName !== '' && !settling && !started,
  })
  const mutation = useMutation(cacheModelMutationOptions(projectId))
  // Pull the commit the preview resolved, so the pulled files match the preview.
  const pullInput: CacheModelInput = (started ? mutation.variables : undefined) ?? {
    project: projectId,
    name: debouncedName,
    revision: preview.data?.commit || debouncedRevision,
  }
  const tracked = useModelCacheStatus(projectId, debouncedName, { revision: pullInput.revision }, {
    enabled: enabled && (started || isTaskRunning(preview.data)),
    placeholderData: (started ? mutation.data : undefined) ?? preview.data,
  })

  const view = resolveServerPullView({
    modelName: debouncedName,
    settling,
    started,
    preview: {
      status: preview.data,
      error: preview.error,
      pending: preview.isPending,
    },
    tracked: {
      status: tracked.data,
      error: tracked.error,
    },
  })

  return {
    view,
    /** The settled model name the view is about. */
    modelName: debouncedName,
    /** Name of the upstream default branch once its preview tells it; undefined for other revisions. */
    defaultBranch: debouncedRevision === '' ? preview.data?.revision || undefined : undefined,
    starting: mutation.isPending,
    start: () => {
      mutation.mutate(pullInput, {
        onSuccess: () => setStartedKey(selectionKey),
      })
    },
    /** Repeats a failed preview. */
    recheck: () => void preview.refetch(),
    rechecking: preview.isFetching,
    /** Repeats a failed status request. */
    reload: () => void tracked.refetch(),
    reloading: tracked.isFetching,
  }
}

export type ServerPull = ReturnType<typeof useServerPull>
