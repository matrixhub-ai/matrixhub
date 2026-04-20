import { Button } from '@mantine/core'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { switchReplicationMutationOptions } from '../replications.mutation'

import type { SyncPolicyItem } from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'

interface ToggleReplicationActionProps {
  syncPolicy: SyncPolicyItem
  disabled?: boolean
}

export function ToggleReplicationAction({
  syncPolicy,
  disabled,
}: ToggleReplicationActionProps) {
  const { t } = useTranslation()
  const mutation = useMutation(switchReplicationMutationOptions())
  const nextIsDisabled = !syncPolicy.isDisabled

  const handleToggle = async () => {
    if (syncPolicy.id == null) {
      return
    }

    await mutation.mutateAsync({
      syncPolicyId: syncPolicy.id,
      isDisabled: nextIsDisabled,
    })
  }

  return (
    <Button
      variant="transparent"
      size="compact-sm"
      color="blue"
      disabled={disabled || mutation.isPending}
      loading={mutation.isPending}
      onClick={() => {
        void handleToggle()
      }}
    >
      {syncPolicy.isDisabled
        ? t('routes.admin.replications.actions.enable')
        : t('routes.admin.replications.actions.disable')}
    </Button>
  )
}
