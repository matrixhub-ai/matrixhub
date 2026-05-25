import { Alert, rem } from '@mantine/core'

export const alertTheme = Alert.extend({
  defaultProps: {
    px: 'sm',
    py: 'sm',
    bd: 'none',
  },
  styles: {
    icon: {
      marginInlineEnd: rem(8),
    },
  },
})
