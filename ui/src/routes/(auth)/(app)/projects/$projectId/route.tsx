import {
  Box, Group, Space, Tabs, Text,
} from '@mantine/core'
import { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb'
import { IconApiApp as ProjectIcon } from '@tabler/icons-react'
import {
  Link,
  createFileRoute,
  linkOptions,
  Outlet,
  useMatchRoute,
  useLocation,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { useProjectRole } from '@/features/auth/useProjectRole'
import { projectDetailQueryOptions } from '@/features/projects/projects.query'

import { Route as ProjectMembersRoute } from './members'
import { Route as ProjectModelsRoute } from './models'
import { Route as ProjectSettingsRoute } from './settings'

export const Route = createFileRoute('/(auth)/(app)/projects/$projectId')({
  loader: async ({
    context: { queryClient }, params: { projectId },
  }) => {
    await queryClient.ensureQueryData(projectDetailQueryOptions(projectId))
  },
  component: RouteComponent,
})

function RouteComponent() {
  const { projectId } = Route.useParams()
  const { t } = useTranslation()
  const matchRoute = useMatchRoute()
  const currentRole = useProjectRole(projectId)
  const pathname = useLocation({ select: s => s.pathname })

  const isAllowEdit = currentRole && [ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN].includes(currentRole)

  const tabRoutes = linkOptions([
    {
      id: 'models',
      label: t('projects.detail.tabs.models'),
      to: ProjectModelsRoute.to,
      params: { projectId },
    },
    {
      id: 'members',
      label: t('projects.detail.tabs.members'),
      to: ProjectMembersRoute.to,
      params: { projectId },
    },
    ...(isAllowEdit
      ? [{
          id: 'settings',
          label: t('projects.detail.tabs.settings'),
          to: ProjectSettingsRoute.to,
          params: { projectId },
        }]
      : []),
  ])

  const activeTab = tabRoutes.find(tab => pathname && matchRoute({
    to: tab.to,
    pending: true,
  }))?.id || tabRoutes[0].id
  const activeTabLabel = tabRoutes.find(tab => tab.id === activeTab)?.label || tabRoutes[0].label

  return (
    <Box>
      <Space h="lg" />
      <Group gap={8} wrap="nowrap" align="center">
        <ProjectIcon size={32} color="var(--mantine-color-gray-9)" />
        <Group gap={4} wrap="nowrap" align="center">
          <Text size="lg" lh="28px" c="gray.9">
            {projectId}
          </Text>
          <Text size="lg" lh="24px" w="24" c="gray.6" ta="center">
            /
          </Text>
          <Text size="lg" lh="24px" c="gray.9">
            {activeTabLabel}
          </Text>
        </Group>
      </Group>

      <Tabs
        mt={24}
        variant="default"
        color="cyan.6"
        value={activeTab}
      >
        <Tabs.List>
          {tabRoutes.map(({
            id, label, ...linkProps
          }) => {
            return (
              <Tabs.Tab
                key={id}
                value={id}
                component={Link}
                {...linkProps}
              >
                {label}
              </Tabs.Tab>
            )
          })}
        </Tabs.List>
      </Tabs>

      <Box>
        <Outlet />
      </Box>
    </Box>
  )
}
