import { Text } from '@mantine/core'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { ModalWrapper } from '@/shared/components/ModalWrapper.tsx'

import { enableRobotAccountMutationOptions } from '../robot.mutation.ts'

import type { RobotAccount } from '../robot.utils.ts'

interface EnableRobotModalProps {
  robot: RobotAccount
  opened: boolean
  onClose: () => void
}

export function EnableRobotModal({
  opened,
  robot,
  onClose,
}: EnableRobotModalProps) {
  const { t } = useTranslation()
  const mutation = useMutation(enableRobotAccountMutationOptions())

  const handleConfirm = async () => {
    await mutation.mutateAsync(robot)
    onClose()
  }

  return (
    <ModalWrapper
      title={t('routes.admin.robots.enableModal.title')}
      type="info"
      size="sm"
      opened={opened}
      onClose={onClose}
      confirmLoading={mutation.isPending}
      onConfirm={handleConfirm}
    >
      <Text size="sm">
        {t('routes.admin.robots.enableModal.description', {
          name: robot.name,
        })}
      </Text>
    </ModalWrapper>
  )
}
