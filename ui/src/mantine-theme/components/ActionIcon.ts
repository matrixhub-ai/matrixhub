import { ActionIcon, rem } from '@mantine/core'

const actionIconSizeMap = {
  xs: rem(16),
  sm: rem(20),
  md: rem(24),
  lg: rem(32),
  xl: rem(40),
} as const

const isMappedActionIconSize = (size: unknown): size is keyof typeof actionIconSizeMap => (
  typeof size === 'string' && Object.prototype.hasOwnProperty.call(actionIconSizeMap, size)
)

export const actionIconTheme = ActionIcon.extend({
  defaultProps: {
    size: 'md',
  },
  vars: (_, props) => ({
    root: {
      '--ai-size': isMappedActionIconSize(props.size) ? actionIconSizeMap[props.size] : undefined,
    },
  }),
})
