import { MantineProvider } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import './i18n/index.ts'
import '@mantine/core/styles.css'
import '@mantine/notifications/styles.css'
// dates styles after core package styles
import '@mantine/dates/styles.css'
import './index.css'
import { mantineTheme, cssVariablesResolver } from './mantineTheme'
import { queryClient } from './queryClient'
import { router } from './router.tsx'

const rootElement = document.getElementById('root')

if (!rootElement) {
  throw new Error('Root element not found')
}

<<<<<<< HEAD
createRoot(rootElement).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <MantineProvider theme={mantineTheme} cssVariablesResolver={cssVariablesResolver}>
        <Notifications position="top-right" />
        <RouterProvider router={router} />
      </MantineProvider>
    </QueryClientProvider>
  </StrictMode>,
)
=======
if (import.meta.env.VITE_DEV_DEPLOY) {
  // Dev deploy mode: dynamically import dev app with Service Worker API proxy.
  // This entire branch is dead-code-eliminated in production builds.
  import('./features/dev-deploy/DevApp').then(({ mountDevApp }) => {
    mountDevApp(rootElement)
  })
} else {
  createRoot(rootElement).render(
    <StrictMode>
      <MantineProvider theme={mantineTheme}>
        <RouterProvider router={router} />
      </MantineProvider>
    </StrictMode>,
  )
}
>>>>>>> 9c3f91d (chore(ui): support deploy pr preview)
