import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import AdminUsersIcon from '@/assets/svgs/admin-users-nav.svg?react'
import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'

export const Route = createFileRoute('/(auth)/admin/users')({
  component: RouteComponent,
})

function RouteComponent() {
  const { t } = useTranslation()

  return (
    <AdminPageLayout
      icon={AdminUsersIcon}
      title={t('admin.users')}
    />
  )
}
