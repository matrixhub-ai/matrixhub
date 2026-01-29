import { MantineProvider } from '@mantine/core'
import { RouterProvider } from '@tanstack/react-router'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import './i18n/index.ts'
import '@mantine/core/styles.css'
import './index.css'
import { mantineTheme } from './mantineTheme'
import { router } from './router.tsx'

const root = document.getElementById('root')

if (!root) {
  throw new Error('Root container missing in index.html')
}

createRoot(root).render(
  <StrictMode>
    <MantineProvider theme={mantineTheme}>
      <RouterProvider router={router} />
    </MantineProvider>
  </StrictMode>,
)
