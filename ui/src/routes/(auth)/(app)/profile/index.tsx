import { createFileRoute, redirect } from '@tanstack/react-router'

import { Route as SecurityRoute } from '@/routes/(auth)/(app)/profile/security'

export const Route = createFileRoute('/(auth)/(app)/profile/')({
  beforeLoad: () => {
    throw redirect({
      to: SecurityRoute.to,
    })
  },
})
