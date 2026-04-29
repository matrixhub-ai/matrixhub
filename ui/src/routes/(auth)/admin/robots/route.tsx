import { Outlet, createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/(auth)/admin/robots')({
  component: RouteComponent,
})

function RouteComponent() {
  return <Outlet />
}
