import { Box, ScrollArea } from '@mantine/core'
import {
  createFileRoute,
  Outlet,
  useMatchRoute,
} from '@tanstack/react-router'

import { setContentViewport } from '@/utils/setContentViewport'
export const Route = createFileRoute('/(auth)/(app)')({
  component: AppLayout,
})

function AppLayout() {
  const matchRoute = useMatchRoute()
  const isModelCreate = matchRoute({ to: '/models/new' })

  return (
    <ScrollArea
      h="100%"
      type="auto"
      viewportRef={setContentViewport}
    >
      <Box
        w={isModelCreate ? 'auto' : '86vw'}
        h="100%"
        maw="1760px"
        px={isModelCreate ? 'md' : '32px'}
        py="0"
        my="0"
        mx="auto"
        miw={isModelCreate ? undefined : '1100px'}
        style={{
          boxSizing: 'content-box',
        }}
      >
        <Outlet />
      </Box>
    </ScrollArea>
  )
}
