import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/(app)/_layout/about')({
  component: AboutRoute,
  staticData: {
    navName: 'About',
  },
})

function AboutRoute() {
  return (
    <div>About MatrixHub</div>
  )
}
