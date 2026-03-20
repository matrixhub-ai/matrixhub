import {
  Flex,
  type FlexProps,
  Group,
  TextInput,
  type TextInputProps,
} from '@mantine/core'
import { useDebouncedCallback } from '@mantine/hooks'
import { IconSearch as SearchIcon } from '@tabler/icons-react'
import {
  startTransition,
  useEffect,
  useRef,
} from 'react'

import type { ReactNode } from 'react'

export interface SearchToolbarProps {
  searchPlaceholder?: string
  searchValue?: string
  onSearchChange?: (value: string) => void
  toolbarProps?: Omit<FlexProps, 'children'>
  searchInputProps?: Omit<
    TextInputProps,
    | 'defaultValue'
    | 'value'
    | 'onChange'
    | 'placeholder'
    | 'leftSection'
    | 'styles'
  >
  children?: ReactNode
}

const DEFAULT_DEBOUNCE_MS = 300

export function SearchToolbar({
  searchPlaceholder,
  searchValue = '',
  onSearchChange,
  toolbarProps,
  searchInputProps,
  children,
}: SearchToolbarProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const {
    w: searchWidth,
    ...restSearchInputProps
  } = searchInputProps ?? {}

  const debouncedSearchChange = useDebouncedCallback((value: string) => {
    startTransition(() => {
      onSearchChange?.(value)
    })
  }, DEFAULT_DEBOUNCE_MS)

  useEffect(() => {
    debouncedSearchChange.cancel()

    const input = inputRef.current

    if (input && input.value !== searchValue) {
      input.value = searchValue
    }
  }, [searchValue, debouncedSearchChange])

  const showSearch = Boolean(searchPlaceholder && onSearchChange)

  return (
    <Flex
      justify="space-between"
      align="center"
      wrap="nowrap"
      gap="md"
      {...toolbarProps}
    >
      {showSearch && (
        <TextInput
          {...restSearchInputProps}
          defaultValue={searchValue}
          ref={inputRef}
          placeholder={searchPlaceholder}
          leftSection={(
            <SearchIcon
              size={16}
              style={{ color: 'var(--mantine-color-gray-5)' }}
            />
          )}
          onChange={(event) => {
            const nextQuery = event.currentTarget.value.trim()

            if (nextQuery === searchValue) {
              debouncedSearchChange.cancel()

              return
            }

            debouncedSearchChange(nextQuery)
          }}
          styles={{
            input: {
              height: 32,
              minHeight: 32,
              borderRadius: 16,
              fontSize: '14px',
              fontWeight: 400,
              lineHeight: '20px',
              color: 'var(--mantine-color-gray-8)',
              '&::placeholder': {
                color: 'var(--mantine-color-gray-5)',
                opacity: 1,
              },
            },
          }}
          w={searchWidth ?? 260}
        />
      )}

      {children && (
        <Group gap="md" wrap="nowrap" ml="auto">
          {children}
        </Group>
      )}
    </Flex>
  )
}
