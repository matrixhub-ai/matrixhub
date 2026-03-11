import {
  AppShell,
  Avatar,
  Box,
  Flex,
  Group, Menu,
  NavLink,
  Text, UnstyledButton,
} from '@mantine/core'
import {
  createFileRoute,
  Link, linkOptions,
  Outlet, useMatchRoute,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import ArrowDownIcon from '@/assets/svgs/arrow-down.svg?react'
import DatasetIcon from '@/assets/svgs/dataset.svg?react'
import LogoIcon from '@/assets/svgs/logo.svg?react'
import LogOutIcon from '@/assets/svgs/logout.svg?react'
import ModelIcon from '@/assets/svgs/model.svg?react'
import ProjectIcon from '@/assets/svgs/project.svg?react'
import SettingsIcon from '@/assets/svgs/settings.svg?react'
import UserIcon from '@/assets/svgs/user.svg?react'
import { LanguageSwitcher } from '@/components/LanguageSwitcher'
import { Route as DatasetsRoute } from '@/routes/(auth)/(app)/datasets'
import { Route as CreateDatasetRoute } from '@/routes/(auth)/(app)/datasets/new'
import { Route as ModelsRoute } from '@/routes/(auth)/(app)/models'
import { Route as CreateModelRoute } from '@/routes/(auth)/(app)/models/new'
import { Route as ProfileRoute } from '@/routes/(auth)/(app)/profile'
import { Route as ProjectsRoute } from '@/routes/(auth)/(app)/projects'
import { Route as AdminRoute } from '@/routes/(auth)/admin'

export const Route = createFileRoute('/(auth)')({
  component: AppLayout,
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
        <LogoIcon width={36} />
        <Text
          fw={600}
          size="xl"
          c="#000"
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
    },
    {
      label: t('nav.datasets'),
      icon: DatasetIcon,
      to: DatasetsRoute.to,
    },
    {
      label: t('nav.projectManagement'),
      icon: ProjectIcon,
      to: ProjectsRoute.to,
    },
  ])
  const matchRoute = useMatchRoute()

  return (
    <Group
      gap="var(--mantine-spacing-md)"
      wrap="nowrap"
    >
      {navRoutes.map((route) => {
        const isActive = !!matchRoute({
          to: route.to,
          fuzzy: true,
        })

        return (
          <NavLink
            key={route.to}
            label={route.label}
            component={Link}
            to={route.to}
            leftSection={<route.icon width={20} />}
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
            gap="var(--mantine-spacing-sm)"
            wrap="nowrap"
          >
            <Avatar
              radius="xl"
              size={24}
            />
            <Text size="sm">
              Admin
            </Text>
            <ArrowDownIcon width={16} />
          </Group>
        </UnstyledButton>
      </Menu.Target>

      <Menu.Dropdown>
        {
          menuItems.map(item => (
            <Menu.Item
              key={item.label}
              leftSection={<item.icon width={16} color="#868E96" />}
              component={Link}
              to={item.to}
            >
              {item.label}
            </Menu.Item>
          ))
        }
        <Menu.Item
          leftSection={<LogOutIcon width={16} />}
        >
          {t('nav.logout')}
        </Menu.Item>
      </Menu.Dropdown>
    </Menu>
  )
}

function AppLayout() {
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

          <Group gap="lg">
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
        <Box
          style={{
            width: '86vw',
            height: '100%',
            maxWidth: '1760px',
            minWidth: '1100px',
            margin: '0 auto',
            padding: '0 32px',
            boxSizing: 'content-box',
          }}
        >
          <Box pt={20} h="100%">
            <Outlet />
          </Box>
        </Box>
      </AppShell.Main>
    </AppShell>
  )
}
