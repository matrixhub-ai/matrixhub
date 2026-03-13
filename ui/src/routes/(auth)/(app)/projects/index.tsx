import { createFileRoute } from '@tanstack/react-router'

import { ProjectsPage } from '@/features/projects/pages/ProjectsPage'

export const Route = createFileRoute('/(auth)/(app)/projects/')({
  component: RouteComponent,
  staticData: {
    navName: 'Projects',
  },
})

function RouteComponent() {
  return <ProjectsPage />
}
