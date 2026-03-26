import { createFileRoute } from '@tanstack/react-router'

import ModelTreePage from '@/features/models/pages/ModelTreePage.tsx'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId/tree/$ref/$',
)({
  component: ModelTreePage,
})
