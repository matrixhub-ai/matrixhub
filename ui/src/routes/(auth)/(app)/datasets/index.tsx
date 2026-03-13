import { Flex } from '@mantine/core'
import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/(auth)/(app)/datasets/')({
  component: RouteComponent,
  staticData: {
    navName: 'Datasets',
  },
})

function RouteComponent() {
  return (
    <Flex h="100%">
      <div style={{
        width: 360,
        flexShrink: 0,
      }}
      >
        search
      </div>
      <div style={{
        flex: 1,
        minWidth: 0,
      }}
      >
        List
      </div>
    </Flex>
  )
}
