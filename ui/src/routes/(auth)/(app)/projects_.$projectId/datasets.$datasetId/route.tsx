import {
  Box,
  Button,
  Space,
  Tabs,
} from '@mantine/core'
import { type Dataset } from '@matrixhub/api-ts/v1alpha1/dataset.pb.ts'
import { Category } from '@matrixhub/api-ts/v1alpha1/model.pb'
import {
  Outlet,
  Link,
  createFileRoute,
  linkOptions,
  useMatchRoute,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import DownloadIcon from '@/assets/svgs/download.svg?react'
import UploadIcon from '@/assets/svgs/upload-cloud.svg?react'

import { DetailHeader } from '../-components/DetailHeader'

// TODO: Replace with real API data
const MOCK_DATA: Dataset = {
  size: '595 GB',
  updatedAt: '2021-12-17 12:12',
  labels: [
    {
      id: 1,
      name: 'Text',
      category: Category.TASK,
      createdAt: '2024-01-01T12:00:00Z',
      updatedAt: '2024-01-01T12:00:00Z',
    },
  ],
}

const ProjectsRolesMock = {
  projectRoles: {
    project1: 'admin',
  },
}

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/datasets/$datasetId',
)({
  component: DatasetLayout,
  // loader: async ({ params }) => {
  //   const [datasetRes, prosRoleRes] = await Promise.allSettled([
  //     Datasets.GetDataset({
  //       project: params.projectId,
  //       name: params.datasetId,
  //     }),
  //     CurrentUser.GetProjectRoles({}),
  //   ])

  //   if (datasetRes.status === 'rejected') {
  //     throw new Error(`Failed to load dataset: ${datasetRes.reason}`)
  //   }

  //   if (prosRoleRes.status === 'rejected') {
  //     throw new Error(`Failed to load project roles: ${prosRoleRes.reason}`)
  //   }

  //   return {
  //     dataset: datasetRes.value,
  //     projectRoles: prosRoleRes.value,
  //   }
  // },
  loader: async () => {
    return {
      dataset: MOCK_DATA,
      projectRoles: ProjectsRolesMock,
    }
  },
})

function DatasetLayout() {
  const { t } = useTranslation()

  const {
    projectId, datasetId,
  } = Route.useParams()

  const {
    dataset, projectRoles,
  } = Route.useLoaderData()

  const hasProjectRole = Object.hasOwn(projectRoles.projectRoles ?? {}, projectId)

  const tabRoutes = linkOptions([
    {
      id: 'desc',
      label: t('dataset.detail.desc'),
      to: Route.to,
      params: {
        projectId,
        datasetId,
      },
    },
    {
      id: 'tree',
      label: t('dataset.detail.tree'),
      to: '/projects/$projectId/datasets/$datasetId/tree/$ref/$',
      params: {
        projectId,
        datasetId,
        ref: 'testDsd',
        _splat: 'test/data',
      },
    },
    ...(hasProjectRole
      ? [{
          id: 'settings',
          label: t('dataset.detail.setting'),
          to: '/projects/$projectId/datasets/$datasetId/settings',
          params: {
            projectId,
            datasetId,
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
          name={datasetId}
          size={dataset.size}
          updatedAt={dataset.updatedAt}
          labels={dataset.labels}
          actions={(
            <>
              <Button size="xs" leftSection={<UploadIcon />}>{t('dataset.uploadFiles')}</Button>
              <Button size="xs" leftSection={<DownloadIcon />}>{t('dataset.downloadDataset')}</Button>
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
      <Space h="md" />
      <div>
        {
          activeTab === 'desc'
            ? (
                <div>
                  Dataset Description Page
                </div>
              )
            : <Outlet />
        }
      </div>
    </>
  )
}
