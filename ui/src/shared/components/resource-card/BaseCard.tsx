import {
  Card,
  Group,
  Stack,
  Text,
} from '@mantine/core'
import { IconDots } from '@tabler/icons-react'
import { Fragment } from 'react'

import { BaseBadge } from '@/shared/components/badges/BaseBadge.tsx'

import classes from './BaseCard.module.css'

import type { ReactNode } from 'react'

export interface ResourceBadge {
  key: string | number
  content?: ReactNode
  icon?: ReactNode
  label?: ReactNode
}

export interface ResourceMetaItem {
  key: string
  label: string
  icon?: ReactNode
  value: ReactNode
}

interface BaseCardProps {
  title?: ReactNode
  badges?: ResourceBadge[]
  maxBadges?: number
  metaItems?: ResourceMetaItem[]
  renderMeta?: () => ReactNode
  renderRoot?: (props: Record<string, unknown>) => ReactNode
}

const EMPTY_BADGES: ResourceBadge[] = []
const EMPTY_META_ITEMS: ResourceMetaItem[] = []
const DEFAULT_MAX_BADGES = 3

export function BaseCard({
  title,
  badges = EMPTY_BADGES,
  maxBadges = DEFAULT_MAX_BADGES,
  metaItems = EMPTY_META_ITEMS,
  renderMeta,
  renderRoot,
}: BaseCardProps) {
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
              <Fragment key={badge.key}>
                {badge.content ?? (
                  <BaseBadge
                    icon={badge.icon}
                    label={badge.label}
                    maw={132}
                  />
                )}
              </Fragment>
            ))}

            {hasOverflow && (
              <BaseBadge
                maw={32}
                miw={32}
                styles={{
                  root: {
                    paddingInline: 8,
                  },
                }}
              >
                <IconDots
                  size={12}
                  color="var(--mantine-color-gray-6)"
                />
              </BaseBadge>
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
