import {
  SimpleGrid,
  Skeleton,
} from '@mantine/core'
import { useMemo } from 'react'

import type { ReactNode } from 'react'

export interface ResourceCardGridProps {
  loading?: boolean
  skeletonCount?: number
  skeletonHeight?: number
  children: ReactNode
}

const DEFAULT_SKELETON_COUNT = 6
const DEFAULT_SKELETON_HEIGHT = 116

export function ResourceCardGrid({
  loading,
  skeletonCount = DEFAULT_SKELETON_COUNT,
  skeletonHeight = DEFAULT_SKELETON_HEIGHT,
  children,
}: ResourceCardGridProps) {
  const skeletonKeys = useMemo(
    () => Array.from(
      { length: skeletonCount },
      (_, i) => `skeleton-${i + 1}`,
    ),
    [skeletonCount],
  )

  return (
    <SimpleGrid
      cols={{
        base: 1,
        md: 2,
      }}
      spacing={20}
    >
      {loading
        ? skeletonKeys.map(key => (
            <Skeleton key={key} h={skeletonHeight} radius="md" />
          ))
        : children}
    </SimpleGrid>
  )
}
