import {
  Category, type Label, type Model,
} from '@matrixhub/api-ts/v1alpha1/model.pb'
import {
  IconClock, IconCube, IconApiApp,
} from '@tabler/icons-react'
import humanFormat from 'human-format'

import i18n from '@/i18n'
import { BaseBadge } from '@/shared/components/badges/BaseBadge'
import { LibraryBadge } from '@/shared/components/badges/LibraryBadge'
import { ParameterCountBadge } from '@/shared/components/badges/ParameterCountBadge'
import { TaskBadge } from '@/shared/components/badges/TaskBadge'
import { formatDateTime } from '@/shared/utils/date'
import { formatStorageSize } from '@/shared/utils/format'

import type { ResourceBadge, ResourceMetaItem } from '@/shared/components/resource-card/BaseCard'
import type { TFunction } from 'i18next'

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
  } = {
    taskCategory: Category.TASK,
    libraryCategory: Category.LIBRARY,
  },
): ResourceBadge[] {
  const badges: ResourceBadge[] = []
  const taskLabels = getLabelsByCategory(model.labels, options.taskCategory)
  const libraryLabels = getLabelsByCategory(model.labels, options.libraryCategory)

  for (const name of taskLabels) {
    badges.push({
      key: `task-${name}`,
      content: <TaskBadge task={name} maw={132} />,
    })
  }

  if (model.parameterCount) {
    badges.push({
      key: 'parameterCount',
      content: <ParameterCountBadge parameterCount={formatParameterCount(model.parameterCount)} maw={132} />,
    })
  }

  for (const name of libraryLabels) {
    badges.push({
      key: `library-${name}`,
      content: <LibraryBadge library={name} maw={132} />,
    })
  }

  const handledCategories = new Set([options.taskCategory, options.libraryCategory])
  const additionalLabels = (model.labels ?? []).filter(
    l => l.name && !handledCategories.has(l.category as Category),
  )

  for (const label of additionalLabels) {
    badges.push({
      key: `label-${label.category}-${label.name}`,
      content: <BaseBadge label={label.name} maw={132} />,
    })
  }

  return badges
}

export function buildModelMetaItems(
  model: Model,
  projectId: string,
  t: TFunction = i18n.t.bind(i18n),
  options?: {
    projectIcon?: ResourceMetaItem['icon']
    sizeIcon?: ResourceMetaItem['icon']
    updatedAtIcon?: ResourceMetaItem['icon']
  },
): ResourceMetaItem[] {
  return [
    {
      key: 'project',
      label: t('common.fromProject'),
      icon: options?.projectIcon ?? <IconApiApp size={20} />,
      value: model.project ?? projectId,
    },
    {
      key: 'size',
      label: t('common.modelSize'),
      icon: options?.sizeIcon ?? <IconCube size={20} />,
      value: formatStorageSize(model.size),
    },
    {
      key: 'updatedAt',
      label: t('common.updatedAt'),
      icon: options?.updatedAtIcon ?? <IconClock size={20} />,
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
