import i18n from '@/i18n'

export function parseExecutionId(executionId: string) {
  const value = Number(executionId)

  if (!Number.isInteger(value) || value < 1) {
    throw new Error(i18n.t('routes.admin.replications.executions.detail.errors.invalidSyncTaskId'))
  }

  return value
}
