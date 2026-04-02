import {
  Button,
  Group,
  Stack,
  Text,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { ModalWrapper } from '@/shared/components/ModalWrapper'

import { batchDeleteUsersMutationOptions } from '../users.mutation'
import { getUserDisplayName } from '../users.utils'

import type { User } from '@matrixhub/api-ts/v1alpha1/user.pb'

interface BatchDeleteUsersModalProps {
  opened: boolean
  users: readonly User[]
  onClose: () => void
  onSuccess: () => void
}

export function BatchDeleteUsersModal({
  opened,
  users,
  onClose,
  onSuccess,
}: BatchDeleteUsersModalProps) {
  const { t } = useTranslation()
  const mutation = useMutation(batchDeleteUsersMutationOptions())
  const usernames = users.map(getUserDisplayName).join(', ')

  const handleDelete = async () => {
    const res = await mutation.mutateAsync(users)

    if (res && res.partial) {
      notifications.show({
        color: 'yellow',
        message: t('routes.admin.users.notifications.batchDeletePartialError', {
          successCount: res.successCount,
          failureCount: res.failureCount,
        }),
      })
    } else {
      notifications.show({
        color: 'green',
        message: t('routes.admin.users.notifications.batchDeleteSuccess'),
      })
    }
    onSuccess()
    onClose()
  }

  return (
    <ModalWrapper
      size="sm"
      opened={opened}
      onClose={onClose}
      type="danger"
      title={t('routes.admin.users.batchDeleteModal.title')}
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
            loading={mutation.isPending}
            onClick={handleDelete}
          >
            {t('routes.admin.users.batchDeleteModal.submit', {
              count: users.length,
            })}
          </Button>
        </Group>
      )}
    >
      <Stack gap="md">
        <Text size="sm">
          {t('routes.admin.users.batchDeleteModal.description', {
            usernames,
            count: users.length,
          })}
        </Text>
      </Stack>
    </ModalWrapper>
  )
}
