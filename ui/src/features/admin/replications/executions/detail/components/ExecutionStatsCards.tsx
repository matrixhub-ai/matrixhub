import {
  Card,
  Grid,
  Group,
  Stack,
  Text,
  ThemeIcon,
  rem,
} from '@mantine/core'
import { type SyncTask } from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import {
  IconCircleCheck,
  IconAlertCircle,
  IconMessageCircleBolt,
  IconAlertTriangle,
} from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

interface ExecutionStatsCardsProps {
  task?: SyncTask
}

export function ExecutionStatsCards({ task }: ExecutionStatsCardsProps) {
  const { t } = useTranslation()

  const safeNumber = (value: string | number | undefined) => Number(value ?? 0) || 0
  const total = safeNumber(task?.totalItems)
  const success = safeNumber(task?.successfulItems)
  const failed = safeNumber(task?.failedItems)
  const stopped = safeNumber(task?.stoppedItems)
  const running = Math.max(0, total - success - failed - stopped)

  const stats = [
    {
      label: t('routes.admin.replications.executions.detail.status.succeeded'),
      value: success,
      icon: IconCircleCheck,
      color: 'var(--mantine-color-teal-6)',
      bgColor: 'var(--mantine-color-teal-light)',
    },
    {
      label: t('routes.admin.replications.executions.detail.status.failed'),
      value: failed,
      icon: IconAlertCircle,
      color: 'var(--mantine-color-red-6)',
      bgColor: 'var(--mantine-color-red-light)',
    },
    {
      label: t('routes.admin.replications.executions.detail.status.running'),
      value: running,
      icon: IconMessageCircleBolt,
      color: 'var(--mantine-primary-color-filled)',
      bgColor: 'var(--mantine-primary-color-light)',
    },
    {
      label: t('routes.admin.replications.executions.detail.status.stopped'),
      value: stopped,
      icon: IconAlertTriangle,
      color: 'var(--mantine-color-yellow-6)',
      bgColor: 'var(--mantine-color-orange-light)',
    },
  ]

  return (
    <Grid gutter="md">
      {stats.map(stat => (
        <Grid.Col
          key={stat.label}
          span={{
            base: 12,
            sm: 6,
            md: 3,
          }}
        >
          <Card
            padding="lg"
            radius="sm"
            style={{
              backgroundColor: stat.bgColor,
              border: 'none',
            }}
          >
            <Group justify="space-between" align="flex-start">
              <Stack gap={4}>
                <ThemeIcon
                  variant="transparent"
                  color={stat.color}
                  size={rem(32)}
                >
                  <stat.icon size={rem(32)} />
                </ThemeIcon>
                <Text size="sm" c="dimmed" fw={500}>
                  {stat.label}
                </Text>
              </Stack>
              <Text
                fw={700}
                style={{
                  fontSize: rem(48),
                  lineHeight: 1,
                }}
              >
                {stat.value}
              </Text>
            </Group>
          </Card>
        </Grid.Col>
      ))}
    </Grid>
  )
}
