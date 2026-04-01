import {
  Alert, Button, Checkbox, rem, Stack, Text, TextInput,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { IconAlertTriangle, IconTrash } from '@tabler/icons-react'
import { useMutation, useSuspenseQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SetPopularModelModal } from '@/features/models/components/SetPopularModelModal.tsx'
import { deleteModelMutationOptions } from '@/features/models/models.mutation'
import { modelQueryOptions } from '@/features/models/models.query.ts'

interface ModelSettingsPageProps {
  projectId: string
  modelId: string
}

export function ModelSettingsPage({
  projectId, modelId,
}: ModelSettingsPageProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const [inputValue, setInputValue] = useState('')
  const { data: model } = useSuspenseQuery(modelQueryOptions(projectId, modelId))

  const [popularModalOpened, popularModalHandlers] = useDisclosure(false)
  const deleteMutation = useMutation(deleteModelMutationOptions())

  const fullName = `${projectId}/${modelId}`
  const isPopular = Boolean(model.popular)

  const handleDelete = async () => {
    await deleteMutation.mutateAsync({
      project: projectId,
      name: modelId,
    })

    // Fixme: confirm navigation to which page
    void navigate({
      to: '/projects/$projectId/models',
      params: {
        projectId,
      },
    })
  }

  return (
    <Stack
      pt={20}
      gap={28}
      align="flex-start"
    >
      <Stack
        gap="md"
        align="flex-start"
      >
        <Text fw="600" lh={rem(24)}>
          {t('model.settings.recommended.title')}
        </Text>

        <Checkbox
          checked={model.popular}
          label={t('model.settings.recommended.set.label')}
          onClick={popularModalHandlers.open}
        />

        <SetPopularModelModal
          opened={popularModalOpened}
          projectId={projectId}
          modelId={modelId}
          isPopular={isPopular}
          onClose={popularModalHandlers.close}
        />
      </Stack>

      <Stack
        gap="md"
        align="flex-start"
      >
        <Text fw="600" lh={rem(24)}>
          {t('model.settings.delete.title')}
        </Text>

        <Alert
          icon={<IconAlertTriangle size={20} />}
          variant="light"
          color="var(--mantine-color-yellow-6)"
        >
          <Text size="sm" lh={rem(20)} c="var(--mantine-color-gray-9)">
            {t('model.settings.delete.warning', { name: fullName })}
          </Text>
        </Alert>

        <Stack gap={8}>
          <Text fw={600} size="sm" lh="20px" c="var(--mantine-color-gray-7)">
            {t('model.settings.delete.confirmation', { name: fullName })}
          </Text>

          <TextInput
            w={260}
            value={inputValue}
            onChange={e => setInputValue(e.currentTarget.value)}
          />
        </Stack>

        <Button
          disabled={inputValue !== fullName || deleteMutation.isPending}
          loading={deleteMutation.isPending}
          leftSection={<IconTrash size={16} />}
          color="var(--mantine-color-red-6)"
          onClick={handleDelete}
        >
          {t('model.settings.delete.action')}
        </Button>
      </Stack>
    </Stack>
  )
}
