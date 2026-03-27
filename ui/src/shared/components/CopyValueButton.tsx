import {
  ActionIcon,
  CopyButton,
  Tooltip,
} from '@mantine/core'
import { IconCopy } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import type { ReactNode } from 'react'

interface CopyRenderProps {
  copied: boolean
  copy: () => void
}

interface CopyValueButtonProps {
  value: string
  timeout?: number
  iconSize?: number
  color?: string
  children?: (props: CopyRenderProps) => ReactNode
}

export function CopyValueButton({
  value,
  color = 'gray',
  timeout = 1500,
  iconSize = 16,
  children,
}: CopyValueButtonProps) {
  const { t } = useTranslation()

  return (
    <CopyButton value={value} timeout={timeout}>
      {({
        copied,
        copy,
      }) => {
        return (
          <Tooltip
            withArrow
            label={copied ? t('shared.copyValueButton.copied') : t('shared.copyValueButton.copy')}
          >
            {
              children
                ? children({
                    copied,
                    copy,
                  })
                : (
                    <ActionIcon
                      size="sm"
                      variant="subtle"
                      color={color}
                      onClick={copy}
                    >
                      <IconCopy size={iconSize} />
                    </ActionIcon>
                  )
            }
          </Tooltip>
        )
      }}
    </CopyButton>
  )
}
