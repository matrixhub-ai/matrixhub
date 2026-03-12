import {
  ActionIcon,
  Badge,
  CopyButton,
  Group,
  Stack,
  Text,
  Tooltip,
} from '@mantine/core'
import { type Label } from '@matrixhub/api-ts/v1alpha1/model.pb.ts'
import { Link } from '@tanstack/react-router'
import { type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import CopyIcon from '@/assets/svgs/copy.svg?react'
import FileIcon from '@/assets/svgs/file.svg?react'

interface DetailHeaderProps {
  projectId: string
  name: string
  size?: string
  updatedAt?: string
  /** Label list to render as badges */
  labels?: Label[]
  /** Action buttons (upload, download, etc.) */
  actions?: ReactNode
}

export function DetailHeader({
  projectId,
  name,
  size,
  updatedAt,
  labels,
  actions,
}: DetailHeaderProps) {
  const { t } = useTranslation()
  const fullName = `${projectId}/${name}`

  return (
    <Stack gap={12}>
      {/* Row 1: Breadcrumb + Action buttons */}
      <Group justify="space-between" align="center">
        <Group gap="4" align="center">
          <Text
            component={Link}
            to={`/projects/${projectId}`}
            c="cyan.6"
            fw={500}
            size="lg"
            td="none"
            style={{ cursor: 'pointer' }}
          >
            {projectId}
          </Text>
          <Text c="dimmed" size="lg" w="1.5rem" ta="center" inline>/</Text>
          <Text size="lg">{name}</Text>
          <CopyButton value={fullName} timeout={2000}>
            {({
              copied,
              copy,
            }) => (
              <Tooltip label={copied ? t('common.copied') : t('common.copyName')} withArrow>
                <ActionIcon variant="subtle" color="gray" onClick={copy} size={24}>
                  <CopyIcon />
                </ActionIcon>
              </Tooltip>
            )}
          </CopyButton>
        </Group>
        {actions && <Group gap="sm">{actions}</Group>}
      </Group>

      {/* Row 2: Badges */}
      {labels && labels.length > 0 && (
        <Group gap="sm">
          {labels.map(label => (
            <Badge key={label.id} variant="light" color="gray" leftSection={<FileIcon />} size="lg" radius="xl" fw={600}>
              {label.name}
            </Badge>
          ))}
        </Group>
      )}

      {/* Row 3: Metadata */}
      <Group gap="xl">
        <Text size="sm" c="dimmed">
          {t('common.fromProject')}
          ：
          {projectId}
        </Text>
        <Text size="sm" c="dimmed">
          {t('common.modelSize')}
          ：
          {size ?? '-'}
        </Text>
        <Text size="sm" c="dimmed">
          {t('common.updatedAt')}
          ：
          {updatedAt ?? '-'}
        </Text>
      </Group>
    </Stack>
  )
}
