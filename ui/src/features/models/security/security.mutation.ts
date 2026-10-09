import { ModelSecurity, type ScanPolicy } from '@matrixhub/api-ts/security'
import { mutationOptions } from '@tanstack/react-query'

import i18n from '@/i18n'

import { securityKeys } from './security.query'

import type { NotificationMeta } from '@/types/tanstack-query'

export function securityTaskMutation(project: string, model: string, action: 'rescan' | 'cancel') {
  return mutationOptions({
    mutationFn: (revision: string) => ModelSecurity[action](project, model, revision),
    meta: {
      errorMessage: i18n.t('security.actionFailed'),
      invalidates: [securityKeys.model(project, model)],
    } satisfies NotificationMeta,
  })
}
export function securityPolicyMutation(project: string, model: string) {
  return mutationOptions({
    mutationFn: (policy: ScanPolicy) => ModelSecurity.setPolicy(project, model, policy),
    meta: {
      errorMessage: i18n.t('security.actionFailed'),
      successMessage: i18n.t('security.policySaved'),
      invalidates: [securityKeys.all],
    } satisfies NotificationMeta,
  })
}
