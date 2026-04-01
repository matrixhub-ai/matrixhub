import { Box, ScrollArea } from '@mantine/core'
import {
  createFileRoute,
  Outlet,
} from '@tanstack/react-router'

import { setContentViewport } from '@/utils/setContentViewport'
export const Route = createFileRoute('/(auth)/(app)')({
  component: AppLayout,
})

function AppLayout() {
  return (
    <ScrollArea
      h="100%"
      type="auto"
      viewportRef={setContentViewport}
    >
      <Box
        w="86vw"
        h="100%"
        maw="1760px"
        px="32px"
        py="0"
        my="0"
        mx="auto"
        miw="1100px"
        style={{
          boxSizing: 'content-box',
        }}
      >
        <Outlet />
      </Box>
    </ScrollArea>
  )
}
