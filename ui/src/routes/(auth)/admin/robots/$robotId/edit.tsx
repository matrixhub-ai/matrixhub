import { createFileRoute } from '@tanstack/react-router'

import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'
import { RobotAccountEditPage } from '@/features/admin/robots/pages/RobotAccountEditPage'
import { robotAccountDetailQueryOptions } from '@/features/admin/robots/robot.query'

export const Route = createFileRoute('/(auth)/admin/robots/$robotId/edit')({
  loader: async ({
    context: { queryClient },
    params: { robotId },
  }) => {
    await queryClient.ensureQueryData(robotAccountDetailQueryOptions(Number(robotId)))
  },
  component: RouteComponent,
})

function RouteComponent() {
  const { robotId } = Route.useParams()

  return (
    <AdminPageLayout>
      <RobotAccountEditPage robotId={Number(robotId)} />
    </AdminPageLayout>
  )
}
