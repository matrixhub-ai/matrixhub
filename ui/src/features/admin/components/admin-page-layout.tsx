import {
  Box,
  Group,
  Title,
  rem,
} from '@mantine/core'

import type { TablerIcon } from '@tabler/icons-react'
import type { ReactNode } from 'react'

interface AdminPageLayoutProps {
  children?: ReactNode
  icon: TablerIcon
  title: string
}

export function AdminPageLayout({
  children,
  icon: Icon,
  title,
}: AdminPageLayoutProps) {
  return (
    <Box component="section">
      {/* FIXME:Text/title (Gray/90) */}
      <Group
        px={28}
        py="lg"
        gap={10}
        wrap="nowrap"
        mih={72}
        align="center"
      >
        <Icon
          size={rem(28)}
          style={{ flexShrink: 0 }}
        />
        <Title
          size={rem(18)}
          fw={600}
          lh={rem(28)}
        >
          {title}
        </Title>
      </Group>
      <Box px="xl">
        {children}
      </Box>
    </Box>
  )
}
