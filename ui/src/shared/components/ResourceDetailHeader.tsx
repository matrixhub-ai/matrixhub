import {
  ActionIcon,
  Badge,
  CopyButton,
  Group,
  rem,
  Text,
  Tooltip,
} from '@mantine/core'
import { type Label } from '@matrixhub/api-ts/v1alpha1/model.pb.ts'
import {
  IconCopy as CopyIcon,
  IconFileDescription as FileIcon,
} from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

interface ResourceDetailHeaderProps {
  projectId: string
  name: string
  size?: string
  updatedAt?: string
  labels?: Label[]
  actions?: ReactNode
}

export function ResourceDetailHeader({
  projectId,
  name,
  size,
  updatedAt,
  labels,
  actions,
}: ResourceDetailHeaderProps) {
  const { t } = useTranslation()
  const fullName = `${projectId}/${name}`

  const metaItems = [
    {
      label: t('common.fromProject'),
      value: projectId,
    },
    {
      label: t('common.modelSize'),
      value: size ?? '-',
    },
    {
      label: t('common.updatedAt'),
      value: updatedAt ?? '-',
    },
  ]

  return (
    <>
      <Group justify="space-between" align="flex-start" mb={10}>
        <Group gap="4" align="center">
          <Text
            component={Link}
            to={`/projects/${projectId}`}
            c="cyan"
            fw={600}
            size="lg"
            lh="28px"
            td="none"
          >
            {projectId}
          </Text>
          <Text c="dimmed" size="lg" w={24} h={24} fw={600} lh="24px" ta="center" inline>/</Text>
          <Text size="lg" c="gray.9" lh="28px">{name}</Text>
          <CopyButton value={fullName} timeout={2000}>
            {({
              copied, copy,
            }) => (
              <Tooltip label={copied ? t('common.copied') : t('common.copyName')} withArrow>
                <ActionIcon variant="subtle" color="gray" onClick={copy} size={24}>
                  <CopyIcon size={16} />
                </ActionIcon>
              </Tooltip>
            )}
          </CopyButton>
        </Group>
        {actions && <Group gap="sm">{actions}</Group>}
      </Group>

      {labels?.length && (
        <Group gap={8} mb={12}>
          {labels.map(label => (
            <Badge
              key={label.id}
              variant="light"
              color="gray"
              leftSection={<FileIcon size={14} />}
              size="md"
              radius="xl"
              fw={600}
              h={24}
            >
              {label.name}
            </Badge>
          ))}
        </Group>
      )}

      <Group gap={24} fz="xs" lh={rem(20)} c="dimmed">
        {metaItems.map(item => (
          <span key={item.label}>
            {item.label}
            {t('common.colon')}
            {item.value}
          </span>
        ))}
      </Group>
    </>
  )
}
