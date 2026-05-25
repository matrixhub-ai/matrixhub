import { Tabs, rem } from '@mantine/core'

export const tabsTheme = Tabs.extend({
  vars: () => ({
    root: {},
    list: {
      '--tabs-list-gap': rem(20),
    },
  }),
})
