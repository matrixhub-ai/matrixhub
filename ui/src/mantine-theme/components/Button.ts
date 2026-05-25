import { Button, rem } from '@mantine/core'

const buttonSizeMap = {
  xs: {
    height: rem(24),
    paddingX: rem(12),
  },
  sm: {
    height: rem(32),
    paddingX: rem(16),
  },
  md: {
    height: rem(40),
    paddingX: rem(20),
  },
  lg: {
    height: rem(48),
    paddingX: rem(24),
  },
  xl: {
    height: rem(56),
    paddingX: rem(32),
  },
} as const

const isMappedButtonSize = (size: unknown): size is keyof typeof buttonSizeMap => (
  typeof size === 'string' && Object.prototype.hasOwnProperty.call(buttonSizeMap, size)
)

export const buttonTheme = Button.extend({
  defaultProps: {
    size: 'sm',
  },
  vars: (_, props) => ({
    root: {
      '--button-bd': props.variant === 'outline' || props.variant === 'default' ? undefined : rem(0),
      '--button-height': isMappedButtonSize(props.size) ? buttonSizeMap[props.size].height : undefined,
      '--button-padding-x': isMappedButtonSize(props.size) ? buttonSizeMap[props.size].paddingX : undefined,
    },
  }),
})
