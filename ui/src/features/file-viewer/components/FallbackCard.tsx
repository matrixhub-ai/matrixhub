import {
  Box,
  Stack,
  Text,
} from '@mantine/core'
import { useTranslation } from 'react-i18next'

import { formatStorageSize } from '@/shared/utils/format'

import type { FileViewerFile } from '../types'

interface FallbackCardProps {
  file: FileViewerFile
}

export function FallbackCard({ file }: FallbackCardProps) {
  const { t } = useTranslation()

  return (
    <Box px="md" py="lg">
      <Stack gap="sm">
        {/* Notice */}
        <Text size="sm" c="dimmed">
          {t('file-viewer.binaryFileNotice')}
        </Text>

        {/* File info */}
        <Stack gap={4}>
          <Text size="sm" c="dimmed">
            {t('file-viewer.fileInfo')}
          </Text>

          {file.sha256
            ? (
                <Text size="sm" ff="monospace">
                  SHA256:
                  {' '}
                  {file.sha256}
                </Text>
              )
            : null}

          <Text size="sm">
            {t('file-viewer.fileSize')}
            {formatStorageSize(file.size)}
          </Text>
        </Stack>

        {/* TODO: Tensor metadata table for .safetensors files */}
      </Stack>
    </Box>
  )
}
