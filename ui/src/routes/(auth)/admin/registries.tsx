import { IconBuildingWarehouse as AdminRegistriesIcon } from '@tabler/icons-react'
import { createFileRoute, stripSearchParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AdminPageHeader } from '@/features/admin/components/admin-page-header'
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
  loader: ({
    context: { queryClient },
    deps: { search },
  }) => {
    queryClient.prefetchQuery(registriesQueryOptions(search))
  },
  component: RouteComponent,
})

export const Icon = AdminRegistriesIcon

function RouteComponent() {
  const { t } = useTranslation()

  return (
    <>
      <AdminPageHeader
        icon={AdminRegistriesIcon}
        items={[{ label: t('admin.registries') }]}
      />
      <AdminPageLayout>
        <RegistriesPage />
      </AdminPageLayout>
    </>
  )
}
