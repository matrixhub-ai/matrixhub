import { createFileRoute } from '@tanstack/react-router'

import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'
import { RobotAccountCreatePage } from '@/features/admin/robots/pages/RobotAccountCreatePage'

export const Route = createFileRoute('/(auth)/admin/robots/new')({
  component: RouteComponent,
})

function RouteComponent() {
  return (
    <AdminPageLayout>
      <RobotAccountCreatePage />
    </AdminPageLayout>
  )
}
