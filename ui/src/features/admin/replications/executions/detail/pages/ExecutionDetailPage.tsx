import {
  Stack,
  Text,
  Group,
  rem,
} from '@mantine/core'
import { useQueryClient, useSuspenseQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { useRouteListState } from '@/shared/hooks/useRouteListState'
import { DEFAULT_PAGE } from '@/utils/constants'

import { ExecutionInfo } from '../components/ExecutionInfo'
import { ExecutionStatsCards } from '../components/ExecutionStatsCards'
import { SyncJobTable } from '../components/SyncJobTable'
import {
  adminReplicationExecutionDetailKeys,
  executionDetailQueryOptions,
  useSyncJobs,
} from '../execution-detail.query'
import { parseExecutionId } from '../execution-detail.utils'

const executionDetailRouteApi = getRouteApi('/(auth)/admin/replications_/$replicationId/executions_/$executionId')

function SectionTitle({ title }: { title: string }) {
  return (
    <Group
      gap={8}
      mb="md"
    >
      <div style={{
        width: rem(4),
        height: rem(16),
        backgroundColor: 'var(--mantine-color-cyan-6)',
        borderRadius: rem(2),
      }}
      />
      <Text
        fw={600}
        size="sm"
      >
        {title}
      </Text>
    </Group>
  )
}

export function ExecutionDetailPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const navigate = executionDetailRouteApi.useNavigate()
  const search = executionDetailRouteApi.useSearch()
  const {
    replicationId, executionId,
  } = executionDetailRouteApi.useParams()
  const syncPolicyId = Number(replicationId)
  const syncTaskId = parseExecutionId(executionId)

  const { data: task } = useSuspenseQuery(
    executionDetailQueryOptions(syncPolicyId, syncTaskId),
  )

  const {
    data: jobsData,
    isLoading,
    isFetching,
  } = useSyncJobs(syncPolicyId, syncTaskId, search)

  const jobs = jobsData?.jobs ?? []
  const pagination = jobsData?.pagination

  const refreshJobs = () => {
    queryClient.invalidateQueries({
      queryKey: adminReplicationExecutionDetailKeys.item(syncPolicyId, syncTaskId),
    })
  }

  const {
    onRefresh,
    onPageChange,
    routeColumnFilterTableOptions,
  } = useRouteListState({
    search,
    navigate,
    records: jobs,
    getRecordId: job => String(job.id ?? '-'),
    refresh: refreshJobs,
    columnFilterSync: [
      {
        columnId: 'status',
        searchKey: 'status',
      },
      {
        columnId: 'resourceType',
        searchKey: 'resourceType',
      },
    ],
  })

  return (
    <Stack gap="xl">
      <Stack gap="sm">
        <SectionTitle title={t('routes.admin.replications.executions.detail.basicInfo')} />
        <ExecutionInfo task={task} />
      </Stack>

      <Stack gap="sm">
        <SectionTitle title={t('routes.admin.replications.executions.detail.statusDistribution')} />
        <ExecutionStatsCards task={task} />
      </Stack>

      <Stack gap="sm">
        <SectionTitle title={t('routes.admin.replications.executions.detail.subTasks')} />
        <SyncJobTable
          syncPolicyId={syncPolicyId}
          syncTaskId={syncTaskId}
          data={jobs}
          pagination={pagination}
          loading={isLoading}
          fetching={isFetching}
          page={search.page ?? DEFAULT_PAGE}
          onRefresh={onRefresh}
          onPageChange={onPageChange}
          tableOptions={routeColumnFilterTableOptions}
        />
      </Stack>
    </Stack>
  )
}
