import { CurrentUser } from '@matrixhub/api-ts/v1alpha1/current_user.pb.ts'
import {
  createRootRoute, Outlet, HeadContent,
} from '@tanstack/react-router'

import { CurrentUserContext } from '@/context/current-user-context.tsx'
import i18n from '@/i18n'

export const Route = createRootRoute({
  loader: async () => {
    // TODO: Add error handling
    return await CurrentUser.GetCurrentUser({})
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
    </>
  )
}
