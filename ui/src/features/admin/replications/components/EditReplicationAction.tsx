import { Button } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { useTranslation } from 'react-i18next'

import { ReplicationFormModal } from './ReplicationFormModal'

import type { SyncPolicyItem } from '@matrixhub/api-ts/v1alpha1/sync_policy.pb'

interface EditReplicationActionProps {
  syncPolicy: SyncPolicyItem
  disabled?: boolean
}

export function EditReplicationAction({
  syncPolicy,
  disabled,
}: EditReplicationActionProps) {
  const { t } = useTranslation()
  const [opened, {
    open, close,
  }] = useDisclosure(false)

  return (
    <>
      <Button
        variant="transparent"
        size="compact-sm"
        color="blue"
        disabled={disabled}
        onClick={open}
      >
        {t('routes.admin.replications.actions.edit')}
      </Button>

      {opened && (
        <ReplicationFormModal
          mode="edit"
          opened
          syncPolicy={syncPolicy}
          onClose={close}
        />
      )}
    </>
  )
}
