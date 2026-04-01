import { createRouter } from '@tanstack/react-router'

import { queryClient } from './queryClient'
import { routeTree } from './routeTree.gen.ts'
import { RouterErrorComponent } from './shared/components/RouterErrorComponent'
import { RouterPendingComponent } from './shared/components/RouterPendingComponent'

const rawBasePath = import.meta.env.VITE_UI_BASE_PATH ?? '/'

export const router = createRouter({
  routeTree,
  scrollRestoration: true,
  scrollToTopSelectors: [
    '[data-scroll-restoration-id="admin-content-scroll"]',
  ],
  basepath: rawBasePath,
  context: {
    queryClient,
  },
  defaultErrorComponent: RouterErrorComponent,
  defaultPendingComponent: RouterPendingComponent,
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
