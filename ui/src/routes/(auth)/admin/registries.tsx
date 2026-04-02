import { IconBuildingWarehouse as AdminRegistriesIcon } from '@tabler/icons-react'
import { createFileRoute, stripSearchParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'
import { RegistriesPage } from '@/features/admin/registries/pages/RegistriesPage'
import { registriesQueryOptions } from '@/features/admin/registries/registries.query'
import { registriesSearchDefaults, registriesSearchSchema } from '@/features/admin/registries/registries.schema'

export const Route = createFileRoute('/(auth)/admin/registries')({
  validateSearch: registriesSearchSchema,
  search: {
    middlewares: [stripSearchParams(registriesSearchDefaults)],
  },
  loaderDeps: ({ search }) => ({ search }),
  loader: async ({
    context: { queryClient },
    deps: { search },
  }) => {
    await queryClient.ensureQueryData(registriesQueryOptions(search))
  },
  component: RouteComponent,
})

export const Icon = AdminRegistriesIcon

function RouteComponent() {
  const { t } = useTranslation()

  return (
    <AdminPageLayout
      icon={AdminRegistriesIcon}
      title={t('admin.registries')}
    >
      <RegistriesPage />
    </AdminPageLayout>
  )
}
