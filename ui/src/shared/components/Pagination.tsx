import {
  Group,
  Pagination as MantinePagination,
  type PaginationProps as MantinePaginationProps,
  Text,
  type TextProps,
} from '@mantine/core'
import { startTransition } from 'react'
import { useTranslation } from 'react-i18next'

import type { ReactNode } from 'react'

export interface PaginationProps {
  total: number
  totalPages: number
  page: number
  onPageChange: (page: number) => void
  totalLabel?: ReactNode
  totalLabelProps?: TextProps
  paginationProps?: Omit<
    MantinePaginationProps,
    'total' | 'value' | 'onChange'
  >
}

export function Pagination({
  total,
  totalPages,
  page,
  onPageChange,
  totalLabel,
  totalLabelProps,
  paginationProps,
}: PaginationProps) {
  const { t } = useTranslation()

  if (total <= 0) {
    return null
  }

  const totalLabelDisplay = totalLabel ?? t('shared.total', { count: total })

  return (
    <Group justify="space-between" px="sm" py="xs">
      <Text size="sm" fw={600} c="gray.6" lh="sm" {...totalLabelProps}>
        {totalLabelDisplay}
      </Text>
      <MantinePagination
        {...paginationProps}
        value={page}
        onChange={(nextPage) => {
          startTransition(() => {
            onPageChange(nextPage)
          })
        }}
        total={totalPages}
      />
    </Group>
  )
}
