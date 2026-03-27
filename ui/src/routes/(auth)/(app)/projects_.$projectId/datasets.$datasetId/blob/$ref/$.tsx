import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/datasets/$datasetId/blob/$ref/$',
)({
  component: RouteComponent,
})

function RouteComponent() {
  const {
    projectId, datasetId, ref,
  } = Route.useParams()
  const filePath = Route.useParams({ select: s => s['_splat'] })

  return (
    <div>
      Dataset Blob Page
      <br />
      Project ID:
      {' '}
      {projectId}
      <br />
      Dataset ID:
      {' '}
      {datasetId}
      <br />
      Ref:
      {' '}
      {ref}
      <br />
      File Path:
      {' '}
      {filePath}
    </div>
  )
}
