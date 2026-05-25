import { IconAffiliate } from '@tabler/icons-react'
import { useSuspenseQuery } from '@tanstack/react-query'
import { createFileRoute, stripSearchParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AdminPageHeader } from '@/features/admin/components/admin-page-header'
import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'
import { replicationExecutionsQueryOptions } from '@/features/admin/replications/executions/executions.query'
import {
  replicationExecutionsSearchDefaults,
  replicationExecutionsSearchSchema,
} from '@/features/admin/replications/executions/executions.schema'
import { parseReplicationId } from '@/features/admin/replications/executions/executions.utils'
import { ReplicationExecutionsPage } from '@/features/admin/replications/executions/pages/ReplicationExecutionsPage'
import { replicationDetailQueryOptions } from '@/features/admin/replications/replications.query'
import { getReplicationDisplayName } from '@/features/admin/replications/replications.utils'

export const Route = createFileRoute(
  '/(auth)/admin/replications_/$replicationId/executions',
)({
  validateSearch: replicationExecutionsSearchSchema,
  search: {
    middlewares: [stripSearchParams(replicationExecutionsSearchDefaults)],
  },
  loaderDeps: ({ search }) => ({ search }),
  loader: async ({
    context,
    deps: { search },
    params: { replicationId },
  }) => {
    const syncPolicyId = parseReplicationId(replicationId)

    await context.queryClient.ensureQueryData(replicationDetailQueryOptions(syncPolicyId))
    context.queryClient.prefetchQuery(replicationExecutionsQueryOptions(syncPolicyId, search))
  },
  component: RouteComponent,
})

function RouteComponent() {
  const { t } = useTranslation()
  const { replicationId } = Route.useParams()
  const syncPolicyId = parseReplicationId(replicationId)
  const { data: syncPolicy } = useSuspenseQuery(replicationDetailQueryOptions(syncPolicyId))

  return (
    <>
      <AdminPageHeader
        icon={IconAffiliate}
        items={[
          {
            label: t('admin.replications'),
            linkProps: { to: '/admin/replications' },
          },
          {
            label: t('routes.admin.replications.executions.breadcrumb', {
              name: getReplicationDisplayName(syncPolicy),
            }),
          },
        ]}
      />
      <AdminPageLayout>
        <ReplicationExecutionsPage />
      </AdminPageLayout>
    </>
  )
}
