import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from '@tanstack/react-router'
import { MantineProvider } from '@mantine/core';
import './i18n'
import '@mantine/core/styles.css';
import './index.css'
import { router } from './router.tsx'
import { mantineTheme } from './mantineTheme';


createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <MantineProvider theme={mantineTheme}>
      <RouterProvider router={router} />
    </MantineProvider>
  </StrictMode>,
)
