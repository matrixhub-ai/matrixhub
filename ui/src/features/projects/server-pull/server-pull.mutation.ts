import { Models } from '@matrixhub/api-ts/v1alpha1/model.pb'
import { mutationOptions } from '@tanstack/react-query'

import {
  modelCacheKeys,
  modelCacheStatusQueryOptions,
} from '@/features/models/cache/cache.query'
import { modelKeys } from '@/features/models/models.query'
import i18n from '@/i18n'

import type { NotificationMeta } from '@/types/tanstack-query'

export interface CacheModelInput {
  project: string
  name: string
  /** Pass the commit shown in the preview so the pull covers exactly what the user saw. */
  revision: string
}

// Starts (or attaches to) a server-side pull. The server creates the model in the proxy
// project, so model lists are refreshed. The project detail (model count) is refreshed once
// the task finishes, see `useModelCacheStatus`.
export function cacheModelMutationOptions() {
  return mutationOptions({
    mutationFn: (input: CacheModelInput) => Models.CacheModel({
      ...input,
      dryRun: false,
    }),
    // Seed the status query with the returned task so progress shows (and polls) right away.
    onSuccess: (status, input, _onMutateResult, { client }) => {
      const { queryKey } = modelCacheStatusQueryOptions(input.project, input.name, { revision: input.revision })

      client.setQueryData(queryKey, status)
    },
    meta: {
      errorMessage: i18n.t('projects.detail.proxyDownload.serverPull.startError'),
      // Statuses only: invalidating the preview would repeat its upstream dry run.
      invalidates: [modelKeys.lists(), modelCacheKeys.statuses()],
    } satisfies NotificationMeta,
  })
}
