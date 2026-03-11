import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/(auth)/(app)/profile/access-token')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/(auth)/(app)/profile/access-token"!</div>
}
