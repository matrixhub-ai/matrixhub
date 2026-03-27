import {
  SimpleGrid,
  Skeleton,
} from '@mantine/core'
import { useMemo } from 'react'

import { EmptyStatePrompt } from '@/shared/components/EmptyStatePrompt'

import type { ReactNode } from 'react'

export interface ResourceCardGridProps {
  loading?: boolean
  skeletonCount?: number
  skeletonHeight?: number
  emptyTitle?: ReactNode
  emptyHelper?: ReactNode
  children: ReactNode[]
}

const DEFAULT_SKELETON_COUNT = 6
const DEFAULT_SKELETON_HEIGHT = 116

export function ResourceCardGrid({
  loading,
  skeletonCount = DEFAULT_SKELETON_COUNT,
  skeletonHeight = DEFAULT_SKELETON_HEIGHT,
  emptyTitle,
  emptyHelper,
  children,
}: ResourceCardGridProps) {
  const skeletonKeys = useMemo(
    () => Array.from(
      { length: skeletonCount },
      (_, i) => `skeleton-${i + 1}`,
    ),
    [skeletonCount],
  )

  const isEmpty = !children?.length && !loading

  return (
    <SimpleGrid
      cols={{
        base: 1,
        md: isEmpty ? 1 : 2,
      }}
      spacing={20}
    >
      {loading
        ? skeletonKeys.map(key => (
            <Skeleton key={key} h={skeletonHeight} radius="md" />
          ))
        : children}

      {isEmpty && (
        <EmptyStatePrompt
          title={emptyTitle}
          helper={emptyHelper}
        />
      )}
    </SimpleGrid>
  )
}
