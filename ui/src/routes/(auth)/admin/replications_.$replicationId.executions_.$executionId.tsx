import { Button } from '@mantine/core'
import { SyncTaskStatus } from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import { IconAffiliate } from '@tabler/icons-react'
import { useMutation, useSuspenseQuery } from '@tanstack/react-query'
import { createFileRoute, stripSearchParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AdminPageHeader } from '@/features/admin/components/admin-page-header'
import { AdminPageLayout } from '@/features/admin/components/admin-page-layout'
import {
  adminReplicationExecutionDetailKeys,
  executionDetailQueryOptions,
  syncJobsQueryOptions,
} from '@/features/admin/replications/executions/detail/execution-detail.query'
import {
  executionDetailSearchDefaults,
  executionDetailSearchSchema,
} from '@/features/admin/replications/executions/detail/execution-detail.schema'
import { parseExecutionId } from '@/features/admin/replications/executions/detail/execution-detail.utils'
import { ExecutionDetailPage } from '@/features/admin/replications/executions/detail/pages/ExecutionDetailPage'
import { stopReplicationExecutionMutationOptions } from '@/features/admin/replications/executions/executions.mutation'
import { parseReplicationId } from '@/features/admin/replications/executions/executions.utils'
import { replicationDetailQueryOptions } from '@/features/admin/replications/replications.query'
import { getReplicationDisplayName } from '@/features/admin/replications/replications.utils'
import { queryClient } from '@/queryClient'

export const Route = createFileRoute(
  '/(auth)/admin/replications_/$replicationId/executions_/$executionId',
)({
  validateSearch: executionDetailSearchSchema,
  search: {
    middlewares: [stripSearchParams(executionDetailSearchDefaults)],
  },
  loaderDeps: ({ search }) => ({ search }),
  loader: async ({
    context,
    deps: { search },
    params: {
      replicationId, executionId,
    },
  }) => {
    const syncPolicyId = parseReplicationId(replicationId)
    const syncTaskId = parseExecutionId(executionId)

    context.queryClient.prefetchQuery(syncJobsQueryOptions(syncPolicyId, syncTaskId, search))

    await Promise.all([
      context.queryClient.ensureQueryData(replicationDetailQueryOptions(syncPolicyId)),
      context.queryClient.ensureQueryData(executionDetailQueryOptions(syncPolicyId, syncTaskId)),
    ])
  },
  component: RouteComponent,
})

function RouteComponent() {
  const { t } = useTranslation()
  const {
    replicationId, executionId,
  } = Route.useParams()
  const syncPolicyId = parseReplicationId(replicationId)
  const syncTaskId = parseExecutionId(executionId)
  const { data: syncPolicy } = useSuspenseQuery(replicationDetailQueryOptions(syncPolicyId))
  const { data: task } = useSuspenseQuery(executionDetailQueryOptions(syncPolicyId, syncTaskId))

  const {
    mutate: stopTask, isPending,
  } = useMutation({
    ...stopReplicationExecutionMutationOptions(syncPolicyId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: adminReplicationExecutionDetailKeys.item(syncPolicyId, syncTaskId),
      })
    },
  })

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
            linkProps: {
              to: '/admin/replications/$replicationId/executions',
              params: { replicationId },
            },
          },
          {
            label: t('routes.admin.replications.executions.detail.breadcrumb', {
              id: executionId,
            }),
          },
        ]}
        actions={(
          <Button
            color="cyan"
            onClick={() => stopTask({ syncTaskId })}
            loading={isPending}
            disabled={task?.status !== SyncTaskStatus.SYNC_TASK_STATUS_RUNNING}
          >
            {t('routes.admin.replications.executions.actions.stop')}
          </Button>
        )}
      />
      <AdminPageLayout>
        <ExecutionDetailPage />
      </AdminPageLayout>
    </>
  )
}
