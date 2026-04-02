import {
  Group,
  rem,
  Text,
} from '@mantine/core'
import { Link } from '@tanstack/react-router'
import { Fragment, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { BaseBadge } from '@/shared/components/badges/BaseBadge'
import { CopyValueButton } from '@/shared/components/CopyValueButton.tsx'

import type { ResourceBadge, ResourceMetaItem } from '@/shared/components/resource-card/BaseCard'

interface ResourceDetailHeaderProps {
  projectId: string
  name: string
  actions?: ReactNode
  badges?: ResourceBadge[]
  metaItems?: ResourceMetaItem[]
}

export function ResourceDetailHeader({
  projectId,
  name,
  badges,
  actions,
  metaItems,
}: ResourceDetailHeaderProps) {
  const { t } = useTranslation()
  const fullName = `${projectId}/${name}`

  return (
    <>
      <Group justify="space-between" align="flex-start" mb={10}>
        <Group gap="4" align="center">
          <Text
            component={Link}
            to={`/projects/${projectId}`}
            c="cyan"
            fw={600}
            size="lg"
            lh="28px"
            td="none"
          >
            {projectId}
          </Text>
          <Text c="dimmed" size="lg" w={24} h={24} fw={600} lh="24px" ta="center" inline>/</Text>
          <Text size="lg" c="gray.9" lh="28px">{name}</Text>
          <CopyValueButton value={fullName} iconSize={15} />
        </Group>
        {actions && <Group gap="sm">{actions}</Group>}
      </Group>

      {badges && badges?.length > 0 && (
        <Group gap={8} mb="sm" wrap="wrap">
          {badges.map(badge => (
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
        </Group>
      )}

      {metaItems && metaItems?.length > 0 && (
        <Group gap={24} fz="xs" lh={rem(20)} c="dimmed">
          {metaItems?.map(item => (
            <span key={item.key}>
              {item.label}
              {t('common.colon')}
              {item.value}
            </span>
          ))}
        </Group>
      )}
    </>
  )
}
