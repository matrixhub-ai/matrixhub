import { IconAffiliate } from '@tabler/icons-react'
import { createFileRoute, stripSearchParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'
import { replicationExecutionsQueryOptions } from '@/features/admin/replications/executions/executions.query'
import {
  replicationExecutionsSearchDefaults,
  replicationExecutionsSearchSchema,
} from '@/features/admin/replications/executions/executions.schema'
import { parseReplicationId } from '@/features/admin/replications/executions/executions.utils'
import { ReplicationExecutionsPage } from '@/features/admin/replications/executions/pages/ReplicationExecutionsPage'
import { replicationDetailQueryOptions } from '@/features/admin/replications/replications.query'

export const Route = createFileRoute(
  '/(auth)/admin/replications_/$replicationId/executions',
)({
  validateSearch: replicationExecutionsSearchSchema,
  search: {
    middlewares: [stripSearchParams(replicationExecutionsSearchDefaults)],
  },
  loaderDeps: ({ search }) => ({ search }),
  loader: async ({
    context: { queryClient },
    deps: { search },
    params: { replicationId },
  }) => {
    const syncPolicyId = parseReplicationId(replicationId)

    await Promise.allSettled([
      queryClient.ensureQueryData(replicationDetailQueryOptions(syncPolicyId)),
      queryClient.ensureQueryData(replicationExecutionsQueryOptions(syncPolicyId, search)),
    ])
  },
  component: RouteComponent,
})

function RouteComponent() {
  const { t } = useTranslation()

  return (
    <AdminPageLayout
      icon={IconAffiliate}
      title={t('admin.replications')}
    >
      <ReplicationExecutionsPage />
    </AdminPageLayout>
  )
}
