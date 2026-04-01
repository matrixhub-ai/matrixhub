import {
  Box,
  Button,
  Tabs,
} from '@mantine/core'
import { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb.ts'
import { IconDownload, IconCloudUpload } from '@tabler/icons-react'
import {
  Link,
  getRouteApi,
  linkOptions,
  useMatchRoute,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { useProjectRole } from '@/context/project-role-context.tsx'
import { buildModelBadges, buildModelMetaItems } from '@/features/models/models.utils'
import { ResourceDetailHeader } from '@/shared/components/ResourceDetailHeader'

import type { ReactNode } from 'react'

interface ModelDetailPageProps {
  children: ReactNode
}

const {
  useLoaderData,
  useParams,
} = getRouteApi('/(auth)/(app)/projects_/$projectId/models/$modelId')

export function ModelDetailPage({
  children,
}: ModelDetailPageProps) {
  const { t } = useTranslation()
  const {
    projectId, modelId,
  } = useParams()

  const { model } = useLoaderData()
  const projectRole = useProjectRole(projectId)
  const hasSettingsRight = projectRole === ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN

  const tabRoutes = linkOptions([
    {
      id: 'desc',
      label: t('model.detail.desc'),
      to: '/projects/$projectId/models/$modelId',
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
        ref: model.defaultBranch ?? 'main',
        _splat: '',
      },
    },
    ...(hasSettingsRight
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
  const isTreeTabRoute = (
    !!matchRoute({ to: '/projects/$projectId/models/$modelId/tree/$ref/$' })
    || !!matchRoute({ to: '/projects/$projectId/models/$modelId/blob/$ref/$' })
    || !!matchRoute({ to: '/projects/$projectId/models/$modelId/commit/$commitId' })
    || !!matchRoute({ to: '/projects/$projectId/models/$modelId/commits/$ref' })
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
              <Button size="xs" color="cyan" variant="light" leftSection={<IconCloudUpload size={16} />}>{t('model.detail.upload')}</Button>
              <Button size="xs" color="cyan" variant="light" leftSection={<IconDownload size={16} />}>{t('model.detail.download')}</Button>
            </>
          )}
        />
      </Box>
      <Tabs value={activeTab}>
        <Tabs.List>
          {tabRoutes.map(({
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
          ))}
        </Tabs.List>
      </Tabs>

      {children}
    </Box>
  )
}
