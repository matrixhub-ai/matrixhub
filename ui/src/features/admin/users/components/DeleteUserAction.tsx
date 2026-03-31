import {
  Button,
  Group,
  Stack,
  Text,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { ModalWrapper } from '@/shared/components/ModalWrapper'

import { deleteUserMutationOptions } from '../users.mutation'
import { getUserDisplayName } from '../users.utils'

import type { User } from '@matrixhub/api-ts/v1alpha1/user.pb'

interface DeleteUserActionProps {
  user: User
  disabled?: boolean
}

interface DeleteUserModalProps {
  opened: boolean
  user: User
  loading?: boolean
  onClose: () => void
  onConfirm: () => void
}

function DeleteUserModal({
  opened,
  user,
  loading,
  onClose,
  onConfirm,
}: DeleteUserModalProps) {
  const { t } = useTranslation()

  return (
    <ModalWrapper
      size="sm"
      opened={opened}
      onClose={onClose}
      type="danger"
      title={t('routes.admin.users.deleteModal.title')}
      footer={(
        <Group justify="flex-end" gap="md">
          <Button
            color="default"
            variant="subtle"
            onClick={onClose}
          >
            {t('common.cancel')}
          </Button>
          <Button
            loading={loading}
            onClick={onConfirm}
          >
            {t('routes.admin.users.deleteModal.submit')}
          </Button>
        </Group>
      )}
    >
      <Stack gap="sm">
        <Text size="sm">
          {t('routes.admin.users.deleteModal.description', {
            username: getUserDisplayName(user),
          })}
        </Text>
      </Stack>
    </ModalWrapper>
  )
}

export function DeleteUserAction({
  user,
  disabled,
}: DeleteUserActionProps) {
  const { t } = useTranslation()
  const [opened, {
    open,
    close,
  }] = useDisclosure(false)
  const mutation = useMutation(deleteUserMutationOptions())

  const handleDelete = async () => {
    await mutation.mutateAsync({ id: user.id })
    close()
  }

  return (
    <>
      <Button
        variant="transparent"
        size="compact-sm"
        color="blue"
        disabled={disabled}
        onClick={open}
      >
        {t('routes.admin.users.actions.delete')}
      </Button>

      {opened && (
        <DeleteUserModal
          opened
          user={user}
          loading={mutation.isPending}
          onClose={close}
          onConfirm={() => {
            void handleDelete()
          }}
        />
      )}
    </>
  )
}
