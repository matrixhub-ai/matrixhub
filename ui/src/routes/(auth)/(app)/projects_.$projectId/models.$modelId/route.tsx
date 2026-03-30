import {
  Box,
  Button,
  Tabs,
} from '@mantine/core'
import { IconDownload, IconCloudUpload } from '@tabler/icons-react'
import {
  Outlet, useMatchRoute, createFileRoute, linkOptions, Link, getRouteApi,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { projectRolesQueryOptions } from '@/features/auth/auth.query.ts'
import { modelQueryOptions } from '@/features/models/models.query'
import { buildModelBadges, buildModelMetaItems } from '@/features/models/models.utils'
import { ResourceDetailHeader } from '@/shared/components/ResourceDetailHeader'

import { Route as ModelBlobRoute } from './blob/$ref/$'
import { Route as ModelCommitRoute } from './commit/$commitId'
import { Route as ModelCommitsRoute } from './commits/$ref'
import { Route as ModelSettingsRoute } from './settings'
import { Route as ModelTreeRoute } from './tree/$ref/$'

import { Route as ModelIndexRoute } from './index'

const { useLoaderData: useUserData } = getRouteApi('/(auth)')
const { useLoaderData } = getRouteApi('/(auth)/(app)/projects_/$projectId/models/$modelId')

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId',
)({
  component: ModelDetailLayout,
  loader: async ({
    context, params,
  }) => {
    const [model, projectRoles] = await Promise.all([
      context.queryClient.ensureQueryData(modelQueryOptions(params.projectId, params.modelId)),
      context.queryClient.ensureQueryData(projectRolesQueryOptions()),
    ])

    return {
      model,
      projectRoles,
    }
  },
})

function ModelDetailLayout() {
  const { t } = useTranslation()
  const {
    projectId, modelId,
  } = Route.useParams()

  const userData = useUserData()
  const {
    model, projectRoles,
  } = useLoaderData()

  const hasSettingsRight = userData.isAdmin || projectRoles.projectRoles?.[projectId] === 'ROLE_TYPE_PROJECT_ADMIN'

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
        ref: model.defaultBranch ?? 'main',
        _splat: '',
      },
    },
    ...(hasSettingsRight
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
  const isTreeTabRoute = (
    !!matchRoute({ to: ModelTreeRoute.to })
    || !!matchRoute({ to: ModelBlobRoute.to })
    || !!matchRoute({ to: ModelCommitRoute.to })
    || !!matchRoute({ to: ModelCommitsRoute.to })
  )

  const activeTab = isTreeTabRoute
    ? 'tree'
    : (tabRoutes.find(tab => matchRoute({ to: tab.to }))?.id || tabRoutes[0].id)

  return (
    <Box pt={20} pb={32}>
      <Box mb={24}>
        <ResourceDetailHeader
          projectId={projectId}
          name={modelId}
          badges={buildModelBadges(model)}
          metaItems={buildModelMetaItems(model, projectId)}
          actions={(
            <>
              <Button size="xs" color="cyan" variant="light" leftSection={<IconCloudUpload size={16} />}>{t('model.upload')}</Button>
              <Button size="xs" color="cyan" variant="light" leftSection={<IconDownload size={16} />}>{t('model.download')}</Button>
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
