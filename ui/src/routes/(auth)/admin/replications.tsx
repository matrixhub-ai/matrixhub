import { IconAffiliate } from '@tabler/icons-react'
import { createFileRoute, stripSearchParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AdminPageHeader } from '@/features/admin/components/admin-page-header'
import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'
import { ReplicationsPage } from '@/features/admin/replications/pages/ReplicationsPage'
import { replicationsQueryOptions } from '@/features/admin/replications/replications.query'
import { replicationsSearchDefaults, replicationsSearchSchema } from '@/features/admin/replications/replications.schema'

export const Route = createFileRoute('/(auth)/admin/replications')({
  validateSearch: replicationsSearchSchema,
  search: {
    middlewares: [stripSearchParams(replicationsSearchDefaults)],
  },
  loaderDeps: ({ search }) => ({ search }),
  loader: ({
    context: { queryClient },
    deps: { search },
  }) => {
    queryClient.prefetchQuery(replicationsQueryOptions(search))
  },
  component: RouteComponent,
})

export const Icon = IconAffiliate

function RouteComponent() {
  const { t } = useTranslation()

  return (
    <>
      <AdminPageHeader
        icon={IconAffiliate}
        items={[{ label: t('admin.replications') }]}
      />
      <AdminPageLayout>
        <ReplicationsPage />
      </AdminPageLayout>
    </>
  )
}
