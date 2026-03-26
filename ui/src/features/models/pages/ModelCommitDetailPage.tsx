import {
  Stack,
  Text,
} from '@mantine/core'
import { useTranslation } from 'react-i18next'

export function ModelCommitDetailPage() {
  const { t } = useTranslation()

  return (
    <Stack pt="sm" gap="md">
      <Text fw={600}>{t('model.commitDetail.title')}</Text>
    </Stack>
  )
}
