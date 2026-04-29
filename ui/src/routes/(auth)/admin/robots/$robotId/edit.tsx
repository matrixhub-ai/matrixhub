import { Button } from '@mantine/core'
import { IconArrowLeft } from '@tabler/icons-react'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'

export const Route = createFileRoute('/(auth)/admin/robots/$robotId/edit')({
  component: RouteComponent,
})

function RouteComponent() {
  const { t } = useTranslation()
  const { robotId } = Route.useParams()

  return (
    <>
      <Button leftSection={<IconArrowLeft />} variant="transparent" color="text/title">{t('routes.admin.robots.editRobot')}</Button>

      <AdminPageLayout>
        robot edit:
        {' '}
        {robotId}
      </AdminPageLayout>
    </>
  )
}
