import {
  createRootRoute, Outlet, HeadContent,
} from '@tanstack/react-router'
import { lazy, Suspense } from 'react'

import { CurrentUserContext } from '@/context/current-user-context.tsx'
import i18n from '@/i18n'

const TanStackRouterDevtools = import.meta.env.DEV
  ? lazy(() =>
      import('@tanstack/react-router-devtools').then(m => ({
        default: m.TanStackRouterDevtools,
      })),
    )
  : () => null

export const Route = createRootRoute({
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
    <>
      <HeadContent />
      <CurrentUserContext value={user}>
        <Outlet />
      </CurrentUserContext>
      <Suspense fallback={null}>
        <TanStackRouterDevtools initialIsOpen={false} />
      </Suspense>
    </>
  )
}
