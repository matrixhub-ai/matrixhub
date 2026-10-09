import {
  Badge, Button, Group, Stack, Text,
} from '@mantine/core'
import {
  type ScanAudit, type ScanFile, type ScanStatus,
} from '@matrixhub/api-ts/security'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyValueButton } from '@/shared/components/CopyValueButton'
import { DataTable } from '@/shared/components/DataTable'
import { TruncatedText } from '@/shared/components/TruncatedText'

import { securityAuditActorSchema, securityPolicySchema } from '../security.schema'

import type { MRT_ColumnDef, MRT_Row } from 'mantine-react-table'

function FilePathCell({ row }: { row: MRT_Row<ScanFile> }) {
  return <TruncatedText value={row.original.path} />
}

function FileStatusCell({ row }: { row: MRT_Row<ScanFile> }) {
  return <SecurityBadge status={row.original.status} />
}

function FileSizeCell({ row }: { row: MRT_Row<ScanFile> }) {
  return (
    <Text size="sm">
      {(row.original.size / 1024 / 1024).toFixed(2)}
      {' '}
      MiB
    </Text>
  )
}

function FileReuseCell({ row }: { row: MRT_Row<ScanFile> }) {
  const { t } = useTranslation()

  return <Text size="sm">{t(row.original.reused ? 'security.reused' : 'security.scanned')}</Text>
}

function AuditTimeCell({ row }: { row: MRT_Row<ScanAudit> }) {
  return <Text size="sm">{new Date(row.original.at).toLocaleString()}</Text>
}

function AuditActorCell({ row }: { row: MRT_Row<ScanAudit> }) {
  const { t } = useTranslation()
  const actor = row.original.actor
  let label = actor || t('security.auditAnonymous')

  if (actor.startsWith('{')) {
    try {
      const parsed = securityAuditActorSchema.safeParse(JSON.parse(actor))

      if (parsed.success) {
        label = parsed.data.payload.name
      }
    } catch {
      // Older audit records may contain plain text; preserve their display.
    }
  } else if (actor.startsWith('system:')) {
    label = t(`security.auditActors.${actor.slice(7)}`, { defaultValue: actor })
  }

  return <TruncatedText value={label} />
}

function AuditActionCell({ row }: { row: MRT_Row<ScanAudit> }) {
  const { t } = useTranslation()
  const action = row.original.action
  const key = action.replace(/[:.-]/g, '_')
  const label = t(`security.auditActions.${key}`, { defaultValue: t(`security.status.${action}`, { defaultValue: action }) })

  return <TruncatedText value={label} />
}

function AuditRevisionCell({ row }: { row: MRT_Row<ScanAudit> }) {
  return <TruncatedText value={row.original.revision} />
}

function AuditDetailCell({ row }: { row: MRT_Row<ScanAudit> }) {
  const { t } = useTranslation()
  const detail = row.original.detail
  let label = t(`security.auditDecisions.${detail.replace(/[:.-]/g, '_')}`, { defaultValue: detail })

  if (row.original.action === 'policy-updated') {
    try {
      const parsed = securityPolicySchema.safeParse(JSON.parse(detail))

      if (parsed.success) {
        label = t('security.auditPolicySummary', {
          threshold: t(`security.${parsed.data.block_severity}`),
          pending: t(`security.auditDecision.${parsed.data.on_pending}`),
          failure: t(`security.auditDecision.${parsed.data.on_failure}`),
        })
      }
    } catch {
      // Preserve an unrecognized historical policy record for inspection.
    }
  }

  return <TruncatedText value={label} />
}

export function SecurityBadge({ status }: { status: ScanStatus }) {
  const { t } = useTranslation()
  const color = status === 'passed' ? 'green' : status === 'blocked' ? 'red' : status === 'warning' ? 'yellow' : 'gray'

  return <Badge color={color} variant="light">{t(`security.status.${status}`)}</Badge>
}

