import { Text } from '@mantine/core'
import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/datasets/$datasetId/blob/$ref/$',
)({
  component: DatasetBlobPage,
})

function DatasetBlobPage() {
  const {
    datasetId, ref,
  } = Route.useParams()
  const filePath = Route.useParams({ select: s => s['_splat'] }) ?? ''

  return (
    <Text size="sm" c="dimmed">
      {`${datasetId} / ${ref} / ${filePath}`}
    </Text>
  )
}
