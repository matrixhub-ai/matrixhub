import {
  Box, Group, Space, Tabs, Text,
} from '@mantine/core'
import { ProjectRoleType } from '@matrixhub/api-ts/v1alpha1/role.pb'
import { IconApiApp as ProjectIcon } from '@tabler/icons-react'
import {
  createFileRoute,
  Outlet,
  useLocation,
  useMatchRoute,
  useNavigate,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { useProjectRole } from '@/features/auth/auth.query'
import { projectDetailQueryOptions } from '@/features/projects/projects.query'

export const Route = createFileRoute('/(auth)/(app)/projects/$projectId')({
  loader: async ({
    context: { queryClient }, params: { projectId },
  }) => {
    await queryClient.ensureQueryData(projectDetailQueryOptions(projectId))
  },
  component: RouteComponent,
})

const tabRoutes = [
  {
    value: 'models',
    to: '/projects/$projectId/models',
  },
  {
    value: 'datasets',
    to: '/projects/$projectId/datasets',
  },
  {
    value: 'members',
    to: '/projects/$projectId/members',
  },
  {
    value: 'settings',
    to: '/projects/$projectId/settings',
  },
] as const

function RouteComponent() {
  const { projectId } = Route.useParams()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const matchRoute = useMatchRoute()
  const pathname = useLocation({ select: s => s.pathname })
  const currentRole = useProjectRole(projectId)

  const isAllowEdit = currentRole && [ProjectRoleType.ROLE_TYPE_PROJECT_ADMIN].includes(currentRole)

  const filteredTabRoutes = isAllowEdit
    ? tabRoutes
    : tabRoutes.filter(tab => tab.value !== 'settings')

  const activeTabRoute
    = filteredTabRoutes.find((tab) => {
      return pathname && !!matchRoute({
        to: tab.to,
        params: { projectId },
        fuzzy: true,
        pending: true,
      })
    })
    ?? filteredTabRoutes[0]
  const activeTabLabel = t(`projects.detail.tabs.${activeTabRoute.value}`)

  return (
    <Box>
      <Space h="lg" />
      <Group gap={8} wrap="nowrap" align="center">
        <ProjectIcon size={32} style={{ color: 'var(--mantine-color-gray-9)' }} />
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
        value={activeTabRoute.value}
        onChange={(value) => {
          const tab = tabRoutes.find(r => r.value === value)

          if (tab) {
            navigate({
              to: tab.to,
              params: { projectId },
            })
          }
        }}
      >
        <Tabs.List style={{ gap: 'var(--mantine-spacing-md)' }}>
          {filteredTabRoutes.map((tab) => {
            const isActive = tab.value === activeTabRoute.value

            return (
              <Tabs.Tab
                key={tab.value}
                value={tab.value}
                c={isActive ? 'gray.7' : 'gray.6'}
                p="8px var(--mantine-spacing-md)"
                fw="600"
                fz="var(--mantine-font-size-sm)"
                lh="sm"
              >
                {t(`projects.detail.tabs.${tab.value}`)}
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
