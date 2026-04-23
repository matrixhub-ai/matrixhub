import {
  type PolymorphicFactory,
  polymorphicFactory,
  useProps,
} from '@mantine/core'
import { IconPhotoUp } from '@tabler/icons-react'

import { BaseBadge, type BaseBadgeProps } from '@/shared/components/badges/BaseBadge'

import type { ReactNode } from 'react'

export interface TaskBadgeProps extends Omit<BaseBadgeProps, 'icon' | 'label'> {
  task: string
}

export type TaskBadgeFactory = PolymorphicFactory<{
  props: TaskBadgeProps
  defaultComponent: 'div'
  defaultRef: HTMLDivElement
  stylesNames: 'root' | 'label'
}>

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

export const TaskBadge = polymorphicFactory<TaskBadgeFactory>((_props, ref) => {
  const {
    task,
    ...badgeProps
  } = useProps('TaskBadge', {}, _props)

  return (
    <BaseBadge
      ref={ref}
      icon={resolveTaskIcon(task)}
      label={task}
      {...badgeProps}
    />
  )
})

TaskBadge.displayName = 'TaskBadge'
