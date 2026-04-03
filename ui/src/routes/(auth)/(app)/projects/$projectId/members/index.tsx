import { createFileRoute, stripSearchParams } from '@tanstack/react-router'

import {
  membersQueryOptions,
  membersSearchDefaults,
  membersSearchSchema,
} from '@/features/projects/members/members.query'
import { ProjectMembersPage } from '@/features/projects/members/pages/ProjectMembersPage'

// -- Route definition --

export const Route = createFileRoute(
  '/(auth)/(app)/projects/$projectId/members/',
)({
  validateSearch: membersSearchSchema,
  search: {
    middlewares: [stripSearchParams(membersSearchDefaults)],
  },
  loaderDeps: ({ search }) => search,
  loader: ({
    context,
    params,
    deps,
  }) => {
    context.queryClient.prefetchQuery(
      membersQueryOptions(params.projectId, deps),
    )
  },
  component: ProjectMembersPage,
})
