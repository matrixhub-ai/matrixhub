import {
  Group,
  Text,
  Tooltip,
} from '@mantine/core'

import IconQuestion from '@/assets/svgs/question.svg?react'

import type { TooltipProps } from '@mantine/core'
import type { ReactNode } from 'react'

interface FieldHintLabelProps {
  label: ReactNode
  hint: ReactNode
  tooltipProps?: Omit<TooltipProps, 'children' | 'label'>
}

const DEFAULT_TOOLTIP_WIDTH = 240

export function FieldHintLabel({
  label,
  hint,
  tooltipProps,
}: FieldHintLabelProps) {
  return (
    <Group
      component="span"
      gap={4}
      align="center"
      wrap="nowrap"
      style={{
        display: 'inline-flex',
        alignItems: 'center',
      }}
    >
      <Text component="span" inherit>
        {label}
      </Text>
      <Tooltip
        label={hint}
        multiline
        withArrow
        w={DEFAULT_TOOLTIP_WIDTH}
        {...tooltipProps}
      >
        <IconQuestion
          width={18}
          height={18}
          style={{
            cursor: 'help',
            flex: 'none',
          }}
        />
      </Tooltip>
    </Group>
  )
}
