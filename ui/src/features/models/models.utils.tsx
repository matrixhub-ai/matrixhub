import {
  IconClock as ClockIcon,
  IconCube as ModelIcon,
  IconApiApp as ProjectIcon,
} from '@tabler/icons-react'
import { filesize } from 'filesize'
import humanFormat from 'human-format'

import { formatDateTime } from '@/shared/utils/date'

import type {
  ResourceCardBadge,
  ResourceCardMetaItem,
} from '@/shared/components/ResourceCard'
import type {
  Category,
  Label,
  Model,
} from '@matrixhub/api-ts/v1alpha1/model.pb'

export function buildModelTitle(model: Model, projectId: string) {
  const projectName = model.project ?? projectId
  const modelName = model.name?.trim()

  return `${projectName} / ${modelName || '-'}`
}

export function buildModelBadges(
  model: Model,
  options: {
    taskCategory: Category
    libraryCategory: Category
    taskIcon: ResourceCardBadge['icon']
    libraryIconFn: (name: string) => ResourceCardBadge['icon']
    parameterCountIcon: ResourceCardBadge['icon']
  },
): ResourceCardBadge[] {
  const badges: ResourceCardBadge[] = []
  const taskLabels = getLabelsByCategory(model.labels, options.taskCategory)
  const libraryLabels = getLabelsByCategory(model.labels, options.libraryCategory)

  for (const name of taskLabels) {
    badges.push({
      key: `task-${name}`,
      icon: options.taskIcon,
      label: name,
    })
  }

  for (const name of libraryLabels) {
    badges.push({
      key: `library-${name}`,
      icon: options.libraryIconFn(name),
      label: name,
    })
  }

  if (model.parameterCount) {
    badges.push({
      key: 'parameterCount',
      icon: options.parameterCountIcon,
      label: formatParameterCount(model.parameterCount),
    })
  }

  return badges
}

export function buildModelMetaItems(
  model: Model,
  projectId: string,
  options?: {
    projectIcon?: ResourceCardMetaItem['icon']
    sizeIcon?: ResourceCardMetaItem['icon']
    updatedAtIcon?: ResourceCardMetaItem['icon']
  },
): ResourceCardMetaItem[] {
  return [
    {
      key: 'project',
      icon: options?.projectIcon ?? <ProjectIcon size={20} />,
      value: model.project ?? projectId,
    },
    {
      key: 'size',
      icon: options?.sizeIcon ?? <ModelIcon size={20} />,
      value: formatStorageSize(model.size),
    },
    {
      key: 'updatedAt',
      icon: options?.updatedAtIcon ?? <ClockIcon size={20} />,
      value: formatDateTime(model.updatedAt),
    },
  ]
}

export function getLabelsByCategory(labels: Label[] | undefined, category: Category) {
  return (labels ?? [])
    .filter(label => label.category === category && !!label.name)
    .map(label => label.name as string)
}

const parameterCountScale = new humanFormat.Scale({
  '': 1,
  K: 1_000,
  M: 1_000_000,
  B: 1_000_000_000,
  T: 1_000_000_000_000,
})

export function formatParameterCount(value: string | undefined) {
  if (!value) {
    return '-'
  }

  const numericValue = Number(value)

  if (!Number.isFinite(numericValue)) {
    return value
  }

  return humanFormat(numericValue, {
    scale: parameterCountScale,
    decimals: 1,
  })
}

export function formatStorageSize(value: string | undefined) {
  if (!value) {
    return '-'
  }

  const numericValue = Number(value)

  if (!Number.isFinite(numericValue)) {
    return value
  }

  return filesize(numericValue, {
    standard: 'jedec',
    round: 1,
  }) as string
}
