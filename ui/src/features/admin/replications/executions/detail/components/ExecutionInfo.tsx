import {
  Card,
  Grid,
  Stack,
  Text,
} from '@mantine/core'
import {
  SyncTaskStatus,
  TriggerType,
  type SyncTask,
} from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import { useTranslation } from 'react-i18next'

import { StatusIndicator } from '@/shared/components/StatusIndicator'
import { formatDateTime } from '@/shared/utils/date'

interface ExecutionInfoProps {
  task?: SyncTask
}

export function ExecutionInfo({ task }: ExecutionInfoProps) {
  const { t } = useTranslation()

  const infoItems = [
    {
      label: 'ID',
      value: task?.id ?? '-',
    },
    {
      label: t('routes.admin.replications.executions.table.status'),
      value: (
        <ExecutionStatusIndicator status={task?.status} />
      ),
    },
    {
      label: t('routes.admin.replications.executions.table.triggerType'),
      value: task?.triggerType === TriggerType.TRIGGER_TYPE_SCHEDULED
        ? t('routes.admin.replications.trigger.scheduled')
        : t('routes.admin.replications.trigger.manual'),
    },
    {
      label: t('routes.admin.replications.executions.table.startedAt'),
      value: formatDateTime(task?.startedTimestamp, 'YYYY/M/D HH:mm'),
    },
  ]

  return (
    <Card withBorder={false} padding="lg" radius="sm" bg="gray.0">
      <Grid gutter="xl">
        {infoItems.map(item => (
          <Grid.Col
            key={item.label}
            span={{
              base: 12,
              sm: 6,
              md: 3,
            }}
          >
            <Stack gap={4}>
              <Text component={typeof item.value === 'object' ? 'div' : undefined} size="lg" fw={500} c="var(--mantine-color-black)">
                {item.value}
              </Text>
              <Text size="sm" c="dimmed">
                {item.label}
              </Text>
            </Stack>
          </Grid.Col>
        ))}
      </Grid>
    </Card>
  )
}

function ExecutionStatusIndicator({ status }: { status?: SyncTaskStatus }) {
  const { t } = useTranslation()

  if (status === SyncTaskStatus.SYNC_TASK_STATUS_RUNNING) {
    return (
      <StatusIndicator
        type="info"
        label={t('routes.admin.replications.executions.status.running')}
      />
    )
  }

  if (status === SyncTaskStatus.SYNC_TASK_STATUS_STOPPED) {
    return (
      <StatusIndicator
        type="warning"
        label={t('routes.admin.replications.executions.status.stopped')}
      />
    )
  }

  if (status === SyncTaskStatus.SYNC_TASK_STATUS_SUCCEEDED) {
    return (
      <StatusIndicator
        type="success"
        label={t('routes.admin.replications.executions.status.succeeded')}
      />
    )
  }

  if (status === SyncTaskStatus.SYNC_TASK_STATUS_FAILED) {
    return (
      <StatusIndicator
        type="danger"
        label={t('routes.admin.replications.executions.status.failed')}
      />
    )
  }

  return (
    <StatusIndicator
      label={t('routes.admin.replications.executions.status.unknown')}
    />
  )
}
