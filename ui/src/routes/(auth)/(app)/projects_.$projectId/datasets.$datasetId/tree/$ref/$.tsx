import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute(
  '/(auth)/(app)/projects_/$projectId/datasets/$datasetId/tree/$ref/$',
)({
  component: RouteComponent,
})

function RouteComponent() {
  const {
    projectId, datasetId, ref,
  } = Route.useParams()
  const treePath = Route.useParams({ select: s => s['_splat'] })

  return (
    <div>
      Dataset Tree Page
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
      Path:
      {' '}
      {treePath}
    </div>
  )
}
