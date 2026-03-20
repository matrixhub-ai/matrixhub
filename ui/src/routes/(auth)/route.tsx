import {
  AppShell,
  Avatar,
  Flex,
  Group,
  Menu,
  NavLink,
  Text,
  UnstyledButton,
  rem,
} from '@mantine/core'
import {
  IconChevronDown as ArrowDownIcon,
  IconCube as ModelIcon,
  IconDatabase as DatasetIcon,
  IconLogout as LogOutIcon,
  IconSettings as SettingsIcon,
  IconUser as UserIcon,
  IconApiApp as ProjectIcon,
} from '@tabler/icons-react'
import {
  createFileRoute,
  Link,
  linkOptions,
  Outlet,
  useMatchRoute,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import LogoIcon from '@/assets/svgs/logo.svg?react'
import { Route as DatasetsRoute } from '@/routes/(auth)/(app)/datasets'
import { Route as CreateDatasetRoute } from '@/routes/(auth)/(app)/datasets/new'
import { Route as ModelsRoute } from '@/routes/(auth)/(app)/models'
import { Route as CreateModelRoute } from '@/routes/(auth)/(app)/models/new'
import { Route as ProfileRoute } from '@/routes/(auth)/(app)/profile'
import { Route as ProjectsRoute } from '@/routes/(auth)/(app)/projects'
import { Route as ProjectDatasetRoute } from '@/routes/(auth)/(app)/projects_.$projectId/datasets.$datasetId/route'
import { Route as ProjectModelRoute } from '@/routes/(auth)/(app)/projects_.$projectId/models.$modelId/route'
import { Route as AdminRoute } from '@/routes/(auth)/admin'
import { LanguageSwitcher } from '@/shared/components/LanguageSwitcher'

export const Route = createFileRoute('/(auth)')({
  component: AuthLayout,
})

function AppLogo() {
  return (
    <UnstyledButton
      component={Link}
      to="/"
    >
      <Group
        gap={8}
        wrap="nowrap"
      >
        <LogoIcon fontSize={rem(36)} />
        <Text
          fw={600}
          size="xl"
          c="var(--mantine-color-black)"
        >
          MatrixHub
        </Text>
      </Group>
    </UnstyledButton>
  )
}

function AppNavbar() {
  const { t } = useTranslation()
  const navRoutes = linkOptions([
    {
      label: t('nav.models'),
      icon: ModelIcon,
      to: ModelsRoute.to,
      extraMatch: ProjectModelRoute.to,
    },
    {
      label: t('nav.datasets'),
      icon: DatasetIcon,
      to: DatasetsRoute.to,
      extraMatch: ProjectDatasetRoute.to,
    },
    {
      label: t('nav.projectManagement'),
      icon: ProjectIcon,
      to: ProjectsRoute.to,
      repelMatch: [ProjectModelRoute.to, ProjectDatasetRoute.to],
    },
  ])
  const matchRoute = useMatchRoute()

  return (
    <Group
      gap="sm"
      wrap="nowrap"
    >
      {navRoutes.map((route) => {
        let isActive = !!matchRoute({
          to: route.to,
          fuzzy: true,
        })

        if (isActive && 'repelMatch' in route) {
          isActive = !route.repelMatch.some(repelRoute => matchRoute({
            to: repelRoute,
            fuzzy: true,
          }))
        } else if (!isActive && 'extraMatch' in route) {
          isActive = !!matchRoute({
            to: route.extraMatch,
            fuzzy: true,
          })
        }

        return (
          <NavLink
            key={route.to}
            label={route.label}
            component={Link}
            to={route.to}
            activeOptions={{ exact: true }}
            leftSection={(
              <route.icon
                size={rem(20)}
              />
            )}
            active={isActive}
            styles={{
              root: {
                width: 'auto',
                height: '32px',
                borderRadius: 'var(--mantine-radius-lg)',
                fontWeight: '600',
                color: isActive ? 'var(--nl-color)' : '#868E96',
              },
              section: {
                marginInlineEnd: '8px',
              },
              label: {
                whiteSpace: 'nowrap',
              },
            }}
          />
        )
      })}
    </Group>
  )
}

function AccountMenu() {
  const { t } = useTranslation()
  const menuItems = linkOptions([
    {
      label: t('nav.profile'),
      icon: UserIcon,
      to: ProfileRoute.to,
    },
    {
      label: t('nav.createModel'),
      icon: ModelIcon,
      to: CreateModelRoute.to,
    },
    {
      label: t('nav.createDataset'),
      icon: DatasetIcon,
      to: CreateDatasetRoute.to,
    },
    {
      label: t('nav.settings'),
      icon: SettingsIcon,
      to: AdminRoute.to,
    },
  ])

  return (
    <Menu
      shadow="md"
      withArrow
    >
      <Menu.Target>
        <UnstyledButton>
          <Group
            gap={8}
            wrap="nowrap"
          >
            <Avatar
              radius="xl"
              size={24}
            />
            <Text size="sm">
              Admin
            </Text>
            <ArrowDownIcon
              size={rem(16)}
            />
          </Group>
        </UnstyledButton>
      </Menu.Target>

      <Menu.Dropdown>
        {
          menuItems.map(item => (
            <Menu.Item
              key={item.label}
              leftSection={(
                <item.icon
                  size={rem(16)}
                  color="var(--mantine-color-gray-6)"
                />
              )}
              component={Link}
              to={item.to}
            >
              {item.label}
            </Menu.Item>
          ))
        }
        <Menu.Item
          leftSection={(
            <LogOutIcon
              size={rem(16)}
              color="var(--mantine-color-gray-6)"
            />
          )}
        >
          {t('nav.logout')}
        </Menu.Item>
      </Menu.Dropdown>
    </Menu>
  )
}

function AuthLayout() {
  return (
    <AppShell
      mode="static"
      header={{ height: 60 }}
    >
      <AppShell.Header
        withBorder={false}
        style={{ background: '#F8F9FA' }}
      >
        <Flex
          h="100%"
          align="center"
          justify="space-between"
          px={24}
        >
          <Group
            gap={135}
            wrap="nowrap"
          >
            <AppLogo />

            <AppNavbar />
          </Group>

          <Group gap="md" wrap="nowrap">
            <LanguageSwitcher />

            <AccountMenu />
          </Group>
        </Flex>
      </AppShell.Header>

      <AppShell.Main
        styles={{
          main: {
            height: 'calc(100vh - var(--app-shell-header-height))',
          },
        }}
      >
        <Outlet />
      </AppShell.Main>
    </AppShell>
  )
}
