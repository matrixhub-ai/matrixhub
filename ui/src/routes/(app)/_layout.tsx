import {
  createFileRoute,
  Outlet,
  useRouter,
  useNavigate
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  AppShell,
  Title,
  NavLink,
} from '@mantine/core'

export const Route = createFileRoute('/(app)/_layout')({
  component: AppLayout,
})


function AppNavbar() {
  const router = useRouter()
  const navigate = useNavigate()

  const layoutRoute = router.routesById['/(app)/_layout']
 console.log('layoutRoute', layoutRoute)

  return (
    <>
        <NavLink label="Home"   onClick={() => navigate({ to: '/' })}  />
        <NavLink label="About"  onClick={() => navigate({ to: '/about' })} />
    </>
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
        breakpoint: 'md',
      }}
    >
      <AppShell.Header p='md'>
          <Title order={2}>{t('title')}</Title>

      </AppShell.Header>

      <AppShell.Navbar>
        <AppNavbar />
      </AppShell.Navbar>

      <AppShell.Main><Outlet></Outlet></AppShell.Main>
    </AppShell>
  )
}
