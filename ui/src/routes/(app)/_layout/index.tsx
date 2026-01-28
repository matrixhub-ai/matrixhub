import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/(app)/_layout/')({
  component: HomeRoute,
  staticData: {
    navName: 'Home',
  },
})

function HomeRoute() {
  return <div>Hello “home”!</div>
}
