import { MantineProvider } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import React, { StrictMode } from 'react'
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

export const MountApp = ({ DevPlaceholder}: { DevPlaceholder?: React.FC }) => (
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <MantineProvider theme={mantineTheme} cssVariablesResolver={cssVariablesResolver}>
        <Notifications position="top-right" />
        <RouterProvider router={router} />
        {DevPlaceholder ? <DevPlaceholder /> : null}
      </MantineProvider>
    </QueryClientProvider>
  </StrictMode>
)

if (import.meta.env.VITE_DEV_DEPLOY) {
  // Dev deploy mode: dynamically import dev app with Service Worker API proxy.
  // This entire branch is dead-code-eliminated in production builds.
  import('./devApp').then(({ mountDevApp }) => {
    mountDevApp(rootElement, MountApp)
  })
} else {
  createRoot(rootElement).render(MountApp({}))
}
