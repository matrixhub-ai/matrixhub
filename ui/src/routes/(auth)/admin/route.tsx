import {
  AppShell,
  Group,
  NavLink,
  ScrollArea,
  Stack,
  Text,
  rem,
} from '@mantine/core'
import { IconSettings as SettingsIcon } from '@tabler/icons-react'
import {
  createFileRoute,
  Link,
  linkOptions,
  Outlet,
  useMatchRoute,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { currentUserQueryOptions } from '@/features/auth/auth.query'
import { queryClient } from '@/queryClient'
import { Route as AdminRegistriesRoute, Icon as AdminRegistriesIcon } from '@/routes/(auth)/admin/registries'
import { Route as AdminReplicationsRoute, Icon as AdminReplicationsIcon } from '@/routes/(auth)/admin/replications'
import { Route as AdminUsersRoute, Icon as AdminUsersIcon } from '@/routes/(auth)/admin/users'
import { forbiddenError } from '@/utils/routerAccess'
import { setAdminContentViewport } from '@/utils/setContentViewport'

export const Route = createFileRoute('/(auth)/admin')({
  beforeLoad: async () => {
    const user = await queryClient.ensureQueryData(currentUserQueryOptions())

    if (!user.isAdmin) {
      throw forbiddenError()
    }
  },
  component: AdminLayout,
  staticData: {
    navName: 'Admin',
  },
})

function AdminNavbar() {
  const { t } = useTranslation()
  const navRoutes = linkOptions([
    {
      label: t('admin.users'),
      icon: AdminUsersIcon,
      to: AdminUsersRoute.to,
    },
    {
      label: t('admin.registries'),
      icon: AdminRegistriesIcon,
      to: AdminRegistriesRoute.to,
    },
    {
      label: t('admin.replications'),
      icon: AdminReplicationsIcon,
      to: AdminReplicationsRoute.to,
    },
  ])

  const matchRoute = useMatchRoute()

  return (
    <ScrollArea
      type="scroll"
      offsetScrollbars="y"
      style={{
        flex: 1,
        minHeight: 0,
      }}
      scrollbarSize={0}
    >
      <Stack gap="sm" pb="lg">
        {navRoutes.map((route) => {
          const Icon = route.icon
          const isActive = !!matchRoute({
            to: route.to,
            fuzzy: true,
          })

          return (
            <NavLink
              key={route.to}
              label={route.label}
              leftSection={<Icon size={rem(16)} />}
              component={Link}
              to={route.to}
              active={isActive}
              h={32}
              fs="sm"
              bdrs="sm"
              styles={{
                label: {
                  whiteSpace: 'nowrap',
                },
              }}
            />
          )
        })}
      </Stack>
    </ScrollArea>
  )
}

function AdminLayout() {
  const { t } = useTranslation()

  return (
    <AppShell
      mode="static"
      navbar={{
        width: 220,
        breakpoint: '0em',
      }}
    >
      <AppShell.Navbar
        component="nav"
        mih={0}
        my="lg"
        px="md"
        h="calc(100% - var(--mantine-spacing-lg) * 2)"
        style={{
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        {/* FIXME: color: Gray80 */}
        <Group
          h={30}
          wrap="nowrap"
          pl={4}
          gap={6}
          c="gray.7"
          mb="lg"
        >
          <SettingsIcon
            size={rem(30)}
            style={{ flexShrink: 0 }}
          />
          <Text
            size="md"
            fw={600}
            lh={rem(24)}
            truncate
          >
            {t('nav.settings')}
          </Text>
        </Group>

        <AdminNavbar />
      </AppShell.Navbar>

      <AppShell.Main
        component="div"
        miw={0}
        mih={0}
        h="100%"
        style={{
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <ScrollArea
          type="auto"
          mih={0}
          flex={1}
          viewportRef={setAdminContentViewport}
        >
          <Outlet />
        </ScrollArea>
      </AppShell.Main>
    </AppShell>
  )
}
