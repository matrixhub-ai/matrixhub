import {
  Flex,
  type FlexProps,
  Group,
} from '@mantine/core'

import { SearchInput, type SearchInputProps } from '@/shared/components/SearchInput'

import classes from './SearchToolbar.module.css'

import type { ReactNode } from 'react'

export interface SearchToolbarProps {
  searchPlaceholder?: string
  searchValue?: SearchInputProps['value']
  onSearchChange?: SearchInputProps['onChange']
  toolbarProps?: Omit<FlexProps, 'children'>
  searchInputProps?: Omit<SearchInputProps, 'placeholder' | 'value' | 'onChange'>
  children?: ReactNode
}

export function SearchToolbar({
  searchPlaceholder,
  searchValue = '',
  onSearchChange,
  toolbarProps,
  searchInputProps,
  children,
}: SearchToolbarProps) {
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
        <SearchInput
          placeholder={searchPlaceholder as string}
          value={searchValue}
          onChange={onSearchChange}
          {...searchInputProps}
        />
      )}

      {children && (
        <Group className={classes.actions} gap="md" wrap="nowrap" ml="auto">
          {children}
        </Group>
      )}
    </Flex>
  )
}
