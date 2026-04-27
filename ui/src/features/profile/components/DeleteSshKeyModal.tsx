import { Text } from '@mantine/core'
import { type SSHKey } from '@matrixhub/api-ts/v1alpha1/current_user.pb'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { deleteSshKeyMutationOptions } from '@/features/profile/profile.mutation'
import { ModalWrapper } from '@/shared/components/ModalWrapper'

interface DeleteSshKeyModalProps {
  sshKey: SSHKey | null
  opened: boolean
  onClose: () => void
}

export function DeleteSshKeyModal({
  sshKey,
  opened,
  onClose,
}: DeleteSshKeyModalProps) {
  const { t } = useTranslation()
  const mutation = useMutation(deleteSshKeyMutationOptions())

  const handleConfirm = () => {
    if (!sshKey?.id) {
      return
    }

    mutation.mutate(sshKey.id, {
      onSuccess: () => onClose(),
    })
  }

  return (
    <ModalWrapper
      type="danger"
      opened={opened}
      onClose={onClose}
      title={t('profile.sshKey.delete.title')}
      closeOnClickOutside={false}
      confirmLoading={mutation.isPending}
      onConfirm={handleConfirm}
    >
      <Text size="sm">
        {t('profile.sshKey.delete.confirm', {
          name: sshKey?.name ?? '',
        })}
      </Text>
    </ModalWrapper>
  )
}
