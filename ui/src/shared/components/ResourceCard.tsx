import {
  Badge,
  Card,
  Group,
  Stack,
  Text,
} from '@mantine/core'
import { IconDots as DotsIcon } from '@tabler/icons-react'

import classes from './ResourceCard.module.css'

import type { ReactNode } from 'react'

export interface ResourceCardBadge {
  key: string | number
  icon?: ReactNode
  label: string
}

export interface ResourceCardMetaItem {
  key: string
  icon?: ReactNode
  value: ReactNode
}

interface ResourceCardProps {
  title?: ReactNode
  badges?: ResourceCardBadge[]
  maxBadges?: number
  metaItems?: ResourceCardMetaItem[]
  renderMeta?: () => ReactNode
  renderRoot?: (props: Record<string, unknown>) => ReactNode
}

const EMPTY_BADGES: ResourceCardBadge[] = []
const EMPTY_META_ITEMS: ResourceCardMetaItem[] = []
const DEFAULT_MAX_BADGES = 3

export function ResourceCard({
  title,
  badges = EMPTY_BADGES,
  maxBadges = DEFAULT_MAX_BADGES,
  metaItems = EMPTY_META_ITEMS,
  renderMeta,
  renderRoot,
}: ResourceCardProps) {
  const isInteractive = Boolean(renderRoot)
  const hasOverflow = badges.length > maxBadges
  const visibleBadges = hasOverflow ? badges.slice(0, maxBadges) : badges

  return (
    <Card
      withBorder
      radius="md"
      px="md"
      py="sm"
      h={116}
      className={classes.card}
      data-interactive={isInteractive || undefined}
      renderRoot={renderRoot}
    >
      <Stack gap={12} h="100%">
        {title && (
          <Text
            className={classes.title}
            fw={600}
            size="16px"
            lh="24px"
            truncate="end"
          >
            {title}
          </Text>
        )}

        {visibleBadges.length > 0 && (
          <Group gap={8} wrap="nowrap" className={classes.badgeRow}>
            {visibleBadges.map(badge => (
              <Badge
                key={badge.key}
                h={24}
                radius={16}
                maw={132}
                styles={{
                  root: {
                    backgroundColor: 'var(--mantine-color-gray-1)',
                    paddingInline: 12,
                  },
                  label: {
                    paddingInline: 0,
                    textTransform: 'none',
                  },
                }}
              >
                <Group gap={4} wrap="nowrap" miw={0}>
                  {badge.icon && (
                    <span className={classes.iconSlot}>{badge.icon}</span>
                  )}
                  <Text
                    component="span"
                    size="12px"
                    lh="20px"
                    fw={600}
                    c="gray.6"
                    truncate="end"
                    miw={0}
                  >
                    {badge.label}
                  </Text>
                </Group>
              </Badge>
            ))}

            {hasOverflow && (
              <Badge
                h={24}
                radius={16}
                maw={32}
                miw={32}
                styles={{
                  root: {
                    backgroundColor: 'var(--mantine-color-gray-1)',
                    paddingInline: 8,
                  },
                  label: {
                    paddingInline: 0,
                    textTransform: 'none',
                  },
                }}
              >
                <DotsIcon
                  width={12}
                  height={3}
                  style={{ color: 'var(--mantine-color-gray-6)' }}
                />
              </Badge>
            )}
          </Group>
        )}

        {renderMeta
          ? renderMeta()
          : metaItems.length > 0 && (
            <Group gap={32} mt="auto" wrap="nowrap" w="100%">
              {metaItems.map(item => (
                <Group key={item.key} gap={8} wrap="nowrap" flex="1 0 0" miw={0} c="dimmed">
                  {item.icon && (
                    <span className={classes.iconSlot}>{item.icon}</span>
                  )}
                  <Text
                    size="14px"
                    lh="16px"
                    fw={600}
                    tt="uppercase"
                    truncate="end"
                  >
                    {item.value}
                  </Text>
                </Group>
              ))}
            </Group>
          )}
      </Stack>
    </Card>
  )
}
