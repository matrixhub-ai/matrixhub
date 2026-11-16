import {
  Alert,
  Anchor,
  Box,
  Button,
  Drawer,
  Group,
  rem,
  SimpleGrid,
  Skeleton,
  Stack,
  Text,
  TextInput,
  ThemeIcon,
  Title,
} from '@mantine/core'
import {
  IconExternalLink,
  IconInfoCircle,
} from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import IconProxyProjectEmpty from '@/assets/svgs/proxy-project-empty.svg?react'
import { createModelSchema } from '@/features/models/models.schema'
import { useSystemConfig } from '@/features/system/system.query'
import { ShikiCodeBlock } from '@/shared/components/ShikiCodeBlock'
import { useForm } from '@/shared/hooks/useForm'
import { fieldError } from '@/shared/utils/form'

import { buildProxyDownloadCommand } from '../projects.utils'

import type { ReactNode } from 'react'

const MODEL_DOWNLOAD_DOC_URL = '/docs/operations/model-repo/upload-download/'
const HF_ENDPOINT_PLACEHOLDER = window.location.origin

interface ProxyProjectDownloadProps {
  organization?: string
  remoteOrganization?: string
  requiresToken?: boolean
  registryType?: string
}

interface ProxyProjectDownloadDrawerProps extends ProxyProjectDownloadProps {
  opened: boolean
  onClose: () => void
}

function StepTitle({
  number,
  children,
}: {
  number: number
  children: ReactNode
}) {
  return (
    <Group gap="xs">
      <ThemeIcon size="sm" radius="xl" variant="filled">
        <Text size="xs" lh="sm" fw={700}>{number}</Text>
      </ThemeIcon>
      <Text size="sm" fw={400}>{children}</Text>
    </Group>
  )
}

