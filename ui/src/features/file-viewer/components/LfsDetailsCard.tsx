import {
  Anchor,
  Box,
  Group,
  Stack,
  Text,
} from '@mantine/core'
import { Trans, useTranslation } from 'react-i18next'

import { formatStorageSize } from '@/shared/utils/format'

import type { FileViewerFile } from '../types'

const XET_DOC_URL = 'https://huggingface.co/docs/hub/xet/index'
const LFS_DOC_URL = 'https://git-lfs.com/'

interface LfsDetailsCardProps {
  file: FileViewerFile
  /** Whether the file content is rendered below this card */
  previewable: boolean
}

interface DetailRowProps {
  label: string
  value: string
  /** Hashes: monospace + break anywhere so 64-hex values wrap in narrow layouts */
  hash?: boolean
}

function DetailRow({
  label,
  value,
  hash = false,
}: DetailRowProps) {
  return (
    <Group gap="xs" wrap="nowrap" align="flex-start">
      <Text size="sm" c="dimmed" style={{ flexShrink: 0 }}>
        {label}
      </Text>
      <Text
        size="sm"
        ff={hash ? 'monospace' : undefined}
        style={hash ? { wordBreak: 'break-all' } : undefined}
      >
        {value}
      </Text>
    </Group>
  )
}

export function LfsDetailsCard({
  file,
  previewable,
}: LfsDetailsCardProps) {
  const { t } = useTranslation()
  const isXet = !!file.xetHash
  // The gateway emits "0" for an unset int64 field
  const hasPointerSize = !!file.pointerSize && file.pointerSize !== '0'

  return (
    <Box px="md" py="lg">
      <Stack gap="sm">
        <Text size="sm">
          {isXet ? t('file-viewer.storedWithXet') : t('file-viewer.storedWithLfs')}
          {previewable
            ? null
            : (
                <>
                  {' '}
                  <Trans
                    t={t}
                    i18nKey="file-viewer.tooBigToDisplay"
                    components={{
                      download: file.url
                        ? <Anchor href={file.url} download inherit />
                        : <span />,
                    }}
                  />
                </>
              )}
        </Text>

        <Stack gap={4}>
          <Text fw={600} size="sm">
            {isXet ? t('file-viewer.xetPointerDetails') : t('file-viewer.lfsDetails')}
          </Text>

          {file.xetHash
            ? <DetailRow label={t('file-viewer.xetHash')} value={file.xetHash} hash />
            : null}

          {file.sha256
            ? <DetailRow label={t('file-viewer.sha256')} value={file.sha256} hash />
            : null}

          {hasPointerSize
            ? <DetailRow label={t('file-viewer.pointerSize')} value={formatStorageSize(file.pointerSize)} />
            : null}

          <DetailRow label={t('file-viewer.remoteFileSize')} value={formatStorageSize(file.size)} />
        </Stack>

        <Text size="sm" c="dimmed">
          {isXet ? t('file-viewer.xetExplanation') : t('file-viewer.lfsExplanation')}
          {' '}
          <Anchor
            href={isXet ? XET_DOC_URL : LFS_DOC_URL}
            target="_blank"
            rel="noreferrer"
            inherit
          >
            {t('file-viewer.moreInfo')}
          </Anchor>
        </Text>
      </Stack>
    </Box>
  )
}
