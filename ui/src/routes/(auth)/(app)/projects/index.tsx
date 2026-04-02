import { createFileRoute, stripSearchParams } from '@tanstack/react-router'
import { z } from 'zod'

import { ProjectsPage } from '@/features/projects/pages/ProjectsPage'
import { projectsQueryOptions } from '@/features/projects/projects.query'

const DEFAULT_PROJECTS_PAGE = 1

const defaults = {
  page: DEFAULT_PROJECTS_PAGE,
  query: '',
}

const pSearchParamSchema = z.object({
  page: z.coerce.number().int().nonnegative().optional().default(defaults.page).catch(defaults.page),
  query: z.string().trim().optional().default(defaults.query).catch(defaults.query),
})

export const Route = createFileRoute('/(auth)/(app)/projects/')({
  component: RouteComponent,
  validateSearch: pSearchParamSchema,
  search: {
    middlewares: [stripSearchParams(defaults)],
  },
  loaderDeps: ({ search }) => ({
    page: search.page,
    query: search.query,
  }),
  loader: ({
    context, deps,
  }) => {
    void context.queryClient.ensureQueryData(
      projectsQueryOptions({
        query: deps.query ?? '',
        page: deps.page ?? DEFAULT_PROJECTS_PAGE,
      }),
    )
  },
  staticData: {
    navName: 'Projects',
  },
})

function RouteComponent() {
  return <ProjectsPage />
}