function ProxyProjectDownloadFlow({
  organization,
  remoteOrganization,
  requiresToken,
  registryType,
  layout,
}: ProxyProjectDownloadProps & {
  layout: 'drawer' | 'guide'
}) {
  const { t } = useTranslation()
  const systemConfigQuery = useSystemConfig()
  const form = useForm({
    defaultValues: {
      modelName: '',
    },
  })
  const { name: modelNameSchema } = createModelSchema(t).shape
  const hfEndpoint = systemConfigQuery.data?.endpoints?.hfBase || HF_ENDPOINT_PLACEHOLDER

  const steps = [
    (
      <Stack key="model-name" gap="sm">
        <StepTitle number={1}>{t('projects.detail.proxyDownload.steps.modelName')}</StepTitle>
        <form.Field
          name="modelName"
          validators={{ onChange: modelNameSchema }}
        >
          {field => (
            <TextInput
              required
              label={t('projects.detail.proxyDownload.modelNameLabel')}
              leftSection={remoteOrganization ? `${remoteOrganization} /` : undefined}
              leftSectionPointerEvents="none"
              leftSectionWidth={remoteOrganization ? `${remoteOrganization.length + 3}ch` : undefined}
              placeholder={t('projects.detail.proxyDownload.modelNamePlaceholder')}
              description={t('projects.detail.proxyDownload.modelNameHelper')}
              value={field.state.value}
              onChange={event => field.handleChange(event.currentTarget.value)}
              onBlur={field.handleBlur}
              error={fieldError(field)}
              styles={
                {
                  section: {
                    backgroundColor: 'var(--mantine-color-gray-1)',
                    borderRight: '1px solid var(--mantine-color-gray-3)',
                    color: 'var(--mantine-color-gray-6)',
                    fontSize: rem(14),
                  },
                  input: {
                    textIndent: rem(5),
                  },
                }
              }
              inputWrapperOrder={['label', 'input', 'description', 'error']}
            />
          )}
        </form.Field>
      </Stack>
    ),
    (
      <form.Subscribe
        key="command"
        selector={state => state.values.modelName}
      >
        {(modelName) => {
          const command = buildProxyDownloadCommand(
            hfEndpoint,
            organization,
            modelName.trim() || t('projects.detail.proxyDownload.commandPlaceholder'),
            requiresToken,
            registryType,
          )

          return (
            <Stack gap="xs">
              <StepTitle number={2}>{t('projects.detail.proxyDownload.steps.command')}</StepTitle>
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
        }}
      </form.Subscribe>
    ),
    (
      <Stack key="completion" gap="xs">
        <StepTitle number={3}>{t('projects.detail.proxyDownload.steps.completion')}</StepTitle>
        <Alert color="cyan" variant="light" icon={<IconInfoCircle size={18} />}>
          {t('projects.detail.proxyDownload.completionNotice')}
        </Alert>
      </Stack>
    ),
  ]

  return layout === 'guide'
    ? (
        <SimpleGrid
          cols={{
            base: 1,
            md: 3,
          }}
          spacing={rem('40px')}
        >
          {steps}
        </SimpleGrid>
      )
    : <Stack gap="sm">{steps}</Stack>
}

export function ProxyProjectDownloadGuide({
  organization,
  remoteOrganization,
  requiresToken,
  registryType,
}: ProxyProjectDownloadProps) {
  const { t } = useTranslation()

  return (
    <Stack mt="xl" gap="xl">
      <Group align="center" gap="40px" wrap="wrap">
        <Stack align="center" gap={0} miw={140}>
          <IconProxyProjectEmpty />
        </Stack>

        <Stack gap="sm" maw={620}>
          <Title order={3}>{t('projects.detail.proxyDownload.guideTitle')}</Title>
          <Text size="sm" c="dimmed">
            {t('projects.detail.proxyDownload.guideDescription')}
          </Text>
          <Stack gap="lg" mt="md">
            {[1, 2, 3].map(number => (
              <StepTitle key={number} number={number}>
                {t(`projects.detail.proxyDownload.steps.${['modelName', 'command', 'completion'][number - 1]}`)}
              </StepTitle>
            ))}
          </Stack>
        </Stack>
      </Group>

      <Box>
        <Title order={4} mb="md">{t('projects.detail.proxyDownload.quickDownload')}</Title>
        <ProxyProjectDownloadFlow
          organization={organization}
          remoteOrganization={remoteOrganization}
          requiresToken={requiresToken}
          registryType={registryType}
          layout="guide"
        />
      </Box>
    </Stack>
  )
}

export function ProxyProjectDownloadDrawer({
  opened,
  organization,
  remoteOrganization,
  requiresToken,
  registryType,
  onClose,
}: ProxyProjectDownloadDrawerProps) {
  const { t } = useTranslation()

  return (
    <Drawer
      opened={opened}
      onClose={onClose}
      position="right"
      size="xl"
      title={<Text fw={600} fz="md">{t('projects.detail.proxyDownload.title')}</Text>}
      styles={{
        body: {
          display: 'flex',
          flexDirection: 'column',
          height: 'calc(100% - var(--drawer-header-height, 60px))',
          padding: 0,
        },
      }}
    >
      <Box
        pt={rem('4px')}
        px="lg"
        pb="md"
        style={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
        }}
      >
        <Stack gap="sm">
          <Alert color="cyan" variant="light" icon={<IconInfoCircle size={18} />}>
            {t('projects.detail.proxyDownload.drawerNotice')}
          </Alert>

          <ProxyProjectDownloadFlow
            organization={organization}
            remoteOrganization={remoteOrganization}
            requiresToken={requiresToken}
            registryType={registryType}
            layout="drawer"
          />
        </Stack>
      </Box>

      <Group
        justify="flex-end"
        px="lg"
        py="sm"
        style={{ borderTop: '1px solid var(--app-color-gray-30)' }}
      >
        <Button color="cyan" variant="light" onClick={onClose}>
          {t('projects.detail.proxyDownload.acknowledge')}
        </Button>
      </Group>
    </Drawer>
  )
}
