import {
  type PolymorphicFactory,
  polymorphicFactory,
  useProps,
} from '@mantine/core'
import { IconBinaryTree } from '@tabler/icons-react'

import { BaseBadge, type BaseBadgeProps } from '@/shared/components/badges/BaseBadge'

export interface ParameterCountBadgeProps extends Omit<BaseBadgeProps, 'icon' | 'label'> {
  parameterCount: string
}

export type ParameterCountBadgeFactory = PolymorphicFactory<{
  props: ParameterCountBadgeProps
  defaultComponent: 'div'
  defaultRef: HTMLDivElement
  stylesNames: 'root' | 'label'
}>

export const ParameterCountBadge = polymorphicFactory<ParameterCountBadgeFactory>((_props, ref) => {
  const {
    parameterCount,
    ...badgeProps
  } = useProps('ParameterCountBadge', {}, _props)

  return (
    <BaseBadge
      ref={ref}
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
})

ParameterCountBadge.displayName = 'ParameterCountBadge'
