import { createFileRoute } from '@tanstack/react-router'

import { ModelBlobPage } from '@/features/models/pages/ModelBlobPage.tsx'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId/blob/$ref/$',
)({
  component: ModelBlobPage,
})
