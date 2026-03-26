import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/(auth)/(app)/projects/$projectId/')({
  beforeLoad: ({ params: { projectId } }) => {
    throw redirect({
      to: '/projects/$projectId/models',
      params: { projectId },
    })
  },
})
