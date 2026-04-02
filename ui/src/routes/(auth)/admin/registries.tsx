import { IconBuildingWarehouse as AdminRegistriesIcon } from '@tabler/icons-react'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'
import { RegistriesPage } from '@/features/admin/registries/pages/RegistriesPage'
import { registriesQueryOptions } from '@/features/admin/registries/registries.query'
import { registriesSearchSchema } from '@/features/admin/registries/registries.schema'

export const Route = createFileRoute('/(auth)/admin/registries')({
  validateSearch: registriesSearchSchema,
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
    <AdminPageLayout
      icon={AdminRegistriesIcon}
      title={t('admin.registries')}
    >
      <RegistriesPage />
    </AdminPageLayout>
  )
}
