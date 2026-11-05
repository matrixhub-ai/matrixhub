import { createFileRoute, getRouteApi } from '@tanstack/react-router'

import { ModelSecurityPage } from '@/features/models/pages/ModelSecurityPage'

const {
  useLoaderData, useParams,
} = getRouteApi('/(auth)/(app)/projects_/$projectId/models/$modelId')

function SecurityTab() {
  const {
    projectId, modelId,
  } = useParams()
  const { model } = useLoaderData()

  return (
    <ModelSecurityPage
      project={model.project ?? projectId}
      name={model.name?.trim() || modelId}
      revision={model.defaultBranch ?? 'main'}
    />
  )
}

export const Route = createFileRoute('/(auth)/(app)/projects_/$projectId/models/$modelId/security/')({
  component: SecurityTab,
})
