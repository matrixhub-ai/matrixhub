import { queryOptions } from '@tanstack/react-query'

import { ModelSecurity } from '@/features/models/security/security.api'

export const securityKeys = {
  all: ['model-security'] as const,
  model: (project: string, model: string) => [...securityKeys.all, project, model] as const,
}
export function securityReportQuery(project: string, model: string, revision: string) {
  return queryOptions({
    queryKey: [...securityKeys.model(project, model), 'report', revision],
    queryFn: () => ModelSecurity.report(project, model, revision),
    refetchInterval: query => ['unscanned', 'pending', 'scanning'].includes(query.state.data?.status ?? '') ? 2000 : false,
  })
}
export function securityAuditQuery(project: string, model: string) {
  return queryOptions({
    queryKey: [...securityKeys.model(project, model), 'audit'],
    queryFn: () => ModelSecurity.audit(project, model),
    refetchInterval: 5000,
  })
}
export function securityPolicyQuery(project: string, model: string) {
  return queryOptions({
    queryKey: [...securityKeys.model(project, model), 'policy'],
    queryFn: () => ModelSecurity.policy(project, model),
  })
}
