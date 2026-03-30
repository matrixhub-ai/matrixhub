import { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb.ts'
import { createFileRoute } from '@tanstack/react-router'

import { ModelSettingsPage } from '@/features/models/pages/ModelSettingsPage'
import { ensureProjectAccess } from '@/utils/routerAccess'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId/settings/',
)({
  beforeLoad: async ({ params }) => {
    // only public project and user has no role for the this project,
    // will return 403
    await ensureProjectAccess(
      params.projectId,
      { allowedRoles: [ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN] })
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
