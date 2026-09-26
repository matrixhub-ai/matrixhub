import {
  Badge,
  Box,
  Button,
  Group,
  Loader,
  ScrollArea,
  Stack,
  Table,
  Text,
  Timeline,
  Tooltip,
} from '@mantine/core'
import {
  IconAlertTriangle,
  IconCircleCheck,
  IconClock,
  IconRefresh,
  IconShield,
  IconShieldX,
  IconX,
} from '@tabler/icons-react'
import {
  useMutation, useQuery, useQueryClient,
} from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import {
  fetchScanAudit,
  fetchScanReport,
  postRescan,
  type ScanReport,
  type ScanStatus,
} from '@/features/models/security/scan.api'

const STATUS_COLOR: Record<ScanStatus, string> = {
  unscanned: 'gray',
  scanning: 'blue',
  pass: 'green',
  warning: 'yellow',
  blocked: 'red',
  failed: 'orange',
}

const SEVERITY_COLOR: Record<string, string> = {
  clean: 'green',
  info: 'gray',
  warning: 'yellow',
  critical: 'red',
}

function StatusBadge({ status }: { status: ScanStatus }) {
  const { t } = useTranslation()

  return (
    <Badge color={STATUS_COLOR[status] ?? 'gray'} variant="light" size="lg" leftSection={<IconShield size={14} />}>
      {t(`model.security.status.${status}`)}
    </Badge>
  )
}

function ScanSummary({ report }: { report: ScanReport }) {
  const { t } = useTranslation()
  const counts = report.counts ?? {}

  return (
    <Group gap="lg">
      <StatusBadge status={report.status} />
      {report.scannedAt && (
        <Text size="sm" c="dimmed">
          {t('model.security.scannedAt')}
          :
          {new Date(report.scannedAt).toLocaleString()}
        </Text>
      )}
      {Object.entries(counts).map(([sev, n]) => (
        <Badge key={sev} color={SEVERITY_COLOR[sev] ?? 'gray'} variant="outline">
          {sev}
          :
          {n}
        </Badge>
      ))}
      {report.scannerVersions && (
        <Tooltip label={Object.entries(report.scannerVersions).map(([k, v]) => `${k}: ${v}`).join('\n')}>
          <Text size="xs" c="dimmed">
            {t('model.security.scanners')}
            :
            {Object.keys(report.scannerVersions).join(', ')}
          </Text>
        </Tooltip>
      )}
    </Group>
  )
}

function FindingsTable({ report }: { report: ScanReport }) {
  const { t } = useTranslation()
  const files = report.files ?? []
  const flagged = files.filter(f => f.severity !== 'clean')
  const rows = (flagged.length > 0 ? flagged : files)

  if (rows.length === 0) {
    return (
      <Text size="sm" c="dimmed" mt="md">
        {t('model.security.noFindingsYet')}
      </Text>
    )
  }

  return (
    <ScrollArea mt="md">
      <Table striped highlightOnHover withTableBorder maw={900}>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>{t('model.security.file')}</Table.Th>
            <Table.Th>{t('model.security.severity')}</Table.Th>
            <Table.Th>{t('model.security.rules')}</Table.Th>
            <Table.Th>{t('model.security.digest')}</Table.Th>
            <Table.Th>{t('model.security.size')}</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {rows.map(f => (
            <Table.Tr key={f.path}>
              <Table.Td style={{ whiteSpace: 'nowrap' }}>{f.path}</Table.Td>
              <Table.Td>
                <Badge color={SEVERITY_COLOR[f.severity] ?? 'gray'} variant="light">
                  {f.severity}
                </Badge>
              </Table.Td>
              <Table.Td>{(f.rules ?? []).join(', ') || '—'}</Table.Td>
              <Table.Td>
                <Tooltip label={f.digest}>
                  <span style={{
                    fontFamily: 'monospace',
                    fontSize: 12,
                  }}
                  >
                    {f.digest.slice(0, 12)}
                  </span>
                </Tooltip>
              </Table.Td>
              <Table.Td>{f.size}</Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </ScrollArea>
  )
}

function AuditTimeline({ name }: { name: string }) {
  const { t } = useTranslation()
  const q = useQuery({
    queryKey: ['scan', 'audit', name],
    queryFn: () => fetchScanAudit(name, 12),
  })

  if (q.isLoading) {
    return <Loader size="sm" />
  }
  const events = q.data?.events ?? []

  if (events.length === 0) {
    return <Text size="sm" c="dimmed">{t('model.security.noAudit')}</Text>
  }

  return (
    <Timeline bulletSize={18} lineWidth={1} mt="sm">
      {events.map(e => (
        <Timeline.Item
          key={e.id}
          bullet={e.decision.startsWith('deny') || e.decision.includes('blocked') || e.decision.includes('failed')
            ? <IconShieldX size={12} />
            : <IconCircleCheck size={12} />}
          title={`${e.action} · ${e.decision}`}
        >
          <Text size="xs" c="dimmed">
            {e.actor}
            {' '}
            ·
            {new Date(e.createdAt).toLocaleString()}
            {e.reason ? ` · ${e.reason}` : ''}
          </Text>
        </Timeline.Item>
      ))}
    </Timeline>
  )
}

export function ModelSecurityPage({
  project,
  name,
  revision,
}: {
  project: string
  name: string
  revision: string
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const repoId = `${project}/${name}`

  const q = useQuery({
    queryKey: ['scan', 'report', repoId, revision],
    queryFn: () => fetchScanReport(repoId, revision),
    refetchInterval: (query) => {
      const st = query.state.data?.status

      return st === 'scanning' || st === 'unscanned' ? 3000 : false
    },
  })

  const rescan = useMutation({
    mutationFn: () => postRescan(repoId, revision),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['scan'] })
    },
  })

  if (q.isLoading) {
    return (
      <Box
        py={40}
        style={{
          display: 'flex',
          justifyContent: 'center',
        }}
      >
        <Loader />
      </Box>
    )
  }
  if (q.isError) {
    return (
      <Text c="red" py={20}>
        {t('model.security.loadFailed')}
        :
        {q.error.message}
      </Text>
    )
  }
  const report = q.data
  const scanning = report?.status === 'scanning' || report?.status === 'unscanned'

  return (
    <Stack gap="md" pt="md" maw={1000}>
      <Group justify="space-between">
        <Group gap="md">
          {scanning && <IconClock size={20} />}
          {report && <ScanSummary report={report} />}
        </Group>
        <Button
          variant="light"
          color="cyan"
          leftSection={rescan.isPending ? <Loader size={14} /> : <IconRefresh size={16} />}
          onClick={() => rescan.mutate()}
          disabled={rescan.isPending || scanning}
        >
          {t('model.security.rescan')}
        </Button>
      </Group>

      {report?.status === 'failed' && report.error && (
        <Group gap="xs" c="orange">
          <IconAlertTriangle size={16} />
          <Text size="sm">{report.error}</Text>
        </Group>
      )}
      {report?.status === 'blocked' && (
        <Group gap="xs" c="red">
          <IconX size={16} />
          <Text size="sm">{t('model.security.blockedNotice')}</Text>
        </Group>
      )}

      <Box>
        <Text fw={500} mb={4}>{t('model.security.findings')}</Text>
        {report && <FindingsTable report={report} />}
      </Box>

      <Box>
        <Text fw={500} mb={4}>{t('model.security.audit')}</Text>
        <AuditTimeline name={name} />
      </Box>
    </Stack>
  )
}
