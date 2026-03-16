import { createFileRoute } from '@tanstack/react-router'

import { ensureProjectAccess } from '@/utils/routerAccess'

export const Route = createFileRoute(
  '/(auth)/(app)/projects/$projectId/settings/',
)({
  beforeLoad: async ({ params }) => {
    await ensureProjectAccess(params.projectId)
  },
  component: RouteComponent,
})

function RouteComponent() {
  const { projectId } = Route.useParams()

  return (
    <div>
      Project Settings Page - Project ID:
      {projectId}
    </div>
  )
}
