import { Outlet, createFileRoute } from '@tanstack/react-router'

import { modelQueryOptions } from '@/features/models/models.query'
import { ModelDetailPage } from '@/features/models/pages/ModelDetailPage'
import { projectDetailQueryOptions } from '@/features/projects/projects.query'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId',
)({
  component: ModelDetailLayout,
  loader: async ({
    context, params,
  }) => {
    // The project tells whether the model comes from a proxy, i.e. whether it has a cache status.
    void context.queryClient.prefetchQuery(projectDetailQueryOptions(params.projectId))
    const model = await context.queryClient.ensureQueryData(modelQueryOptions(params.projectId, params.modelId))

    return { model }
  },
})

function ModelDetailLayout() {
  return (
    <ModelDetailPage>
      <Outlet />
    </ModelDetailPage>
  )
}
