import { IconAffiliate } from '@tabler/icons-react'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'
import { ReplicationsPage } from '@/features/admin/replications/pages/ReplicationsPage'
import { replicationsQueryOptions } from '@/features/admin/replications/replications.query'
import { replicationsSearchSchema } from '@/features/admin/replications/replications.schema'

export const Route = createFileRoute('/(auth)/admin/replications')({
  validateSearch: replicationsSearchSchema,
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
    <AdminPageLayout
      icon={IconAffiliate}
      title={t('admin.replications')}
    >
      <ReplicationsPage />
    </AdminPageLayout>
  )
}
