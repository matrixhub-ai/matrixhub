import {
  Box, Group, Select, Tabs, Text,
} from '@mantine/core'
import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'
import {
  createFileRoute,
  notFound,
  Outlet,
  useLocation,
  useNavigate,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import ProjectIcon from '@/assets/svgs/project.svg?react'
import SlashIcon from '@/assets/svgs/slash.svg?react'

interface SelectOption {
  value: string
  label: string
}

export const Route = createFileRoute('/(auth)/(app)/projects/$projectId')({
  loader: async ({ params: { projectId } }) => {
    const res = await Projects.ListProjects({
      page: 1,
      pageSize: -1,
    })
    const projectOptionsMap = new Map<string, SelectOption>()

    for (const project of res.projects ?? []) {
      if (!project.name) {
        continue
      }

      projectOptionsMap.set(project.name, {
        value: project.name,
        label: project.name,
      })
    }

    const projectOptions = Array.from(projectOptionsMap.values())

    const hasCurrentProject = projectOptions
      .some(projectOption => projectOption.value === projectId)

    if (!hasCurrentProject) {
      // TODO: replace this with the dedicated 404 page once it's implemented.
      throw notFound()
    }

    return { projectOptions }
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
  const { projectOptions } = Route.useLoaderData()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()

  const activeTabRoute
    = tabRoutes.find(tab => location.pathname.includes(`/${tab.value}`))
      ?? tabRoutes[0]
  const activeTabLabel = t(`projects.detail.tabs.${activeTabRoute.value}`)

  return (
    <Box>
      <Group gap={4} wrap="nowrap">
        <ProjectIcon width={20} height={20} style={{ color: 'var(--mantine-gray-7)' }} />
        <Text size="md" c="gray.8">{t('projects.detail.project')}</Text>
        <Select
          allowDeselect={false}
          data={projectOptions}
          w={160}
          size="xs"
          onChange={(value) => {
            if (!value || value === projectId) {
              return
            }

            navigate({
              to: activeTabRoute.to,
              params: { projectId: value },
            })
          }}
          value={projectId}
        />

        <SlashIcon width={24} height={24} style={{ color: 'var(--mantine-color-dimmed)' }} />

        <Text size="md" c="gray.8">{activeTabLabel}</Text>
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
