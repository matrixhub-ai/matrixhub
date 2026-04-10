import { createRouter } from '@tanstack/react-router'

import { queryClient } from './queryClient'
import { routeTree } from './routeTree.gen.ts'
import { RouterPendingComponent } from './shared/components/RouterPendingComponent'
import { adminContentViewportSelector, contentViewportSelector } from './utils/setContentViewport'

const rawBasePath = import.meta.env.VITE_UI_BASE_PATH ?? '/'

export const router = createRouter({
  routeTree,
  scrollRestoration: true,
  scrollToTopSelectors: [
    contentViewportSelector,
    adminContentViewportSelector,
  ],
  basepath: rawBasePath,
  context: {
    queryClient,
  },
  defaultPendingComponent: RouterPendingComponent,
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
