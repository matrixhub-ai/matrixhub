import {
  Anchor,
  Box,
  Button,
  Group,
  Skeleton,
  Stack,
  Text,
  TextInput,
} from '@mantine/core'
import { IconExternalLink } from '@tabler/icons-react'
import { useStore } from '@tanstack/react-form'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { createUpstreamModelNameSchema } from '@/features/models/models.schema'
import {
  DownloadDrawer,
  DownloadDrawerBody,
  DownloadDrawerFooter,
} from '@/features/models/server-pull/components/DownloadDrawer'
import { DownloadMethodCards } from '@/features/models/server-pull/components/DownloadMethodCards'
import { RevisionInputField } from '@/features/models/server-pull/components/RevisionFields'
import { ServerPullActions } from '@/features/models/server-pull/components/ServerPullActions'
import { ServerPullContent } from '@/features/models/server-pull/components/ServerPullContent'
import { useServerPull } from '@/features/models/server-pull/hooks/useServerPull'
import {
  type DownloadMethod,
  resolveDownloadHintKey,
  resolveDownloadMethod,
} from '@/features/models/server-pull/server-pull.utils'
import { useSystemConfig } from '@/features/system/system.query'
import { ShikiCodeBlock } from '@/shared/components/ShikiCodeBlock'
import { useForm } from '@/shared/hooks/useForm'
import { fieldError } from '@/shared/utils/form'

import { buildProxyDownloadCommand } from '../projects.utils'
import classes from './ProxyProjectDownload.module.css'

const MODEL_DOWNLOAD_DOC_URL = '/docs/operations/model-repo/upload-download/'
const HF_ENDPOINT_PLACEHOLDER = window.location.origin

interface ProxyProjectDownloadDrawerProps {
  opened: boolean
  projectId: string
  /** The MatrixHub project, used in the download command. */
  organization?: string
  /** The upstream organization, shown in front of the model name. */
  remoteOrganization?: string
  requiresToken?: boolean
  /** Lets the user have MatrixHub pull the model on the server (project admins and editors). */
  canPullOnServer?: boolean
  onClose: () => void
}

/**
 * "Download Model" of a proxy project: models are not created by hand, they are pulled from
 * the upstream registry, on the server or with the hf CLI.
 */
export function ProxyProjectDownloadDrawer({
  opened,
  onClose,
  ...props
}: ProxyProjectDownloadDrawerProps) {
  const { t } = useTranslation()

  return (
    <DownloadDrawer
      opened={opened}
      title={t('projects.detail.proxyDownload.title')}
      subtitle={t('projects.detail.proxyDownload.targetProject', { project: props.projectId })}
      onClose={onClose}
    >
      <ProxyProjectDownloadForm {...props} onClose={onClose} />
    </DownloadDrawer>
  )
}

function ProxyProjectDownloadForm({
  projectId,
  organization,
  remoteOrganization,
  requiresToken,
  canPullOnServer = false,
  onClose,
}: Omit<ProxyProjectDownloadDrawerProps, 'opened'>) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const defaultMethod: DownloadMethod = canPullOnServer ? 'server' : 'cli'
  const form = useForm({
    defaultValues: {
      modelName: '',
      method: defaultMethod,
      revision: '',
    },
  })
  const modelNameSchema = createUpstreamModelNameSchema(t)
  const modelName = useStore(form.store, state => state.values.modelName.trim())
  // The field validates on change, so a name without errors is one the server accepts.
  const modelNameValid = useStore(form.store, state => (state.fieldMeta.modelName?.errors.length ?? 0) === 0)
  const selectedMethod = useStore(form.store, state => state.values.method)
  const method = resolveDownloadMethod(canPullOnServer, selectedMethod)
  const revision = useStore(form.store, state => state.values.revision)
  const pull = useServerPull({
    projectId,
    modelName: modelNameValid ? modelName : '',
    revision,
    enabled: method === 'server',
  })
  const hintKey = resolveDownloadHintKey(method, pull.view)

  return (
    <>
      <DownloadDrawerBody>
        <form.Field
          name="modelName"
          validators={{ onChange: modelNameSchema }}
        >
          {field => (
            <TextInput
              label={t('projects.detail.proxyDownload.modelNameLabel')}
              withAsterisk
              placeholder={t('projects.detail.proxyDownload.modelNamePlaceholder')}
              value={field.state.value}
              onChange={event => field.handleChange(event.currentTarget.value)}
              onBlur={field.handleBlur}
              error={fieldError(field)}
              inputContainer={input => (remoteOrganization
                ? (
                    <Group gap="xs" wrap="nowrap" align="stretch">
                      <Box className={classes.organization}>{remoteOrganization}</Box>
                      <Text c="dimmed" className={classes.separator} aria-hidden>/</Text>
                      <Box flex={1} miw={0}>{input}</Box>
                    </Group>
                  )
                : input)}
            />
          )}
        </form.Field>

        {canPullOnServer && (
          <form.Field name="method">
            {field => (
              <DownloadMethodCards
                value={field.state.value}
                onChange={field.handleChange}
                onBlur={field.handleBlur}
              />
            )}
          </form.Field>
        )}

        {method === 'server'
          ? (
              <>
                <form.Field name="revision">
                  {field => (
                    <RevisionInputField
                      value={field.state.value}
                      onChange={field.handleChange}
                      onBlur={field.handleBlur}
                      defaultBranch={pull.defaultBranch}
                    />
                  )}
                </form.Field>
                <ServerPullContent
                  pull={pull}
                  revision={revision.trim()}
                  idleHint={t('model.serverPull.summary.enterName')}
                />
              </>
            )
          : (
              <ProxyDownloadCommand
                organization={organization}
                modelName={modelName}
                requiresToken={requiresToken}
              />
            )}
      </DownloadDrawerBody>

      <DownloadDrawerFooter
        hint={hintKey && t(hintKey)}
      >
        {method === 'server'
          ? (
              <ServerPullActions
                pull={pull}
                onClose={onClose}
                onViewModel={() => {
                  onClose()
                  void navigate({
                    to: '/projects/$projectId/models/$modelId',
                    params: {
                      projectId,
                      modelId: pull.modelName,
                    },
                  })
                }}
              />
            )
          : <Button onClick={onClose}>{t('common.gotIt')}</Button>}
      </DownloadDrawerFooter>
    </>
  )
}

function ProxyDownloadCommand({
  organization,
  modelName,
  requiresToken,
}: {
  organization?: string
  modelName: string
  requiresToken?: boolean
}) {
  const { t } = useTranslation()
  const systemConfigQuery = useSystemConfig()
  const hfEndpoint = systemConfigQuery.data?.endpoints?.hfBase || HF_ENDPOINT_PLACEHOLDER
  const command = buildProxyDownloadCommand(
    hfEndpoint,
    organization,
    modelName || t('projects.detail.proxyDownload.commandPlaceholder'),
    requiresToken,
  )

  return (
    <Stack gap="xs">
      <Text size="sm" fw={500}>{t('projects.detail.proxyDownload.command')}</Text>
      {systemConfigQuery.isPending
        ? <Skeleton h={76} />
        : <ShikiCodeBlock code={command} lang="bash" />}
      <Anchor
        size="sm"
        c="cyan.6"
        href={t('common.docs', { doc: MODEL_DOWNLOAD_DOC_URL })}
        target="_blank"
        rel="noopener noreferrer"
        ml="auto"
      >
        <Group gap={4} wrap="nowrap">
          <Text inherit>{t('projects.detail.proxyDownload.viewDocs')}</Text>
          <IconExternalLink size={14} />
        </Group>
      </Anchor>
    </Stack>
  )
}
