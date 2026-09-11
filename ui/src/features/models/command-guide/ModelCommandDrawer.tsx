import {
  Code, Skeleton, Stack, Text,
} from '@mantine/core'
import { useTranslation } from 'react-i18next'

import {
  GuideDrawer,
  GuideNote,
  GuideSnippetBlock,
  GuideStep,
} from '@/features/models/guides/GuideDrawer'
import { useSystemConfig } from '@/features/system/system.query'

import {
  buildDownloadSnippet,
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
  onClose: () => void
}

/** Step-by-step guide for uploading to / downloading from a model repo with the `hf` CLI. */
export function ModelCommandDrawer({
  opened,
  type,
  modelPath,
  onClose,
}: ModelCommandDrawerProps) {
  const { t } = useTranslation()
  const systemConfigQuery = useSystemConfig()
  const hfEndpoint = systemConfigQuery.data?.endpoints?.hfBase || window.location.origin
  const localPath = buildLocalPathPlaceholder(modelPath)

  return (
    <GuideDrawer
      opened={opened}
      onClose={onClose}
      title={t(`model.detail.commandGuide.${type}.title`)}
      intro={t(`model.detail.commandGuide.${type}.intro`)}
    >
      <GuideStep index={1} title={t('model.detail.commandGuide.steps.install')}>
        <Text size="sm">{t('model.detail.commandGuide.steps.installHint')}</Text>
        <GuideSnippetBlock snippet={buildInstallHfSnippet()} />
      </GuideStep>

      <GuideStep
        index={2}
        title={t(`model.detail.commandGuide.${type}.configure`)}
        hint={type === 'upload' ? t('model.detail.commandGuide.steps.configureHint') : undefined}
      >
        {systemConfigQuery.isPending
          ? <Skeleton height={type === 'upload' ? 72 : 46} radius="sm" />
          : <GuideSnippetBlock snippet={buildHfEnvSnippet(type, hfEndpoint)} />}
      </GuideStep>

      {type === 'download'
        ? (
            <>
              <GuideStep index={3} title={t('model.detail.commandGuide.download.run')}>
                <GuideSnippetBlock snippet={buildDownloadSnippet(modelPath)} />
              </GuideStep>
              <GuideNote>
                {t('model.detail.commandGuide.download.note')}
                {' '}
                <Code>{HF_DEFAULT_CACHE_DIR}</Code>
              </GuideNote>
            </>
          )
        : (
            <>
              <GuideStep index={3} title={t('model.detail.commandGuide.upload.run')}>
                <Text size="sm">{t('model.detail.commandGuide.upload.runHint')}</Text>
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
            </>
          )}
    </GuideDrawer>
  )
}
