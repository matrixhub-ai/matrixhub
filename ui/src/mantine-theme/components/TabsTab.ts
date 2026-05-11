import { Tabs, rem } from '@mantine/core'

export const tabsTabTheme = Tabs.Tab.extend({
  defaultProps: {
    lh: rem(20),
    fw: 600,
    px: 12,
    pt: 8,
    pb: 6,
  },
})
