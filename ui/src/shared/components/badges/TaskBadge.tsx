import { IconPhotoUp } from '@tabler/icons-react'

import { BaseBadge } from '@/shared/components/badges/BaseBadge'

import type { ComponentProps, ReactNode } from 'react'

type BaseBadgeProps = ComponentProps<typeof BaseBadge>

interface TaskBadgeProps extends Omit<BaseBadgeProps, 'icon' | 'label'> {
  task: string
}

const TASK_ICON_ENTRIES = [
  {
    matcher: /(image|vision|photo)/i,
    icon: <IconPhotoUp size={16} color="var(--mantine-color-blue-4)" />,
  },
] as const

function resolveTaskIcon(task: string): ReactNode {
  const matched = TASK_ICON_ENTRIES.find(item => item.matcher.test(task))

  return matched?.icon ?? null
}

export function TaskBadge({
  task,
  ...badgeProps
}: TaskBadgeProps) {
  return (
    <BaseBadge
      icon={resolveTaskIcon(task)}
      label={task}
      {...badgeProps}
    />
  )
}
