import {
  Center,
  Stack,
  Text,
} from '@mantine/core'
import { useTranslation } from 'react-i18next'

import type { ReactNode } from 'react'

export interface EmptyStatePromptProps {
  title?: ReactNode
  helper?: ReactNode
}

export function EmptyStatePrompt({
  title,
  helper,
}: EmptyStatePromptProps) {
  const { t } = useTranslation()

  return (
    <Center py="xl">
      <Stack align="center" gap="xs">
        <Text fw={500} size="sm">{title ?? t('common.noResults')}</Text>
        {helper && (
          <Text size="sm" c="dimmed">
            {helper}
          </Text>
        )}
      </Stack>
    </Center>
  )
}
