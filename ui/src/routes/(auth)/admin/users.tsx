import { IconUsers as AdminUsersIcon } from '@tabler/icons-react'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'
import { UsersPage } from '@/features/admin/users/pages/UsersPage'
import { usersQueryOptions } from '@/features/admin/users/users.query'
import { usersSearchSchema } from '@/features/admin/users/users.schema'

export const Route = createFileRoute('/(auth)/admin/users')({
  validateSearch: usersSearchSchema,
  loaderDeps: ({ search }) => ({ search }),
  loader: ({
    context: { queryClient },
    deps: { search },
  }) => {
    queryClient.prefetchQuery(usersQueryOptions(search))
  },
  component: RouteComponent,
})

export const Icon = AdminUsersIcon

function RouteComponent() {
  const { t } = useTranslation()

  return (
    <AdminPageLayout
      icon={AdminUsersIcon}
      title={t('admin.users')}
    >
      <UsersPage />
    </AdminPageLayout>
  )
}
