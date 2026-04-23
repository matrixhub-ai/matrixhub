import {
  Badge,
  type BadgeProps,
  Group,
  type PolymorphicFactory,
  polymorphicFactory,
  Text,
  useProps,
} from '@mantine/core'

import type { ReactNode } from 'react'

export interface BaseBadgeProps extends Omit<BadgeProps, 'children'> {
  icon?: ReactNode
  label?: ReactNode
  children?: ReactNode
}

export type BaseBadgeFactory = PolymorphicFactory<{
  props: BaseBadgeProps
  defaultComponent: 'div'
  defaultRef: HTMLDivElement
  stylesNames: 'root' | 'label'
}>

type BadgeStylesObject = Exclude<NonNullable<BadgeProps['styles']>, (...args: never[]) => unknown>

const DEFAULT_BADGE_STYLES: BadgeStylesObject = {
  root: {
    backgroundColor: 'var(--mantine-color-gray-1)',
    paddingInline: 12,
    border: '1px solid var(--mantine-color-gray-1)',
  },
  label: {
    paddingInline: 0,
    textTransform: 'none',
  },
}

const defaultProps: Partial<BaseBadgeProps> = {
  h: 24,
  radius: 'lg',
}

export const BaseBadge = polymorphicFactory<BaseBadgeFactory>((_props, ref) => {
  const {
    icon,
    label,
    children,
    styles,
    ...badgeProps
  } = useProps('BaseBadge', defaultProps, _props)

  const objectStyles: BadgeStylesObject | undefined = styles && typeof styles !== 'function'
    ? styles
    : undefined

  const resolvedStyles = typeof styles === 'function'
    ? styles
    : {
        ...objectStyles,
        root: {
          ...DEFAULT_BADGE_STYLES.root,
          ...objectStyles?.root,
        },
        label: {
          ...DEFAULT_BADGE_STYLES.label,
          ...objectStyles?.label,
        },
      }

  return (
    <Badge
      ref={ref}
      styles={resolvedStyles}
      {...badgeProps}
    >
      <Group gap={4} wrap="nowrap" miw={0}>
        {icon
          ? (
              <span style={{
                display: 'flex',
                flexShrink: 0,
              }}
              >
                {icon}
              </span>
            )
          : null}
        {children ?? (
          <Text
            component="span"
            size="12px"
            lh="22px"
            fw={600}
            c="gray.6"
            truncate="end"
            miw={0}
          >
            {label}
          </Text>
        )}
      </Group>
    </Badge>
  )
})

BaseBadge.displayName = 'BaseBadge'
