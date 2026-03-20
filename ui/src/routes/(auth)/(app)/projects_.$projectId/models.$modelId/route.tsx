import {
  Box,
  Button,
  Tabs,
} from '@mantine/core'
import { Models } from '@matrixhub/api-ts/v1alpha1/model.pb.ts'
import { IconCloudUpload as UploadIcon, IconDownload as DownloadIcon } from '@tabler/icons-react'
import {
  Outlet, useMatchRoute, createFileRoute, linkOptions, Link,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { ResourceDetailHeader } from '@/shared/components/ResourceDetailHeader'

import { Route as ModelSettingsRoute } from './settings'
import { Route as ModelTreeRoute } from './tree/$ref/$'

import { Route as ModelIndexRoute } from './index'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId',
)({
  component: ModelDetailLayout,
  loader: async ({ params }) => {
    return await Models.GetModel({
      project: params.projectId,
      name: params.modelId,
    })
  },
})

function ModelDetailLayout() {
  const { t } = useTranslation()
  const {
    projectId, modelId,
  } = Route.useParams()

  const model = Route.useLoaderData()
  // TODO: use real project role
  const hasProjectRole = true

  const tabRoutes = linkOptions([
    {
      id: 'desc',
      label: t('model.detail.desc'),
      to: ModelIndexRoute.to,
      params: {
        projectId,
        modelId,
      },
    },
    {
      id: 'tree',
      label: t('model.detail.tree'),
      to: ModelTreeRoute.to,
      params: {
        projectId,
        modelId,
        // TODO: use real ref
        ref: 'main',
      },
    },
    ...(hasProjectRole
      ? [{
          id: 'settings',
          label: t('model.detail.setting'),
          to: ModelSettingsRoute.to,
          params: {
            projectId,
            modelId,
          },
        }]
      : []),
  ])

  const matchRoute = useMatchRoute()
  const activeTab = tabRoutes.find(tab => matchRoute({ to: tab.to }))?.id || tabRoutes[0].id

  return (
    <Box pt={20} pb={32}>
      <Box mb={24}>
        <ResourceDetailHeader
          projectId={projectId}
          name={modelId}
          size={model.size}
          updatedAt={model.updatedAt}
          labels={model.labels}
          actions={(
            <>
              <Button
                size="xs"
                color="cyan"
                variant="light"
                leftSection={<UploadIcon size={16} />}
              >
                {t('model.upload')}
              </Button>
              <Button
                size="xs"
                color="cyan"
                variant="light"
                leftSection={<DownloadIcon size={16} />}
              >
                {t('model.download')}
              </Button>
            </>
          )}
        />
      </Box>
      <Tabs value={activeTab}>
        <Tabs.List>
          {
            tabRoutes.map(({
              id, label, ...linkProps
            }) => (
              <Tabs.Tab
                key={id}
                value={id}
                component={Link}
                {...linkProps}
              >
                {label}
              </Tabs.Tab>
            ))
          }
        </Tabs.List>
      </Tabs>

      <Outlet />
    </Box>
  )
}
