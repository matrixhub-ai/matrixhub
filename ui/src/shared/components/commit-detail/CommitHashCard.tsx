import {
  Box,
  Flex,
  Group,
  rem,
  Text,
} from '@mantine/core'
import { IconCopy } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import { CopyValueButton } from '@/shared/components/CopyValueButton'

interface CommitHashCardProps {
  commitId?: string
}

export function CommitHashCard({ commitId }: CommitHashCardProps) {
  const { t } = useTranslation()

  const shownCommitId = commitId ?? ''
  const shortCommitId = shownCommitId.slice(0, 7)

  if (!shortCommitId) {
    return null
  }

  return (
    <Group gap={8} wrap="nowrap">
      <Text fw={600} size="sm">
        {t('shared.commitDetail.commit')}
      </Text>
      <Box
        bdrs="sm"
        bg="#fff"
        style={{
          border: '1px solid var(--mantine-color-gray-3)',
        }}
      >
        <Flex align="center">
          <Text
            py={3}
            pl={12}
            pr={6}
            size="xs"
            c="gray.5"
            lh={rem(16)}
            style={{ borderRight: '1px solid var(--mantine-color-gray-3)' }}
          >
            {shortCommitId}
          </Text>
          <CopyValueButton value={shownCommitId}>
            {({ copy }) => {
              return (
                <Box onClick={copy} px={8} style={{ cursor: 'pointer' }} lh={1}>
                  <IconCopy size={12} style={{ color: 'var(--mantine-color-gray-6)' }} />
                </Box>
              )
            }}
          </CopyValueButton>
        </Flex>
      </Box>
    </Group>
  )
}
