import { Button } from '@mantine/core'
import { useStore } from '@tanstack/react-form'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { ModelCommandSteps } from '@/features/models/command-guide/ModelCommandDrawer'
import { modelRevisionsQueryOptions } from '@/features/models/models.query'
import { useForm } from '@/shared/hooks/useForm'

import {
  DownloadDrawer,
  DownloadDrawerBody,
  DownloadDrawerFooter,
} from './DownloadDrawer'
import { DownloadMethodCards } from './DownloadMethodCards'
import { RevisionSelectField } from './RevisionFields'
import { ServerPullActions } from './ServerPullActions'
import { ServerPullContent } from './ServerPullContent'
import { useServerPull } from '../hooks/useServerPull'
import {
  type DownloadMethod,
  resolveDownloadHintKey,
} from '../server-pull.utils'

const DEFAULT_METHOD: DownloadMethod = 'server'

interface ModelDownloadDrawerProps {
  opened: boolean
  projectId: string
  modelId: string
  /** `project/model`, for the hf CLI command. */
  modelPath: string
  defaultBranch: string
  onClose: () => void
}

/**
 * "Download Model" of a proxy-project model, for users who can pull: pull a revision into
 * MatrixHub on the server, or download it with the hf CLI.
 */
export function ModelDownloadDrawer({
  opened,
  projectId,
  modelId,
  modelPath,
  defaultBranch,
  onClose,
}: ModelDownloadDrawerProps) {
  const { t } = useTranslation()

  return (
    <DownloadDrawer
      opened={opened}
      title={t('model.detail.download')}
      subtitle={`${projectId} / ${modelId}`}
      onClose={onClose}
    >
      <ModelDownloadForm
        projectId={projectId}
        modelId={modelId}
        modelPath={modelPath}
        defaultBranch={defaultBranch}
        onClose={onClose}
      />
    </DownloadDrawer>
  )
}

function ModelDownloadForm({
  projectId,
  modelId,
  modelPath,
  defaultBranch,
  onClose,
}: Omit<ModelDownloadDrawerProps, 'opened'>) {
  const { t } = useTranslation()
  const form = useForm({
    defaultValues: {
      method: DEFAULT_METHOD,
      revision: defaultBranch,
    },
  })
  const method = useStore(form.store, state => state.values.method)
  const revision = useStore(form.store, state => state.values.revision)
  const revisionsQuery = useQuery(modelRevisionsQueryOptions(projectId, modelId))
  const pull = useServerPull({
    projectId,
    modelName: modelId,
    revision,
    enabled: method === 'server',
    // A selected revision needs no debounce.
    debounceMs: 0,
  })
  const hintKey = resolveDownloadHintKey(method, pull.view)

  return (
    <>
      <DownloadDrawerBody>
        <form.Field name="method">
          {field => (
            <DownloadMethodCards
              value={field.state.value}
              onChange={field.handleChange}
              onBlur={field.handleBlur}
            />
          )}
        </form.Field>

        {method === 'server'
          ? (
              <>
                <form.Field name="revision">
                  {field => (
                    <RevisionSelectField
                      value={field.state.value}
                      onChange={field.handleChange}
                      onBlur={field.handleBlur}
                      branches={revisionsQuery.data?.items?.branches}
                      tags={revisionsQuery.data?.items?.tags}
                      defaultBranch={defaultBranch}
                    />
                  )}
                </form.Field>
                <ServerPullContent pull={pull} revision={revision} />
              </>
            )
          : <ModelCommandSteps type="download" modelPath={modelPath} />}
      </DownloadDrawerBody>

      <DownloadDrawerFooter
        hint={hintKey && t(hintKey)}
      >
        {method === 'server'
          ? <ServerPullActions pull={pull} onClose={onClose} />
          : <Button onClick={onClose}>{t('common.gotIt')}</Button>}
      </DownloadDrawerFooter>
    </>
  )
}
