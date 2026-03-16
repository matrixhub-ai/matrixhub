import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/(auth)/(app)/profile/language')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/(auth)/(app)/profile/language"!</div>
}
