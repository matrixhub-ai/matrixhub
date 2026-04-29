import {
  Breadcrumbs,
  Group,
  Text,
  Title,
  rem,
} from '@mantine/core'

import AnchorLink from '@/shared/components/AnchorLink'

import type { TablerIcon } from '@tabler/icons-react'
import type { LinkComponentProps } from '@tanstack/react-router'
import type { ReactNode } from 'react'

export interface AdminPageHeaderItem {
  label: ReactNode
  linkProps?: LinkComponentProps
}

interface AdminPageHeaderProps {
  actions?: ReactNode
  icon: TablerIcon
  items: [AdminPageHeaderItem, ...AdminPageHeaderItem[]]
}

const headerItemStyles = [
  {
    fw: 600,
    fz: rem(18),
    c: 'inherit',
  },
  {
    fw: 600,
    fz: rem(14),
    c: 'inherit',
  },
  {
    fw: 400,
    fz: rem(14),
    c: 'inherit',
  },
] as const

function HeaderItem({
  item,
  index,
}: {
  item: AdminPageHeaderItem
  index: number
}) {
  const style = headerItemStyles[index] ?? headerItemStyles[headerItemStyles.length - 1]
  const isPrimary = index === 0

  if (item.linkProps) {
    return (
      <AnchorLink
        c={style.c}
        underline="hover"
        fw={style.fw}
        fz={style.fz}
        {...item.linkProps}
      >
        {item.label}
      </AnchorLink>
    )
  }

  if (isPrimary) {
    return (
      <Title
        size={style.fz}
        fw={style.fw}
        lh={rem(28)}
        c={style.c}
      >
        {item.label}
      </Title>
    )
  }

  return (
    <Text
      c={style.c}
      fw={style.fw}
      fz={style.fz}
    >
      {item.label}
    </Text>
  )
}

export function AdminPageHeader({
  actions,
  icon: Icon,
  items,
}: AdminPageHeaderProps) {
  return (
    <Group
      px={28}
      py="lg"
      justify="space-between"
      wrap="nowrap"
      mih={72}
      align="center"
    >
      <Group
        gap={10}
        wrap="nowrap"
        align="center"
      >
        <Icon
          size={rem(28)}
          style={{ flexShrink: 0 }}
        />
        <Breadcrumbs
          separator="/"
          separatorMargin="xs"
        >
          {items.map((item, index) => (
            <HeaderItem
              key={item.label?.toString() ?? index}
              item={item}
              index={index}
            />
          ))}
        </Breadcrumbs>
      </Group>

      {actions && (
        <Group
          gap="sm"
          wrap="nowrap"
          align="center"
        >
          {actions}
        </Group>
      )}
    </Group>
  )
}
