import { Group } from '@mantine/core'
import { type Label } from '@matrixhub/api-ts/v1alpha1/model.pb.ts'

import { LibraryBadge } from '@/shared/components/badges/LibraryBadge'
import { BaseFilterPanel } from '@/shared/components/resource-filter-panel/BaseFilterPanel'
import { filterByKeyword } from '@/shared/utils'

interface LibraryFilterPanelProps {
  options: Label[]
  loading: boolean
  searchPlaceholder?: string
  selectedNames: string[]
  onSelect: (name: string) => void
}

export function LibraryFilterPanel({
  options,
  loading,
  searchPlaceholder = '',
  selectedNames,
  onSelect,
}: LibraryFilterPanelProps) {
  return (
    <BaseFilterPanel
      searchPlaceholder={searchPlaceholder}
      loading={loading}
    >
      {({ keyword }) => {
        const filteredOptions = filterByKeyword(options, keyword)

        if (!filteredOptions.length) {
          return null
        }

        return (
          <Group gap={8}>
            {
              filteredOptions.map(option => (
                <LibraryBadge
                  library={option.name as string}
                  key={option.id}
                  component="button"
                  type="button"
                  onClick={() => onSelect(option.name as string)}
                  styles={{
                    root: {
                      cursor: 'pointer',
                      borderColor: selectedNames.includes(option.name as string)
                        ? 'var(--mantine-primary-color-6)'
                        : undefined,
                    },
                  }}
                />
              ))
            }
          </Group>
        )
      }}
    </BaseFilterPanel>
  )
}
