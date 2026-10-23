import { Models } from '@matrixhub/api-ts/v1alpha1/model.pb'
import { queryOptions } from '@tanstack/react-query'

import { getCacheStatusPollInterval } from './cache.utils'

import type { NotificationMeta } from '@/types/tanstack-query'

/** Which revision a cache status or pull covers; it includes every file of the revision. */
export interface ModelCacheSelection {
  /** Branch, tag or commit; empty means the upstream default branch. */
  revision: string
}

// -- Query key factory --

export const modelCacheKeys = {
  all: ['modelCache'] as const,
  statuses: () => [...modelCacheKeys.all, 'status'] as const,
  status: (projectId: string, modelName: string, params: ModelCacheSelection) => [
    ...modelCacheKeys.statuses(),
    projectId,
    modelName,
    params,
  ] as const,
  previews: () => [...modelCacheKeys.all, 'preview'] as const,
  preview: (projectId: string, modelName: string, params: ModelCacheSelection) => [
    ...modelCacheKeys.previews(),
    projectId,
    modelName,
    params,
  ] as const,
}

// -- Query options factory --

// Local cache status of a model; never contacts upstream. Polls while a pull task runs.
// Callers render errors inline or hide the status, so global notifications are skipped.
export function modelCacheStatusQueryOptions(
  projectId: string,
  modelName: string,
  params: ModelCacheSelection,
) {
  return queryOptions({
    queryKey: modelCacheKeys.status(projectId, modelName, params),
    queryFn: () => Models.GetModelCache({
      project: projectId,
      name: modelName,
      revision: params.revision,
    }),
    refetchInterval: query => getCacheStatusPollInterval(query.state.data, query.state.error),
    meta: {
      skipNotification: true,
    } satisfies NotificationMeta,
  })
}

// Dry run of a server-side pull: lists the upstream files of the revision and what is
// already cached. Read-only and side-effect free (it never creates the model), so it can
// run as a query while the user types.
export function modelCachePreviewQueryOptions(
  projectId: string,
  modelName: string,
  params: ModelCacheSelection,
) {
  return queryOptions({
    queryKey: modelCacheKeys.preview(projectId, modelName, params),
    queryFn: () => Models.CacheModel({
      project: projectId,
      name: modelName,
      revision: params.revision,
      dryRun: true,
    }),
    meta: {
      skipNotification: true,
    } satisfies NotificationMeta,
  })
}
