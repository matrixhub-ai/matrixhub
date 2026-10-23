import {
  type CacheModelRequest,
  Models,
} from '@matrixhub/api-ts/v1alpha1/model.pb'
import { mutationOptions } from '@tanstack/react-query'

import {
  modelCacheKeys,
  modelCacheStatusQueryOptions,
} from '@/features/models/cache/cache.query'
import { modelKeys } from '@/features/models/models.query'
import { projectKeys } from '@/features/projects/projects.query'
import i18n from '@/i18n'

import type { NotificationMeta } from '@/types/tanstack-query'

/** `revision` is the commit shown in the preview, so the pull covers exactly what the user saw. */
export type CacheModelInput = Required<Pick<CacheModelRequest, 'project' | 'name' | 'revision'>>

// Starts (or attaches to) a server-side pull. The server syncs the model before the task
// starts, which creates it in a proxy project, so the model cards and the project's model
// count are refreshed right away.
export function cacheModelMutationOptions(projectId: string) {
  return mutationOptions({
    mutationFn: (input: CacheModelInput) => Models.CacheModel({
      ...input,
      dryRun: false,
    }),
    // Seed the status query with the returned task so progress shows (and polls) right away.
    onSuccess: (status, input, _onMutateResult, { client }) => {
      const { queryKey } = modelCacheStatusQueryOptions(input.project, input.name, { revision: input.revision })

      client.setQueryData(queryKey, status)
      // Previews now show stale cache counts; mark them stale without repeating the upstream
      // dry run of the preview on screen.
      void client.invalidateQueries({
        queryKey: modelCacheKeys.previews(),
        refetchType: 'none',
      })
    },
    meta: {
      errorMessage: i18n.t('model.serverPull.startError'),
      invalidates: [modelKeys.lists(), projectKeys.detail(projectId), modelCacheKeys.statuses()],
    } satisfies NotificationMeta,
  })
}
