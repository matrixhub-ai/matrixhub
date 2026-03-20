import { IconBuildingWarehouse as AdminRegistriesIcon } from '@tabler/icons-react'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'

export const Route = createFileRoute('/(auth)/admin/registries')({
  component: RouteComponent,
})

export const Icon = AdminRegistriesIcon

function RouteComponent() {
  const { t } = useTranslation()

  return (
    <AdminPageLayout
      icon={AdminRegistriesIcon}
      title={t('admin.registries')}
    />
  )
}
