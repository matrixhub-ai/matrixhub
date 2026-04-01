import { Outlet, createFileRoute } from '@tanstack/react-router'

import { modelQueryOptions } from '@/features/models/models.query'
import { ModelDetailPage } from '@/features/models/pages/ModelDetailPage'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId',
)({
  component: ModelDetailLayout,
  loader: async ({
    context, params,
  }) => {
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
