import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import AdminReplicationsIcon from '@/assets/svgs/admin-replications-nav.svg?react'
import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'

export const Route = createFileRoute('/(auth)/admin/replications')({
  component: RouteComponent,
})

function RouteComponent() {
  const { t } = useTranslation()

  return (
    <AdminPageLayout
      icon={AdminReplicationsIcon}
      title={t('admin.replications')}
    />
  )
}
