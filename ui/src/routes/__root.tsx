import { createRootRoute, Outlet, HeadContent } from '@tanstack/react-router'

import i18n from '@/i18n'

export const Route = createRootRoute({
  component: () => (
    <>
      <HeadContent />
      <Outlet />
    </>
  ),
  head: () => ({
    meta: [{
      title: i18n.t('translation.title'),
    },
    ],
    links: [
      { rel: 'icon', href: '/favicon.ico?' },
    ],
  }),
})
