import { createRouter } from '@tanstack/react-router'
import { routeTree } from './routeTree.gen.ts'

const rawBasePath = import.meta.env.VITE_UI_BASE_PATH ?? '/'

export const router = createRouter({
  routeTree,
  scrollRestoration: true,
  basepath: rawBasePath,
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
