import {
  createRootRouteWithContext, Outlet, HeadContent,
} from '@tanstack/react-router'
import { lazy, Suspense } from 'react'

import { CurrentUserContext } from '@/context/current-user-context'
import i18n from '@/i18n'
import { AuthProvider } from '@/provider/auth'

import type { QueryClient } from '@tanstack/react-query'

const TanStackDevtools = import.meta.env.DEV
  ? lazy(() =>
      Promise.all([
        import('@tanstack/react-devtools'),
        import('@tanstack/react-query-devtools'),
        import('@tanstack/react-router-devtools'),
        import('@tanstack/react-form-devtools'),
      ]).then(([devtools, queryDevtools, routerDevtools, formDevtools]) => ({
        default: () => (
          <devtools.TanStackDevtools
            plugins={[
              {
                name: 'TanStack Query',
                render: <queryDevtools.ReactQueryDevtoolsPanel />,
              },
              {
                name: 'TanStack Router',
                render: <routerDevtools.TanStackRouterDevtoolsPanel />,
              },
              formDevtools.formDevtoolsPlugin(),
            ]}
          />
        ),
      })),
    )
  : () => null

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient
}>()({
  loader: async () => {
    // TODO: Add error handling
    // return await CurrentUser.GetCurrentUser({})
    return { username: 'Admin' }
  },
  component: RootComponent,
  head: () => ({
    meta: [{
      title: i18n.t('translation.title'),
    }],
    links: [
      {
        rel: 'icon',
        href: '/favicon.ico?',
      },
    ],
  }),
})

function RootComponent() {
  const user = Route.useLoaderData()

  return (
    // TODO FIX
    <AuthProvider>
      <HeadContent />
      <CurrentUserContext value={user}>
        <Outlet />
      </CurrentUserContext>
      <Suspense fallback={null}>
        <TanStackDevtools />
      </Suspense>
    </AuthProvider>
  )
}
