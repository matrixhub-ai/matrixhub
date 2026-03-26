import {
  ActionIcon,
  CopyButton,
  Tooltip,
} from '@mantine/core'
import { IconCopy } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

interface CopyValueButtonProps {
  value: string
  timeout?: number
  iconSize?: number
  color?: string
}

export function CopyValueButton({
  value,
  color = 'gray',
  timeout = 1500,
  iconSize = 16,
}: CopyValueButtonProps) {
  const { t } = useTranslation()

  return (
    <CopyButton value={value} timeout={timeout}>
      {({
        copied,
        copy,
      }) => (
        <Tooltip
          withArrow
          label={copied ? t('shared.copyValueButton.copied') : t('shared.copyValueButton.copy')}
        >
          <ActionIcon
            size="sm"
            variant="subtle"
            color={color}
            onClick={copy}
          >
            <IconCopy size={iconSize} />
          </ActionIcon>
        </Tooltip>
      )}
    </CopyButton>
  )
}
