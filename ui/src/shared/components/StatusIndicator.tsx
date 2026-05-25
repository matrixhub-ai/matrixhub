import {
  Box,
  Group,
  Text,
  type TextProps,
} from '@mantine/core'

import type { ReactNode } from 'react'

export type StatusIndicatorType = 'danger' | 'info' | 'neutral' | 'success' | 'warning'

export interface StatusIndicatorProps {
  label: ReactNode
  type?: StatusIndicatorType
  dotSize?: number
  textProps?: TextProps
}

const statusIndicatorColors: Record<StatusIndicatorType, string> = {
  danger: 'var(--mantine-color-error)',
  info: 'var(--mantine-primary-color-filled)',
  neutral: 'var(--mantine-color-dimmed)',
  success: 'var(--mantine-color-green-filled)',
  warning: 'var(--mantine-color-yellow-filled)',
}

export function StatusIndicator({
  label,
  type = 'neutral',
  dotSize = 10,
  textProps,
}: StatusIndicatorProps) {
  return (
    <Group gap={6} wrap="nowrap">
      <Box
        h={dotSize}
        w={dotSize}
        bg={statusIndicatorColors[type]}
        style={{
          borderRadius: '50%',
          flexShrink: 0,
        }}
      />
      <Text size="sm" {...textProps}>
        {label}
      </Text>
    </Group>
  )
}
