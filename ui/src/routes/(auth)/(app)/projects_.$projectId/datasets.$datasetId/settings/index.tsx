import { createFileRoute } from '@tanstack/react-router'

import { ensureProjectAccess } from '@/utils/routerAccess'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/datasets/$datasetId/settings/',
)({
  beforeLoad: async ({ params }) => {
    await ensureProjectAccess(params.projectId)
  },
  component: RouteComponent,
})

function RouteComponent() {
  const {
    projectId, datasetId,
  } = Route.useParams()

  return (
    <div>
      Dataset Settings Form
      <br />
      Project ID:
      {' '}
      {projectId}
      <br />
      Dataset ID:
      {' '}
      {datasetId}
    </div>
  )
}
