import {
  Accordion,
  Anchor,
  Badge,
  Group,
  Select,
  Skeleton,
  Stack,
  Tabs,
  Text,
  Tooltip,
} from '@mantine/core'
import {
  IconExternalLink,
  IconInfoCircle,
} from '@tabler/icons-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  GuideDrawer,
  GuideSnippetBlock as SnippetBlock,
  GuideStep as Step,
} from '@/features/models/guides/GuideDrawer'
import guideClasses from '@/features/models/guides/GuideDrawer.module.css'
import { useSystemConfig } from '@/features/system/system.query'

import classes from './UseModelDrawer.module.css'
import {
  buildDockerSnippet,
  buildEnvSnippet,
  buildInstallSnippet,
  buildLowLevelSnippet,
  buildPipelineSnippet,
  buildServeSnippet,
  buildTestRequestSnippet,
  ENGINE_DOC_URLS,
  ENGINE_LABELS,
  type SnippetPrompts,
  type UseModelEngine,
  type UseModelTask,
} from './useModelGuides'

const TASK_OPTIONS: UseModelTask[] = ['text-generation', 'image-text-to-text']

interface UseModelDrawerProps {
  opened: boolean
  engine: UseModelEngine
  modelPath: string
  /** Task detected from the model's labels; the user may override it inside the drawer. */
  defaultTask: UseModelTask
  onClose: () => void
}

export function UseModelDrawer({
  opened,
  engine,
  modelPath,
  defaultTask,
  onClose,
}: UseModelDrawerProps) {
  const { t } = useTranslation()
  const systemConfigQuery = useSystemConfig()
  const [task, setTask] = useState<UseModelTask>(defaultTask)

  const hfEndpoint = systemConfigQuery.data?.endpoints?.hfBase || window.location.origin
  const engineLabel = ENGINE_LABELS[engine]
  const prompts: SnippetPrompts = {
    textPrompt: t('model.detail.useModel.prompts.text'),
    imageUrl: t('model.detail.useModel.prompts.imageUrl'),
    imagePrompt: t('model.detail.useModel.prompts.image'),
  }

  const taskSelect = (
    <Group gap="xs" wrap="nowrap">
      <Text size="xs" c="dimmed">{t('model.detail.useModel.currentTask')}</Text>
      <Select
        size="xs"
        w={150}
        allowDeselect={false}
        value={task}
        onChange={value => value && setTask(value as UseModelTask)}
        data={TASK_OPTIONS.map(option => ({
          value: option,
          label: t(`model.detail.useModel.tasks.${option}`),
        }))}
      />
    </Group>
  )

  const envStep = (
    <Step
      index={3}
      title={t('model.detail.useModel.steps.configure')}
      hint={t('model.detail.useModel.steps.configureHint')}
    >
      {systemConfigQuery.isPending
        ? <Skeleton height={72} radius="sm" />
        : <SnippetBlock snippet={buildEnvSnippet(hfEndpoint)} />}
    </Step>
  )

  return (
    <GuideDrawer
      opened={opened}
      onClose={onClose}
      title={t('model.detail.useModel.title', { engine: engineLabel })}
      intro={t(`model.detail.useModel.intro.${engine}`)}
      footerExtra={(
        <Anchor
          href={ENGINE_DOC_URLS[engine]}
          target="_blank"
          rel="noopener noreferrer"
          size="sm"
        >
          <Group gap={4} wrap="nowrap">
            {t('model.detail.useModel.viewDocs', { engine: engineLabel })}
            <IconExternalLink size={14} />
          </Group>
        </Anchor>
      )}
    >
      <Step
        index={1}
        title={t('model.detail.useModel.steps.prepare')}
        hint={t('model.detail.useModel.steps.prepareHint')}
      >
        <Group gap="xs">
          <Badge variant="light" color="cyan" radius="sm" tt="none">Linux</Badge>
          <Badge variant="light" color="cyan" radius="sm" tt="none">Python 3.10+</Badge>
          <Badge variant="light" color="cyan" radius="sm" tt="none">{t('model.detail.useModel.gpu')}</Badge>
        </Group>
      </Step>

      <Step index={2} title={t('model.detail.useModel.steps.install', { engine: engineLabel })}>
        <SnippetBlock snippet={buildInstallSnippet(engine)} />
      </Step>

      {envStep}

      {engine === 'transformers'
        ? (
            <Step
              index={4}
              title={(
                <Group gap={4} wrap="nowrap">
                  {t('model.detail.useModel.steps.generate')}
                  <Tooltip label={t('model.detail.useModel.steps.generateTooltip')} multiline maw={280} withArrow>
                    <IconInfoCircle size={14} className={classes.infoIcon} />
                  </Tooltip>
                </Group>
              )}
              extra={taskSelect}
              hint={t('model.detail.useModel.steps.generateHint')}
            >
              <Tabs defaultValue="pipeline" variant="default">
                <Tabs.List mb="xs">
                  <Tabs.Tab value="pipeline">{t('model.detail.useModel.pipelineTab')}</Tabs.Tab>
                  <Tabs.Tab value="lowLevel">{t('model.detail.useModel.lowLevelTab')}</Tabs.Tab>
                </Tabs.List>
                <Tabs.Panel value="pipeline">
                  <SnippetBlock snippet={buildPipelineSnippet(task, modelPath, prompts)} />
                </Tabs.Panel>
                <Tabs.Panel value="lowLevel">
                  <Stack gap="xs">
                    <SnippetBlock snippet={buildLowLevelSnippet(task, modelPath, prompts)} />
                    <Text size="xs" c="dimmed">{t('model.detail.useModel.lowLevelNote')}</Text>
                  </Stack>
                </Tabs.Panel>
              </Tabs>
            </Step>
          )
        : (
            <>
              <Step index={4} title={t('model.detail.useModel.steps.serve')}>
                <SnippetBlock snippet={buildServeSnippet(engine, modelPath)} />
              </Step>
              <Step
                index={5}
                title={t('model.detail.useModel.steps.testRequest')}
                extra={taskSelect}
                hint={t('model.detail.useModel.steps.testRequestHint')}
              >
                <SnippetBlock snippet={buildTestRequestSnippet(engine, modelPath, task, prompts)} />
              </Step>
            </>
          )}

      <Accordion
        variant="contained"
        radius="sm"
        chevronPosition="left"
        classNames={{
          item: guideClasses.accordionItem,
          control: guideClasses.accordionControl,
          content: guideClasses.accordionContent,
        }}
      >
        <Accordion.Item value="docker">
          <Accordion.Control>
            <Group gap="xs" wrap="nowrap">
              <Text size="sm" fw={600}>{t('model.detail.useModel.otherMethods')}</Text>
              <Text size="sm" fw={600}>Docker</Text>
            </Group>
          </Accordion.Control>
          <Accordion.Panel>
            {systemConfigQuery.isPending
              ? <Skeleton height={44} radius="sm" />
              : <SnippetBlock snippet={buildDockerSnippet(hfEndpoint, modelPath)} />}
          </Accordion.Panel>
        </Accordion.Item>
      </Accordion>
    </GuideDrawer>
  )
}
