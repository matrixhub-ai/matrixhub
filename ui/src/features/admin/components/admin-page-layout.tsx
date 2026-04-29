import { Box } from '@mantine/core'

import type { ReactNode } from 'react'

interface AdminPageLayoutProps {
  children?: ReactNode
}

export function AdminPageLayout({
  children,
}: AdminPageLayoutProps) {
  return (
    <Box component="section" px="xl">
      {children}
    </Box>
  )
}
