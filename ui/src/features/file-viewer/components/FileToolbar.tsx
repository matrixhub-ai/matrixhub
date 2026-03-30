import {
  Anchor,
  Group,
  SegmentedControl,
  Text,
} from '@mantine/core'
import { IconClock } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import { formatDateTime } from '@/shared/utils/date'
import { formatStorageSize } from '@/shared/utils/format'

import type { FileViewerFile } from '../types'

type ViewMode = 'preview' | 'code'

interface FileToolbarProps {
  file: FileViewerFile
  /** Whether this file is markdown — controls Preview/Code toggle visibility */
  isMarkdown?: boolean
  /** Current view mode (only relevant when isMarkdown is true) */
  viewMode?: ViewMode
  /** Callback when the user switches between Preview and Code */
  onViewModeChange?: (mode: ViewMode) => void
}

export function FileToolbar({
  file,
  isMarkdown = false,
  viewMode = 'code',
  onViewModeChange,
}: FileToolbarProps) {
  const { t } = useTranslation()

  return (
    <Group
      justify="space-between"
      px="sm"
      py="xs"
      bg="var(--mantine-color-default-hover)"
    >
      {/* Left side: commit info */}
      <Group
        gap="sm"
        style={{
          flex: 1,
          minWidth: 0,
        }}
      >
        {file.commit?.authorName
          ? (
              <Text size="sm" c="dimmed" truncate>
                {file.commit.authorName}
                {' '}
                {file.commit.message}
              </Text>
            )
          : null}

        {file.commit?.id
          ? (
              <Text
                size="xs"
                ff="monospace"
                px={6}
                py={2}
                bg="var(--mantine-color-gray-light)"
                style={{
                  borderRadius: 'var(--mantine-radius-sm)',
                  flexShrink: 0,
                }}
              >
                {file.commit.id.slice(0, 7).toUpperCase()}
              </Text>
            )
          : null}
      </Group>

      {/* Right side: toggle + meta + download */}
      <Group gap="md" wrap="nowrap">
        {isMarkdown
          ? (
              <SegmentedControl
                size="xs"
                color="cyan"
                value={viewMode}
                onChange={v => onViewModeChange?.(v as ViewMode)}
                data={[
                  {
                    label: t('file-viewer.preview'),
                    value: 'preview',
                  },
                  {
                    label: t('file-viewer.code'),
                    value: 'code',
                  },
                ]}
              />
            )
          : null}

        {file.commit?.committerDate
          ? (
              <Group gap={4} wrap="nowrap">
                <IconClock size={14} color="var(--mantine-color-dimmed)" />
                <Text size="xs" truncate>
                  {t('common.updatedAt')}
                  {' : '}
                  {formatDateTime(file.commit.committerDate)}
                </Text>
              </Group>
            )
          : null}

        {file.url
          ? (
              <Anchor
                href={file.url}
                download
                underline="never"
                size="xs"
              >
                <Text size="xs">{t('file-viewer.download')}</Text>
              </Anchor>
            )
          : null}

        <Text size="xs">
          {formatStorageSize(file.size)}
        </Text>
      </Group>
    </Group>
  )
}
