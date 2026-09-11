import {
  Box,
  Button,
  Menu,
  Tabs,
} from '@mantine/core'
import { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb.ts'
import {
  IconChevronDown,
  IconCloudUpload,
  IconDownload,
  IconTerminal2,
} from '@tabler/icons-react'
import {
  Link,
  getRouteApi,
  linkOptions,
  useMatchRoute,
} from '@tanstack/react-router'
import { type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { useProjectRole } from '@/features/auth/useProjectRole'
import { type ModelCommandType } from '@/features/models/command-guide/commandGuides'
import { ModelCommandDrawer } from '@/features/models/command-guide/ModelCommandDrawer'
import { buildModelBadges, buildModelMetaItems } from '@/features/models/models.utils'
import { UseModelDrawer } from '@/features/models/use-model/UseModelDrawer'
import {
  ENGINE_LABELS,
  resolveUseModelTask,
  USE_MODEL_ENGINES,
  type UseModelEngine,
} from '@/features/models/use-model/useModelGuides'
import { ResourceDetailHeader } from '@/shared/components/ResourceDetailHeader'
import { usePayloadModal } from '@/shared/hooks/usePayloadModal'

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
  const {
    t, i18n,
  } = useTranslation()
  const {
    projectId, modelId,
  } = useParams()

  const { model } = useLoaderData()
  const commandDialog = usePayloadModal<ModelCommandType>()
  const useModelDrawer = usePayloadModal<UseModelEngine>()
  const projectRole = useProjectRole(projectId)
  const hasSettingsRight = projectRole === ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN
  const modelPath = `${model.project ?? projectId}/${model.name?.trim() || modelId}`

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
          metaItems={buildModelMetaItems(model, projectId, i18n.t.bind(i18n))}
          actions={(
            <>
              <Button
                color="cyan"
                fw="normal"
                variant="light"
                leftSection={<IconCloudUpload size={16} />}
                onClick={() => commandDialog.open('upload')}
              >
                {t('model.detail.upload')}
              </Button>
              <Button
                color="cyan"
                fw="normal"
                variant="light"
                leftSection={<IconDownload size={16} />}
                onClick={() => commandDialog.open('download')}
              >
                {t('model.detail.download')}
              </Button>
              <Menu shadow="md" position="bottom-end" width={160} closeOnItemClick>
                <Menu.Target>
                  <Button
                    color="cyan"
                    fw="normal"
                    leftSection={<IconTerminal2 size={16} />}
                    rightSection={<IconChevronDown size={14} />}
                  >
                    {t('model.detail.use')}
                  </Button>
                </Menu.Target>
                <Menu.Dropdown>
                  {USE_MODEL_ENGINES.map(engine => (
                    <Menu.Item key={engine} onClick={() => useModelDrawer.open(engine)}>
                      {ENGINE_LABELS[engine]}
                    </Menu.Item>
                  ))}
                </Menu.Dropdown>
              </Menu>
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
      {commandDialog.payload && (
        <ModelCommandDrawer
          opened={commandDialog.opened}
          type={commandDialog.payload}
          modelPath={modelPath}
          onClose={commandDialog.close}
        />
      )}
      {useModelDrawer.payload && (
        <UseModelDrawer
          key={useModelDrawer.payload}
          opened={useModelDrawer.opened}
          engine={useModelDrawer.payload}
          modelPath={modelPath}
          defaultTask={resolveUseModelTask(model)}
          onClose={useModelDrawer.close}
        />
      )}
    </Box>
  )
}
