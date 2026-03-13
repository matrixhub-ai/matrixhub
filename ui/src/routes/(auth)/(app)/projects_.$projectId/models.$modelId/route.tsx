import {
  Box,
  Button,
  Space,
  Tabs,
} from '@mantine/core'
import { Category, type Model } from '@matrixhub/api-ts/v1alpha1/model.pb.ts'
import {
  Outlet, Link, useMatchRoute, createFileRoute, linkOptions,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import DownloadIcon from '@/assets/svgs/download.svg?react'
import UploadIcon from '@/assets/svgs/upload-cloud.svg?react'

import { DetailHeader } from '../-components/DetailHeader'

// TODO: Replace with real API data
const MOCK_DATA: Model = {
  labels: [
    {
      id: 1,
      name: 'Image-to-Text',
      category: Category.TASK,
      createdAt: '2024-01-01T12:00:00Z',
      updatedAt: '2024-01-01T12:00:00Z',
    },
  ],
  size: '595 GB',
  updatedAt: '2021-12-17 12:12',
}

const ProjectsRolesMock = {
  projectRoles: {
    project1: 'admin',
  },
}

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId',
)({
  component: ModelLayout,
  // loader: async ({ params }) => {
  //   const [modelRes, prosRoleRes] = await Promise.allSettled([
  //     Models.GetModel({
  //       project: params.projectId,
  //       name: params.modelId,
  //     }),
  //     CurrentUser.GetProjectRoles({}),
  //   ])

  //   if (modelRes.status === 'rejected') {
  //     throw new Error(`Failed to load model: ${modelRes.reason}`)
  //   }

  //   if (prosRoleRes.status === 'rejected') {
  //     throw new Error(`Failed to load project roles: ${prosRoleRes.reason}`)
  //   }

  //   return {
  //     model: modelRes.value,
  //     projectRoles: prosRoleRes.value,
  //   }
  // },
  loader: async () => {
    return {
      model: MOCK_DATA,
      projectRoles: ProjectsRolesMock,
    }
  },
})

function ModelLayout() {
  const { t } = useTranslation()

  const {
    projectId, modelId,
  } = Route.useParams()

  const {
    model, projectRoles,
  } = Route.useLoaderData()

  const hasProjectRole = Object.hasOwn(projectRoles.projectRoles ?? {}, projectId)

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
    ...(hasProjectRole
      ? [{
          id: 'settings',
          label: t('model.detail.setting'),
          to: '/projects/$projectId/models/$modelId/settings',
          params: {
            projectId,
            modelId,
          },
        }]
      : []),
  ])

  const matchRoute = useMatchRoute()

  const activeTab = tabRoutes.find(tab => matchRoute({
    to: tab.to,
  }))?.id || 'desc'

  return (
    <>
      <Box>
        <DetailHeader
          projectId={projectId}
          name={modelId}
          size={model.size}
          updatedAt={model.updatedAt}
          labels={model.labels}
          actions={(
            <>
              <Button size="xs" leftSection={<UploadIcon />}>{t('model.upload')}</Button>
              <Button size="xs" leftSection={<DownloadIcon />}>{t('model.download')}</Button>
            </>
          )}
        />
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
                key={id}
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
