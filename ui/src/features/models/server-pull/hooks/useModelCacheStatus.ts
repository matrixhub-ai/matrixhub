import {
  CacheTaskState,
  type ModelCacheStatus,
} from '@matrixhub/api-ts/v1alpha1/model.pb'
import {
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import {
  useEffect,
  useRef,
} from 'react'

import {
  type ModelCacheSelection,
  modelCacheStatusQueryOptions,
} from '@/features/models/cache/cache.query'
import { isFinishedTaskState } from '@/features/models/cache/cache.utils'
import { modelKeys } from '@/features/models/models.query'
import { projectKeys } from '@/features/projects/projects.query'

interface UseModelCacheStatusOptions {
  enabled: boolean
  /** Shown until the first status arrives, e.g. the status returned when the pull started. */
  placeholderData?: ModelCacheStatus
}

/**
 * Polls the local cache status of a model while its pull task runs. When the task finishes,
 * the project's model cards and model count are refreshed, because the pull filled the model
 * on the server.
 */
export function useModelCacheStatus(
  projectId: string,
  modelName: string,
  params: ModelCacheSelection,
  {
    enabled,
    placeholderData,
  }: UseModelCacheStatusOptions,
) {
  const queryClient = useQueryClient()
  const query = useQuery({
    ...modelCacheStatusQueryOptions(projectId, modelName, params),
    enabled: enabled && modelName !== '',
    placeholderData,
  })
  const taskState = query.data?.task?.state
  const previousTaskStateRef = useRef(taskState)

  useEffect(() => {
    const previousTaskState = previousTaskStateRef.current

    previousTaskStateRef.current = taskState

    if (previousTaskState === CacheTaskState.CACHE_TASK_STATE_RUNNING && isFinishedTaskState(taskState)) {
      void queryClient.invalidateQueries({ queryKey: modelKeys.lists() })
      void queryClient.invalidateQueries({ queryKey: projectKeys.detail(projectId) })
    }
  }, [projectId, queryClient, taskState])

  return query
}
