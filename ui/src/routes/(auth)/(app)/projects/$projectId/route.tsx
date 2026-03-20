import {
  Box, Group, Space, Tabs, Text,
} from '@mantine/core'
import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'
import { IconApiApp as ProjectIcon } from '@tabler/icons-react'
import {
  createFileRoute,
  notFound,
  Outlet,
  useLocation,
  useMatchRoute,
  useNavigate,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

export const Route = createFileRoute('/(auth)/(app)/projects/$projectId')({
  loader: async ({ params: { projectId } }) => {
    try {
      const res = await Projects.GetProject({
        name: projectId,
      })

      if (!res) {
        // TODO: replace this with the dedicated 404 page once it's implemented.
        throw notFound()
      }

      return { project: res }
    } catch (error) {
      console.error('Failed to load project data:', error)
      throw notFound()
    }
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

  const activeTabRoute
    = tabRoutes.find((tab) => {
      return pathname && !!matchRoute({
        to: tab.to,
        params: { projectId },
        fuzzy: true,
        pending: true,
      })
    })
    ?? tabRoutes[0]
  const activeTabLabel = t(`projects.detail.tabs.${activeTabRoute.value}`)

  return (
    <Box>
      <Space h="lg" />
      <Group gap={4} wrap="nowrap">
        <ProjectIcon size={20} style={{ color: 'var(--mantine-gray-7)' }} />
        <Text size="md" c="gray.8">
          {projectId}
          {' '}
          /
          {' '}
          {activeTabLabel}
        </Text>
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
          {tabRoutes.map((tab) => {
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
