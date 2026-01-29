import {
  AppShell,
  Title,
  NavLink,
  ScrollArea,
  Stack,
  Text,
} from '@mantine/core'
import {
  createFileRoute,
  Link,
  Outlet,
  useRouter,
  useRouterState,
  type AnyRoute,
} from '@tanstack/react-router'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

export const Route = createFileRoute('/(app)/_layout')({
  component: AppLayout,
})

function AppNavbar() {
  const router = useRouter()
  const activeRouteIds = useRouterState({
    select: state => state.matches.map(match => match.routeId),
  })

  const layoutRoute = router.routesById['/(app)/_layout']
  const navRoutes = useMemo(() => {
    const children = layoutRoute.children
    if (!children) return []
    return (Object.values(children) as AnyRoute[])
      .filter(route => route.options.staticData?.navName !== undefined)
  }, [layoutRoute])

  if (!navRoutes.length) {
    return (
      <Text size="sm" c="dimmed">
        No navigation routes available
      </Text>
    )
  }

  return (
    <ScrollArea h="100%">
      <Stack gap={0}>
        {navRoutes.map((route) => {
          const isActive = activeRouteIds.includes(route.id as typeof activeRouteIds[number])
          return (
            <NavLink
              key={route.id as string}
              label={route.options.staticData?.navName ?? route.id as string}
              component={Link}
              to={route.to as string}
              active={isActive}
            />
          )
        })}
      </Stack>
    </ScrollArea>
  )
}

function AppLayout() {
  const { t } = useTranslation()
  return (
    <AppShell
      padding="md"
      header={{ height: 60 }}
      navbar={{
        width: 300,
        breakpoint: '',
      }}
    >
      <AppShell.Header p="md">
        <Title order={2}>{t('translation.title')}</Title>
      </AppShell.Header>

      <AppShell.Navbar>
        <AppNavbar />
      </AppShell.Navbar>

      <AppShell.Main>
        <Outlet />
      </AppShell.Main>
    </AppShell>
  )
}
