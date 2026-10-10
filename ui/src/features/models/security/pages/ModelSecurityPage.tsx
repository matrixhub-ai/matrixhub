import {
  Alert, Button, Group, Modal, Paper, Stack, Tabs, Text, TextInput, Title,
} from '@mantine/core'
import { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb'
import { useMutation, useSuspenseQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { useProjectRole } from '@/features/auth/useProjectRole'
import { type ScanFile } from '@/features/models/security/security.api'
import { CopyValueButton } from '@/shared/components/CopyValueButton'
import { useForm } from '@/shared/hooks/useForm'
import { fieldError } from '@/shared/utils/form'

import { SecurityPolicyForm } from '../components/SecurityPolicyForm'
import {
  SecurityAuditTable, SecurityBadge, SecurityFileDetails, SecurityFilesTable,
} from '../components/SecurityTables'
import { securityTaskMutation } from '../security.mutation'
import {
  securityAuditQuery, securityPolicyQuery, securityReportQuery,
} from '../security.query'
import { securityRevisionSchema } from '../security.schema'

const route = getRouteApi('/(auth)/(app)/projects_/$projectId/models/$modelId/security/')

export function ModelSecurityPage() {
  const {
    projectId, modelId,
  } = route.useParams()
  const { revision } = route.useSearch()
  const navigate = route.useNavigate()
  const { t } = useTranslation()
  const { data: report } = useSuspenseQuery(securityReportQuery(projectId, modelId, revision))
  const { data: policy } = useSuspenseQuery(securityPolicyQuery(projectId, modelId))
  const { data: audit } = useSuspenseQuery(securityAuditQuery(projectId, modelId))
  const role = useProjectRole(projectId)
  const writable = role === ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN || role === ProjectRoleType.ROLE_TYPE_PROJECT_EDITOR
  const manager = role === ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN
  const rescan = useMutation(securityTaskMutation(projectId, modelId, 'rescan'))
  const cancel = useMutation(securityTaskMutation(projectId, modelId, 'cancel'))
  const [inspected, setInspected] = useState<ScanFile | null>(null)
  const [confirmCancel, setConfirmCancel] = useState(false)
  const form = useForm({
    defaultValues: { revision },
    onSubmit: async ({ value }) => {
      await navigate({ search: { revision: value.revision } })
    },
  })
  const active = report.status === 'pending' || report.status === 'scanning'
  const knownHighRisk = (report.files ?? []).some(file => (file.findings ?? []).some(finding => finding.severity === 'high'))

  return (
    <Stack py="lg" gap="md">
      <Group justify="space-between">
        <Stack gap="xs">
          <Title order={3}>{t('security.title')}</Title>
          <Text c="dimmed" size="sm">{t('security.intro')}</Text>
        </Stack>
        <Group>
          <form.Field name="revision" validators={{ onChange: securityRevisionSchema }}>{field => <TextInput aria-label={t('security.revision')} placeholder={t('security.revision')} value={field.state.value} onChange={event => field.handleChange(event.currentTarget.value)} onBlur={field.handleBlur} error={fieldError(field)} />}</form.Field>
          <Button variant="default" onClick={() => void form.handleSubmit()}>{t('security.view')}</Button>
          {writable && <Button loading={rescan.isPending} onClick={() => rescan.mutate(report.revision)}>{t('security.rescan')}</Button>}
          {writable && active && <Button color="red" variant="light" onClick={() => setConfirmCancel(true)}>{t('security.cancel')}</Button>}
        </Group>
      </Group>
      <Paper withBorder p="md" radius="md">
        <Group justify="space-between">
          <Group>
            <SecurityBadge status={report.status} />
            <Text size="sm">{t('security.attempt', { number: report.attempt })}</Text>
          </Group>
          <Text size="sm" c="dimmed">{report.updated_at ? new Date(report.updated_at).toLocaleString() : ''}</Text>
        </Group>
        <Group mt="sm">
          <Text size="sm" ff="monospace">{report.revision}</Text>
          <CopyValueButton value={report.revision} />
        </Group>
        {report.ruleset && (
          <Text size="xs" c="dimmed" mt="xs">
            {t('security.ruleset')}
            :
            {' '}
            {report.ruleset}
          </Text>
        )}
        {report.error && <Alert color="red" mt="sm">{t(knownHighRisk ? 'security.incompleteHighRisk' : 'security.reportIncomplete')}</Alert>}
        <Text size="sm" c="dimmed" mt="sm">{t('security.historyHint')}</Text>
      </Paper>
      <Tabs defaultValue="files">
        <Tabs.List>
          <Tabs.Tab value="files">{t('security.files')}</Tabs.Tab>
          <Tabs.Tab value="audit">{t('security.audit')}</Tabs.Tab>
          <Tabs.Tab value="policy">{t('security.policy')}</Tabs.Tab>
        </Tabs.List>
        <Tabs.Panel value="files" pt="md"><SecurityFilesTable files={report.files ?? []} onInspect={setInspected} /></Tabs.Panel>
        <Tabs.Panel value="audit" pt="md"><SecurityAuditTable events={audit.events} /></Tabs.Panel>
        <Tabs.Panel value="policy" pt="md"><SecurityPolicyForm key={JSON.stringify(policy)} project={projectId} model={modelId} policy={policy} writable={manager} /></Tabs.Panel>
      </Tabs>
      <Modal opened={!!inspected} onClose={() => setInspected(null)} title={t('security.details')} size="lg">{inspected && <SecurityFileDetails file={inspected} />}</Modal>
      <Modal opened={confirmCancel} onClose={() => setConfirmCancel(false)} title={t('security.cancel')}>
        <Stack>
          <Text>{t('security.cancelHint')}</Text>
          <Button
            color="red"
            loading={cancel.isPending}
            onClick={async () => {
              await cancel.mutateAsync(report.revision)
              setConfirmCancel(false)
            }}
          >
            {t('security.cancel')}
          </Button>
        </Stack>
      </Modal>
    </Stack>
  )
}