export function SecurityFilesTable({
  files, onInspect,
}: { files: ScanFile[]
  onInspect: (file: ScanFile) => void }) {
  const { t } = useTranslation()
  const columns = useMemo<MRT_ColumnDef<ScanFile>[]>(() => [
    {
      accessorKey: 'path',
      header: t('security.file'),
      size: 280,
      Cell: FilePathCell,
    },
    {
      accessorKey: 'status',
      header: t('security.result'),
      size: 140,
      grow: false,
      Cell: FileStatusCell,
    },
    {
      accessorKey: 'size',
      header: t('security.size'),
      size: 130,
      grow: false,
      Cell: FileSizeCell,
    },
    {
      id: 'reuse',
      header: t('security.reuse'),
      size: 120,
      grow: false,
      Cell: FileReuseCell,
    },
  ], [t])

  return <DataTable data={files} columns={columns} emptyTitle={t('security.noFiles')} enableRowActions renderRowActions={({ row }) => <Button variant="subtle" size="xs" onClick={() => onInspect(row.original)}>{t('security.details')}</Button>} tableOptions={{ mantineTableContainerProps: { mah: 380 } }} />
}

export function SecurityFileDetails({ file }: { file: ScanFile }) {
  const { t } = useTranslation()

  return (
    <Stack gap="md">
      <Text fw={600} style={{ overflowWrap: 'anywhere' }}>{file.path}</Text>
      <SecurityBadge status={file.status} />
      <Group wrap="nowrap">
        <Text size="sm" style={{ overflowWrap: 'anywhere' }}>{file.sha256}</Text>
        <CopyValueButton value={file.sha256} />
      </Group>
      <Group>
        <Text size="sm">
          {t('security.fileType')}
          :
          {' '}
          {t(`security.types.${file.file_type ?? 'unknown'}`, { defaultValue: file.file_type ?? t('security.types.unknown') })}
        </Text>
        {file.checked_at && !file.checked_at.startsWith('0001-') && (
          <Text size="sm" c="dimmed">
            {t('security.checkedAt')}
            :
            {' '}
            {new Date(file.checked_at).toLocaleString()}
          </Text>
        )}
      </Group>
      {file.recommended_action && <Text size="sm">{t(`security.recommendations.${file.recommended_action}`)}</Text>}
      {file.error && <Text c="red" size="sm">{t(`security.errors.${file.error}`, { defaultValue: t('security.errors.scanner_incomplete') })}</Text>}
      <Text fw={600}>{t('security.scanners')}</Text>
      {(file.checks ?? []).map(check => <Text key={check} size="sm">{check === 'pickle:not-applicable' ? t('security.pickleNotApplicable') : check === 'pytorch:raw-tensor-storage' ? t('security.rawTensorStorage') : check}</Text>)}
      <Text fw={600}>{t('security.findings')}</Text>
      {(file.findings ?? []).length === 0
        ? <Text size="sm" c="dimmed">{t(file.status === 'passed' ? 'security.noFindings' : 'security.incompleteFindings')}</Text>
        : Array.from(new Map((file.findings ?? []).map(f => [`${f.scanner}:${f.rule}:${f.version}`, f])).values()).map(finding => (
            <Stack key={`${finding.scanner}:${finding.rule}:${finding.version}`} gap="xs">
              <Text size="sm" fw={600}>{finding.rule === 'ML_CONSTRUCTION_REVIEW' ? t('security.mlConstructionReview') : finding.rule}</Text>
              <Text size="sm">
                {finding.scanner}
                {' '}
                /
                {' '}
                {finding.version}
                {' '}
                /
                {' '}
                {finding.severity}
              </Text>
            </Stack>
          ))}
    </Stack>
  )
}

export function SecurityAuditTable({ events }: { events: ScanAudit[] }) {
  const { t } = useTranslation()
  const columns = useMemo<MRT_ColumnDef<ScanAudit>[]>(() => [
    {
      accessorKey: 'at',
      header: t('security.time'),
      size: 190,
      grow: false,
      Cell: AuditTimeCell,
    },
    {
      accessorKey: 'actor',
      header: t('security.actor'),
      enableColumnFilter: true,
      Cell: AuditActorCell,
    },
    {
      accessorKey: 'action',
      header: t('security.action'),
      enableColumnFilter: true,
      Cell: AuditActionCell,
    },
    {
      accessorKey: 'revision',
      header: t('security.revision'),
      enableColumnFilter: true,
      Cell: AuditRevisionCell,
    },
    {
      accessorKey: 'detail',
      header: t('security.details'),
      size: 280,
      Cell: AuditDetailCell,
    },
  ], [t])

  return <DataTable data={events} columns={columns} emptyTitle={t('security.noEvents')} tableOptions={{ mantineTableContainerProps: { mah: 380 } }} />
}
