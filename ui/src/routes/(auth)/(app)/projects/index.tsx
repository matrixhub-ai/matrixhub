import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'
import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { ProjectsPage } from '@/features/projects/pages/ProjectsPage'

const DEFAULT_PROJECTS_PAGE = 1
const DEFAULT_PROJECTS_PAGE_SIZE = 10

const pSearchParamSchema = z.object({
  page: z.coerce.number().int().nonnegative().optional().catch(DEFAULT_PROJECTS_PAGE),
  query: z.string().trim().optional().catch(undefined),
})

export const Route = createFileRoute('/(auth)/(app)/projects/')({
  component: RouteComponent,
  validateSearch: pSearchParamSchema,
  loaderDeps: ({ search }) => ({
    page: search.page,
    query: search.query,
  }),
  loader: async ({ deps }) => {
    const response = await Projects.ListProjects({
      name: deps.query || undefined,
      page: deps.page,
      pageSize: DEFAULT_PROJECTS_PAGE_SIZE,
    })

    return {
      projects: response.projects ?? [],
      pagination: response.pagination,
    }
  },
  staticData: {
    navName: 'Projects',
  },
})

function RouteComponent() {
  return <ProjectsPage />
}
