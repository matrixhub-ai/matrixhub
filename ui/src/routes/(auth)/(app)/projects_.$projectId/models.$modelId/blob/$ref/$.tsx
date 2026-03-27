import { Text } from '@mantine/core'
import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/models/$modelId/blob/$ref/$',
)({
  component: ModelBlobPage,
})

function ModelBlobPage() {
  const {
    modelId, ref,
  } = Route.useParams()
  const filePath = Route.useParams({ select: s => s['_splat'] }) ?? ''

  return (
    <Text size="sm" c="dimmed">
      {`${modelId} / ${ref} / ${filePath}`}
    </Text>
  )
}
