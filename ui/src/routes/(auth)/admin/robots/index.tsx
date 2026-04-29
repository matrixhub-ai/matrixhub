import { IconRobot } from '@tabler/icons-react'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AdminPageHeader } from '@/features/admin/components/admin-page-header'
import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'

export const Route = createFileRoute('/(auth)/admin/robots/')({
  component: RouteComponent,
})

export const Icon = IconRobot

function RouteComponent() {
  const { t } = useTranslation()

  return (
    <>
      <AdminPageHeader
        icon={IconRobot}
        items={[{ label: t('admin.robots') }]}
      />
      <AdminPageLayout>
        robot list
      </AdminPageLayout>
    </>
  )
}
