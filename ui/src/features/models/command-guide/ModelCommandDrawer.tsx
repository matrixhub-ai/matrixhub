import {
  Code, SegmentedControl, Skeleton, Stack, Text,
} from '@mantine/core'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  GuideDrawer,
  GuideNote,
  GuideSnippetBlock,
  GuideStep,
} from '@/features/models/guides/GuideDrawer'
import {
  resolveModelDownloadProvider, resolveModelGuideRevision, type ModelDownloadProvider,
} from '@/features/models/use-model/useModelGuides'
import { projectDetailQueryOptions } from '@/features/projects/projects.query'
import { useSystemConfig } from '@/features/system/system.query'

import {
  buildDownloadSnippet,
  buildModelScopeInstallSnippet,
  buildModelScopeEnvSnippet,
  buildModelScopeDownloadSnippet,
  buildModelScopePythonSnippet,
  buildHfEnvSnippet,
  buildInstallHfSnippet,
  buildLocalPathPlaceholder,
  buildUploadSnippet,
  HF_DEFAULT_CACHE_DIR,
  type ModelCommandType,
} from './commandGuides'

interface ModelCommandDrawerProps {
  opened: boolean
  type: ModelCommandType
  modelPath: string
  revision?: string
  onClose: () => void
}

/** Upload and download instructions for the supported model clients. */
export function ModelCommandDrawer({
  opened,
  type,
  modelPath,
  revision,
  onClose,
}: ModelCommandDrawerProps) {
  const { t } = useTranslation()
  const [providerChoice, setProviderChoice] = useState<ModelDownloadProvider | null>(null)
  const projectQuery = useQuery({
    ...projectDetailQueryOptions(modelPath.split('/')[0]),
    enabled: opened && type === 'download',
  })
  const provider = providerChoice ?? resolveModelDownloadProvider(projectQuery.data?.registryType)
  const isModelScope = type === 'download' && provider === 'modelscope'
  const systemConfigQuery = useSystemConfig()
  const hfEndpoint = systemConfigQuery.data?.endpoints?.hfBase || window.location.origin
  const localPath = buildLocalPathPlaceholder(modelPath)
  const guideRevision = resolveModelGuideRevision(provider, revision)
  const downloadSnippet = isModelScope
    ? buildModelScopeDownloadSnippet(modelPath, guideRevision)
    : buildDownloadSnippet(modelPath, revision)

  return (
    <GuideDrawer
      opened={opened}
      onClose={onClose}
      title={t(`model.detail.commandGuide.${type}.title`)}
      intro={t(`model.detail.commandGuide.${type}.intro`)}
    >
      {type === 'download' && (
        <SegmentedControl
          value={provider}
          onChange={value => setProviderChoice(value === 'modelscope' ? 'modelscope' : 'hf')}
          aria-label={t('model.detail.commandGuide.download.provider')}
          data={[
            {
              value: 'hf',
              label: t('routes.admin.registries.provider.huggingFace'),
            },
            {
              value: 'modelscope',
              label: t('routes.admin.registries.provider.modelScope'),
            },
          ]}
        />
      )}
      <GuideStep
        index={1}
        title={t(type === 'download'
          ? 'model.detail.commandGuide.download.install'
          : 'model.detail.commandGuide.steps.install')}
      >
        <Text size="sm">{t('model.detail.commandGuide.steps.installHint')}</Text>
        <GuideSnippetBlock snippet={isModelScope ? buildModelScopeInstallSnippet() : buildInstallHfSnippet()} />
      </GuideStep>

      <GuideStep
        index={2}
        title={t(`model.detail.commandGuide.${type}.configure`)}
        hint={type === 'upload' ? t('model.detail.commandGuide.steps.configureHint') : undefined}
      >
        {systemConfigQuery.isPending
          ? <Skeleton height={type === 'upload' ? 72 : 46} radius="sm" />
          : <GuideSnippetBlock snippet={isModelScope ? buildModelScopeEnvSnippet(hfEndpoint) : buildHfEnvSnippet(type, hfEndpoint)} />}
      </GuideStep>

      {type === 'download'
        ? (
            <Stack gap="md">
              <GuideStep
                index={3}
                title={t('model.detail.commandGuide.download.run')}
              >
                <GuideSnippetBlock snippet={downloadSnippet} />
              </GuideStep>
              {isModelScope && (
                <Stack gap="xs">
                  <Text size="sm">{t('model.detail.commandGuide.download.python')}</Text>
                  <GuideSnippetBlock snippet={buildModelScopePythonSnippet(modelPath, hfEndpoint, guideRevision)} />
                  <Text size="sm" c="dimmed">{t('model.detail.commandGuide.download.modelScopeToken')}</Text>
                </Stack>
              )}
              <GuideNote>
                {t('model.detail.commandGuide.download.note')}
                {' '}
                <Code>{isModelScope ? `./${modelPath.split('/').pop()}` : HF_DEFAULT_CACHE_DIR}</Code>
              </GuideNote>
            </Stack>
          )
        : (
            <Stack gap="md">
              <GuideStep
                index={3}
                title={t('model.detail.commandGuide.upload.run')}
              >
                <GuideSnippetBlock snippet={buildUploadSnippet(modelPath)} />
              </GuideStep>
              <GuideNote>
                <Stack gap={4}>
                  <Text size="sm">
                    {t('model.detail.commandGuide.upload.localPath')}
                    {' '}
                    <Code>{localPath}</Code>
                  </Text>
                  <Text size="sm">
                    {t('model.detail.commandGuide.upload.localPathHint')}
                    <br />
                    <Code>{t('model.detail.commandGuide.upload.localPathExample', { name: modelPath.split('/').pop() })}</Code>
                  </Text>
                </Stack>
              </GuideNote>
            </Stack>
          )}
    </GuideDrawer>
  )
}
