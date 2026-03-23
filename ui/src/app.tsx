import { MantineProvider } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import './i18n/index.ts'
import '@mantine/core/styles.css'
import '@mantine/notifications/styles.css'
import '@mantine/dates/styles.css'
import './index.css'
import { mantineTheme, cssVariablesResolver } from './mantineTheme'
import { queryClient } from './queryClient'
import { router } from './router.tsx'

export function mountApp(rootElement: HTMLElement) {
  const appRoot = createRoot(rootElement)

  appRoot.render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <MantineProvider theme={mantineTheme} cssVariablesResolver={cssVariablesResolver}>
          <Notifications position="top-right" />
          <RouterProvider router={router} />
        </MantineProvider>
      </QueryClientProvider>
    </StrictMode>,
  )

  return appRoot
}
