import { Button } from '@mantine/core'
import {
  SyncTaskStatus,
  type SyncTask,
} from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { stopReplicationExecutionMutationOptions } from '../executions.mutation'

interface StopReplicationExecutionActionProps {
  syncPolicyId: number
  task: SyncTask
  disabled?: boolean
}

export function StopReplicationExecutionAction({
  syncPolicyId,
  task,
  disabled,
}: StopReplicationExecutionActionProps) {
  const { t } = useTranslation()
  const mutation = useMutation(stopReplicationExecutionMutationOptions(syncPolicyId))
  const canStop = task.status === SyncTaskStatus.SYNC_TASK_STATUS_RUNNING

  const handleStop = async () => {
    if (task.id == null) {
      return
    }

    await mutation.mutateAsync({ syncTaskId: task.id })
  }

  return (
    <Button
      variant="transparent"
      size="compact-sm"
      color="blue"
      disabled={disabled || !canStop || mutation.isPending}
      loading={mutation.isPending}
      onClick={() => {
        void handleStop()
      }}
    >
      {t('routes.admin.replications.executions.actions.stop')}
    </Button>
  )
}
