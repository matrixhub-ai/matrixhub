import { useQuery } from '@tanstack/react-query'

import { projectDetailQueryOptions } from '@/features/projects/projects.query'
import { isProxyProject } from '@/features/projects/projects.utils'

import { modelCacheStatusQueryOptions } from '../cache.query'

/**
 * Local cache status of the default branch of a proxy-project model; undefined for models of
 * local projects, while loading, and when the status cannot be read. Read-only: it never
 * contacts upstream or starts downloads, and polls only while a server-side pull runs.
 */
export function useProxyModelCacheStatus(projectId: string, modelName: string) {
  const { data: project } = useQuery(projectDetailQueryOptions(projectId))
  const isProxy = isProxyProject(project?.registryUrl)
  const { data: status } = useQuery({
    ...modelCacheStatusQueryOptions(projectId, modelName, { revision: '' }),
    enabled: isProxy,
  })

  return isProxy ? status : undefined
}
