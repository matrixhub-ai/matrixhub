import {
  Tabs, Space, Stack,
  Group,
  Text,
  Badge,
  Button,
  CopyButton,
  ActionIcon,
  Tooltip,
  Box,
} from '@mantine/core'
import { Category, type Label } from '@matrixhub/api-ts/v1alpha1/model.pb.ts'
import {
  Outlet, Link, useMatchRoute, createFileRoute, linkOptions,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import CopyIcon from '@/assets/svgs/copy.svg?react'
import DownloadIcon from '@/assets/svgs/download.svg?react'
import FileIcon from '@/assets/svgs/file.svg?react'
import UploadIcon from '@/assets/svgs/upload-cloud.svg?react'

// TODO: Replace with real API data
const MOCK_META: Meta = {
  labels: [
    {
      id: 1,
      name: '文本分类',
      category: Category.TASK,
      createdAt: '2024-01-01T12:00:00Z',
      updatedAt: '2024-01-01T12:00:00Z',
    },
  ],
  size: '595 GB',
  updatedAt: '2021-12-17 12:12',
}

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId',
)({
  component: ModelLayout,
  // loader: async ({ params }) => {
  // const model = await Models.GetModel({
  //   project: params.projectId,
  //   name: params.modelId,
  // })
  loader: async () => {
    return {
      model: MOCK_META,
    }
  },
})

interface Meta {
  labels?: Label[]
  size?: string
  updatedAt?: string
}

function DetailHeader({
  projectId,
  name,
  meta,
}: {
  projectId: string
  name: string
  meta: Meta
}) {
  const { t } = useTranslation()
  const fullName = `${projectId}/${name}`

  return (
    <Stack gap={12}>
      {/* Row 1: Breadcrumb + Action buttons */}
      <Group justify="space-between" align="center">
        <Group gap="4" align="center">
          <Text
            component={Link}
            to={`/projects/${projectId}`}
            c="cyan.6"
            fw={500}
            size="lg"
            td="none"
            style={{ cursor: 'pointer' }}
          >
            {projectId}
          </Text>

          <Text c="dimmed" size="lg" w="1.5rem" ta="center" inline>/</Text>
          <Text size="lg">{name}</Text>
          <CopyButton value={fullName} timeout={2000}>
            {({
              copied,
              copy,
            }) => (
              <Tooltip label={copied ? t('model.copied') : t('model.copyName')} withArrow>
                <ActionIcon
                  variant="subtle"
                  color="gray"
                  onClick={copy}
                  size={24}
                >
                  <CopyIcon />
                </ActionIcon>
              </Tooltip>
            )}
          </CopyButton>
        </Group>

        <Group gap="sm">
          <Button
            size="xs"
            leftSection={<UploadIcon />}
          >
            {t('model.upload')}
          </Button>
          <Button
            size="xs"
            leftSection={<DownloadIcon />}
          >
            {t('model.download')}
          </Button>
        </Group>
      </Group>

      {/* Row 2: Badges */}
      <Group gap="sm">
        {meta.labels?.map(label => (
          <Badge
            key={label.id}
            variant="light"
            color="gray"
            leftSection={<FileIcon />}
            size="lg"
            radius="xl"
            fw={600}
          >
            {label.name}
          </Badge>
        ))}
      </Group>

      {/* Row 3: Metadata */}
      <Group gap="xl">
        <Text size="sm" c="dimmed">
          {t('model.project')}
          ：
          {projectId}
        </Text>
        <Text size="sm" c="dimmed">
          {t('model.size')}
          ：
          {meta.size ?? '-'}
        </Text>
        <Text size="sm" c="dimmed">
          {t('model.updatedAt')}
          ：
          {meta.updatedAt ?? '-'}
        </Text>
      </Group>
    </Stack>
  )
}

function ModelLayout() {
  const { t } = useTranslation()

  const {
    projectId, modelId,
  } = Route.useParams()

  const { model } = Route.useLoaderData()

  const tabRoutes = linkOptions([
    {
      id: 'desc',
      label: t('model.detail.desc'),
      to: Route.to,
      params: {
        projectId,
        modelId,
      },
    },
    {
      id: 'tree',
      label: t('model.detail.tree'),
      to: '/projects/$projectId/models/$modelId/tree/$ref/$',
      params: {
        projectId,
        modelId,
        ref: 'testDsd',
        _splat: 'test/data',
      },
    },
    {
      id: 'settings',
      label: t('model.detail.setting'),
      to: '/projects/$projectId/models/$modelId/settings',
      params: {
        projectId,
        modelId,
      },
    },
  ])

  const matchRoute = useMatchRoute()

  const activeTab = tabRoutes.find(tab => matchRoute({
    to: tab.to,
  }))?.id || tabRoutes[0].id

  return (
    <>
      <Box>
        <DetailHeader projectId={projectId} name={modelId} meta={model} />
      </Box>
      <Space h="1.5rem" />
      <Tabs value={activeTab}>
        <Tabs.List style={{ gap: 'var(--mantine-spacing-md)' }}>
          {
            tabRoutes.map(({
              id,
              label, ...linkProps
            }) => (
              <Tabs.Tab
                key={label}
                value={id}
                component={Link}
                fw={600}
                fz="sm"
                lh="xs"
                px="12px"
                py="8px"
                c={id === activeTab ? 'var(--mantine-color-gray-7)' : 'var(--mantine-color-gray-6)'}
                {...linkProps}
              >
                {label}
              </Tabs.Tab>
            ))
          }
        </Tabs.List>
      </Tabs>

      <Box>
        {
          activeTab === 'desc'
            ? (
                <div>
                  Description Page
                </div>
              )
            : <Outlet />
        }
      </Box>
    </>
  )
}
