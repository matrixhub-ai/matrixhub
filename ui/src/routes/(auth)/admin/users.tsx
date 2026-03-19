import { Alert } from '@mantine/core'
import { IconUsers as AdminUsersIcon } from '@tabler/icons-react'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import z from 'zod'

import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'
import { UsersPage } from '@/features/admin/users/pages/UsersPage'
import { adminUsersQueryOptions } from '@/features/admin/users/users.query'

const DEFAULT_USERS_PAGE = 1

const usersSearchSchema = z.object({
  page: z.coerce.number().int().gte(1).catch(DEFAULT_USERS_PAGE),
  query: z.string().transform(value => value.trim()).catch(''),
})

export const Route = createFileRoute('/(auth)/admin/users')({
  validateSearch: usersSearchSchema,
  loaderDeps: ({ search }) => search,
  loader: async ({ context, deps }) => {
    await context.queryClient.ensureQueryData(adminUsersQueryOptions(deps))
  },
  component: RouteComponent,
  errorComponent: ({ error }) => <UsersRouteErrorState error={error} />,
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

function UsersRouteErrorState({ error }: { error: unknown }) {
  const { t } = useTranslation()

  return (
    <AdminPageLayout
      icon={AdminUsersIcon}
      title={t('admin.users')}
    >
      <Alert
        color="red"
        title={t('routes.admin.users.errors.loadTitle')}
        variant="light"
      >
        {getErrorMessage(error)}
      </Alert>
    </AdminPageLayout>
  )
}

function getErrorMessage(error: unknown) {
  if (error instanceof Error) {
    return error.message
  }

  return String(error)
}
