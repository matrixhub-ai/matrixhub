import { createFileRoute } from '@tanstack/react-router'

import { RouteStatusPage } from '@/shared/components/RouteStatusPage'

export const Route = createFileRoute('/$')({
  component: NotFoundPage,
})

function NotFoundPage() {
  return (
    <RouteStatusPage
      code={404}
      fullScreen
    />
  )
}
