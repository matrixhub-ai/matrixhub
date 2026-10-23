import { Alert, Text } from '@mantine/core'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { ModelSecurityPage } from '@/features/models/security/pages/ModelSecurityPage'
import {
  securityAuditQuery, securityPolicyQuery, securityReportQuery,
} from '@/features/models/security/security.query'
import { securitySearchSchema } from '@/features/models/security/security.schema'

export const Route = createFileRoute('/(auth)/(app)/projects_/$projectId/models/$modelId/security/')({
  validateSearch: securitySearchSchema,
  loaderDeps: ({ search }) => search,
  loader: async ({
    context, params, deps,
  }) => {
    const results = await Promise.allSettled([
      context.queryClient.ensureQueryData(securityReportQuery(params.projectId, params.modelId, deps.revision)),
      context.queryClient.ensureQueryData(securityAuditQuery(params.projectId, params.modelId)),
      context.queryClient.ensureQueryData(securityPolicyQuery(params.projectId, params.modelId)),
    ])

    for (const result of results) {
      if (result.status === 'rejected') {
        throw result.reason
      }
    }
  },
  component: SecurityPage,
  errorComponent: SecurityError,
})

function SecurityPage() {
  const { revision } = Route.useSearch()

  return <ModelSecurityPage key={revision} />
}

function SecurityError({ error }: { error: Error }) {
  const { t } = useTranslation()

  return <Alert color="red" mt="lg" title={t('security.unavailable')}><Text size="sm">{error.message}</Text></Alert>
}
