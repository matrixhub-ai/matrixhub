import {
  Badge,
  Group,
  Radio,
  Stack,
  Text,
} from '@mantine/core'
import { useTranslation } from 'react-i18next'

import classes from './DownloadMethodCards.module.css'

import type { DownloadMethod } from '../server-pull.utils'

interface DownloadMethodCardsProps {
  value: DownloadMethod
  onChange: (method: DownloadMethod) => void
  onBlur: () => void
}

/** "Download method": pull on the MatrixHub server (recommended) or download with the hf CLI. */
export function DownloadMethodCards({
  value,
  onChange,
  onBlur,
}: DownloadMethodCardsProps) {
  const { t } = useTranslation()

  return (
    <Radio.Group
      label={t('model.serverPull.method.label')}
      value={value}
      onChange={next => onChange(next === 'cli' ? 'cli' : 'server')}
      onBlur={onBlur}
    >
      <Stack gap="sm" mt="xs">
        <MethodCard
          value="server"
          title={t('model.serverPull.method.server')}
          badge={t('model.serverPull.method.recommended')}
          description={t('model.serverPull.method.serverDescription')}
        />
        <MethodCard
          value="cli"
          title={t('model.serverPull.method.cli')}
          description={t('model.serverPull.method.cliDescription')}
        />
      </Stack>
    </Radio.Group>
  )
}

function MethodCard({
  value,
  title,
  badge,
  description,
}: {
  value: DownloadMethod
  title: string
  badge?: string
  description: string
}) {
  return (
    <Radio.Card value={value} radius="md" className={classes.card}>
      <Group wrap="nowrap" align="flex-start" gap="sm">
        <Radio.Indicator mt={2} />
        <Stack gap={4}>
          <Group gap="xs">
            <Text size="sm" fw={600}>{title}</Text>
            {badge && <Badge size="sm" radius="sm" variant="light">{badge}</Badge>}
          </Group>
          <Text size="xs" c="dimmed">{description}</Text>
        </Stack>
      </Group>
    </Radio.Card>
  )
}
