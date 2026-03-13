import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import AdminRegistriesIcon from '@/assets/svgs/admin-registries-nav.svg?react'
import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'

export const Route = createFileRoute('/(auth)/admin/registries')({
  component: RouteComponent,
})

function RouteComponent() {
  const { t } = useTranslation()

  return (
    <AdminPageLayout
      icon={AdminRegistriesIcon}
      title={t('admin.registries')}
    />
  )
}
