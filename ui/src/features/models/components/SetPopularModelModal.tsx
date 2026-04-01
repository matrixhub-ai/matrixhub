import { rem, Text } from '@mantine/core'
import { useMutation } from '@tanstack/react-query'
import { useRouter } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { updateModelSettingMutationOptions } from '@/features/models/models.mutation'
import { ModalWrapper } from '@/shared/components/ModalWrapper'

interface RecommendedModelModalProps {
  opened: boolean
  onClose: () => void
  projectId: string
  modelId: string
  isPopular: boolean
}

export function SetPopularModelModal({
  opened,
  onClose,
  projectId,
  modelId,
  isPopular,
}: RecommendedModelModalProps) {
  const { t } = useTranslation()
  const router = useRouter()
  const updateMutation = useMutation(updateModelSettingMutationOptions(!isPopular))
  const fullName = `${projectId}/${modelId}`
  const title = isPopular
    ? t('model.settings.recommended.unset.label')
    : t('model.settings.recommended.set.label')
  const message = isPopular
    ? t('model.settings.recommended.unset.confirmation', { name: fullName })
    : t('model.settings.recommended.set.confirmation', { name: fullName })

  const handleConfirm = async () => {
    await updateMutation.mutateAsync({
      project: projectId,
      name: modelId,
      popular: !isPopular,
    })

    onClose()
    await router.invalidate()
  }

  return (
    <ModalWrapper
      type="info"
      size="sm"
      title={title}
      opened={opened}
      onClose={onClose}
      confirmLoading={updateMutation.isPending}
      onConfirm={handleConfirm}
    >
      <Text fz="sm" lh={rem(20)}>
        {message}
      </Text>
    </ModalWrapper>
  )
}
