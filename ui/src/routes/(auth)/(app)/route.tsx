import { Box } from '@mantine/core'
import {
  createFileRoute,
  Outlet,
} from '@tanstack/react-router'

export const Route = createFileRoute('/(auth)/(app)')({
  component: AppLayout,
})

function AppLayout() {
  return (
    <Box
      style={{
        width: '86vw',
        height: '100%',
        maxWidth: '1760px',
        minWidth: '1100px',
        margin: '0 auto',
        padding: '0 32px',
        boxSizing: 'content-box',
      }}
    >
      <Outlet />
    </Box>
  )
}
