import { IconBinaryTree } from '@tabler/icons-react'

import { BaseBadge } from '@/shared/components/badges/BaseBadge'

import type { ComponentProps } from 'react'

type BaseBadgeProps = ComponentProps<typeof BaseBadge>

interface ParameterCountBadgeProps extends Omit<BaseBadgeProps, 'icon' | 'label'> {
  parameterCount: string
}

export function ParameterCountBadge({
  parameterCount,
  ...badgeProps
}: ParameterCountBadgeProps) {
  return (
    <BaseBadge
      icon={(
        <IconBinaryTree
          size={16}
          color="var(--mantine-color-violet-4)"
        />
      )}
      label={parameterCount}
      {...badgeProps}
    />
  )
}
