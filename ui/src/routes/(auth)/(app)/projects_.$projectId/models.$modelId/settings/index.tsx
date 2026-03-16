import { createFileRoute } from '@tanstack/react-router'

import { ModelSettingsPage } from '@/features/models/pages/ModelSettingsPage'
import { ensureProjectAccess } from '@/utils/routerAccess'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId/settings/',
)({
  beforeLoad: async ({ params }) => {
    await ensureProjectAccess(params.projectId)
  },
  component: ModelSettings,
})

function ModelSettings() {
  const {
    projectId, modelId,
  } = Route.useParams()

  return (
    <ModelSettingsPage
      projectId={projectId}
      modelId={modelId}
    />
  )
}
