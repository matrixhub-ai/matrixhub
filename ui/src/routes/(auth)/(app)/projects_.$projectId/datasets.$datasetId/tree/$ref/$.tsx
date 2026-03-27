import { Text } from '@mantine/core'
import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/datasets/$datasetId/tree/$ref/$',
)({
  component: DatasetTreePage,
})

function DatasetTreePage() {
  const {
    datasetId, ref,
  } = Route.useParams()
  const treePath = Route.useParams({ select: s => s['_splat'] }) ?? ''

  return (
    <Text size="sm" c="dimmed">
      {`${datasetId} / ${ref} / ${treePath}`}
    </Text>
  )
}
