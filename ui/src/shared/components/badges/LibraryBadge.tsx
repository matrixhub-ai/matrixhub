import {
  type PolymorphicFactory,
  polymorphicFactory,
  useProps,
} from '@mantine/core'

import PytorchIcon from '@/assets/svgs/pytorch.svg?react'
import { BaseBadge, type BaseBadgeProps } from '@/shared/components/badges/BaseBadge'

import type { ReactNode } from 'react'

export interface LibraryBadgeProps extends Omit<BaseBadgeProps, 'icon' | 'label'> {
  library: string
}

export type LibraryBadgeFactory = PolymorphicFactory<{
  props: LibraryBadgeProps
  defaultComponent: 'div'
  defaultRef: HTMLDivElement
  stylesNames: 'root' | 'label'
}>

const LIBRARY_ICON_ENTRIES = [
  {
    matcher: /pytorch/i,
    icon: <PytorchIcon width={16} height={16} style={{ color: 'gray.6' }} />,
  },
] as const

function resolveLibraryIcon(library: string): ReactNode {
  const matched = LIBRARY_ICON_ENTRIES.find(item => item.matcher.test(library))

  return matched?.icon ?? null
}

export const LibraryBadge = polymorphicFactory<LibraryBadgeFactory>((_props, ref) => {
  const {
    library,
    ...badgeProps
  } = useProps('LibraryBadge', {}, _props)

  return (
    <BaseBadge
      ref={ref}
      icon={resolveLibraryIcon(library)}
      label={library}
      {...badgeProps}
    />
  )
})

LibraryBadge.displayName = 'LibraryBadge'
