import { TextInput, type TextInputProps } from '@mantine/core'
import { useDebouncedCallback } from '@mantine/hooks'
import { IconSearch } from '@tabler/icons-react'
import {
  startTransition,
  useEffect,
  useState,
} from 'react'

export interface SearchInputProps extends Omit<TextInputProps, 'value' | 'onChange'> {
  value?: string
  onChange?: (value: string) => void
  debounceMs?: number
}

const DEFAULT_DEBOUNCE_MS = 300

export function SearchInput({
  value = '',
  onChange,
  debounceMs = DEFAULT_DEBOUNCE_MS,
  placeholder,
  w = 260,
  styles,
  ...restTextInputProps
}: SearchInputProps) {
  const [inputValue, setInputValue] = useState(value)

  const resolvedStyles = typeof styles === 'function'
    ? styles
    : {
        ...styles,
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
          ...styles?.input,
        },
      }

  const debouncedSearchChange = useDebouncedCallback((value: string) => {
    startTransition(() => {
      onChange?.(value)
    })
  }, debounceMs)

  useEffect(() => {
    return () => {
      debouncedSearchChange.cancel()
    }
  }, [debouncedSearchChange])

  return (
    <TextInput
      {...restTextInputProps}
      value={inputValue}
      placeholder={placeholder}
      leftSection={(
        <IconSearch
          size={16}
          color="var(--mantine-color-gray-5)"
        />
      )}
      onChange={(event) => {
        const nextQuery = event.currentTarget.value.trim()

        setInputValue(nextQuery)

        if (nextQuery === value) {
          debouncedSearchChange.cancel()

          return
        }

        debouncedSearchChange(nextQuery)
      }}
      styles={resolvedStyles}
      w={w}
    />
  )
}
