import { Text } from '@mantine/core'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { ModalWrapper } from '@/shared/components/ModalWrapper.tsx'

import { disableRobotAccountMutationOptions } from '../robot.mutation.ts'

import type { RobotAccount } from '../robot.utils.ts'

interface DisableRobotModalProps {
  robot: RobotAccount
  opened: boolean
  onClose: () => void
}

export function DisableRobotModal({
  opened,
  robot,
  onClose,
}: DisableRobotModalProps) {
  const { t } = useTranslation()
  const mutation = useMutation(disableRobotAccountMutationOptions())

  const handleConfirm = async () => {
    await mutation.mutateAsync(robot)
    onClose()
  }

  return (
    <ModalWrapper
      title={t('routes.admin.robots.disableModal.title')}
      type="danger"
      size="sm"
      opened={opened}
      onClose={onClose}
      confirmLoading={mutation.isPending}
      onConfirm={handleConfirm}
    >
      <Text size="sm">
        {t('routes.admin.robots.disableModal.description', {
          name: robot.name,
        })}
      </Text>
    </ModalWrapper>
  )
}
