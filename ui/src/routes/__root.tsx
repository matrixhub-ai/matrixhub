import {
  createRootRouteWithContext, Outlet, HeadContent,
} from '@tanstack/react-router'
import { lazy, Suspense } from 'react'

import { CurrentUserContext } from '@/context/current-user-context.tsx'
import { ProjectRolesContext } from '@/context/project-role-context'
import { currentUserQueryOptions, projectRolesQueryOptions } from '@/features/auth/auth.query'
import i18n from '@/i18n'
import { queryClient } from '@/queryClient'

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
    try {
      const [user, projectRoles] = await Promise.all([
        queryClient.ensureQueryData(currentUserQueryOptions()),
        queryClient.ensureQueryData(projectRolesQueryOptions()),
      ])

      return {
        user,
        projectRoles,
      }
    } catch {
      return {
        user: undefined,
        projectRoles: undefined,
      }
    }
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
  const {
    user, projectRoles,
  } = Route.useLoaderData()

  return (
    <>
      <HeadContent />
      <CurrentUserContext value={user}>
        <ProjectRolesContext value={projectRoles}>
          <Outlet />
        </ProjectRolesContext>
      </CurrentUserContext>
      <Suspense fallback={null}>
        <TanStackDevtools />
      </Suspense>
    </>
  )
}
