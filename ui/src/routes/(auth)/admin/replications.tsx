import { IconAffiliate } from '@tabler/icons-react'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'

export const Route = createFileRoute('/(auth)/admin/replications')({
  component: RouteComponent,
})

export const Icon = IconAffiliate

function RouteComponent() {
  const { t } = useTranslation()

  return (
    <AdminPageLayout
      icon={IconAffiliate}
      title={t('admin.replications')}
    />
  )
}
