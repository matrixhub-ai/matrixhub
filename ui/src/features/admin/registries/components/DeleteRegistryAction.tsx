import {
  Button,
  Stack,
  Text,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { ModalWrapper } from '@/shared/components/ModalWrapper'

import { deleteRegistryMutationOptions } from '../registries.mutation'
import { getRegistryDisplayName } from '../registries.utils'

import type { Registry } from '@matrixhub/api-ts/v1alpha1/registry.pb'

interface DeleteRegistryActionProps {
  registry: Registry
  disabled?: boolean
}

interface DeleteRegistryModalProps {
  opened: boolean
  registry: Registry
  loading?: boolean
  onClose: () => void
  onConfirm: () => void
}

function DeleteRegistryModal({
  opened,
  registry,
  loading,
  onClose,
  onConfirm,
}: DeleteRegistryModalProps) {
  const { t } = useTranslation()

  return (
    <ModalWrapper
      opened={opened}
      onClose={onClose}
      type="danger"
      size="sm"
      closeOnClickOutside={false}
      title={t('routes.admin.registries.deleteModal.title')}
      confirmLoading={loading}
      onConfirm={onConfirm}
    >
      <Stack gap="sm">
        <Text size="sm">
          {t('routes.admin.registries.deleteModal.description', {
            name: getRegistryDisplayName(registry),
          })}
        </Text>
      </Stack>
    </ModalWrapper>
  )
}

export function DeleteRegistryAction({
  registry,
  disabled,
}: DeleteRegistryActionProps) {
  const { t } = useTranslation()
  const [opened, {
    open,
    close,
  }] = useDisclosure(false)
  const mutation = useMutation(deleteRegistryMutationOptions())

  const handleDelete = async () => {
    await mutation.mutateAsync({ id: registry.id })
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
        {t('routes.admin.registries.actions.delete')}
      </Button>

      {opened && (
        <DeleteRegistryModal
          opened
          registry={registry}
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
